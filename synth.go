package main

import (
	"math"
	"unicode"
)

// The synthesizer is a faithful timing model of the 6502 playback routine at
// $09AD in the original program. That routine reads a stream of delay values
// and toggles the Apple II speaker ($C030) after each one; the waveform is
// therefore a square wave whose edge spacing is the data. Two encodings
// exist:
//
//   - byte mode: each data byte is one delay (1-255 units; 0 means a 256-unit
//     silent gap with no toggle). Used for voiced sounds (vowels, л, м, н...).
//   - nibble mode (segment starts with $FF): each byte packs two delays of
//     1-15 units, high nibble first. Used for the unvoiced fricatives and
//     stops (с, ш, ф, х, ц, ч, т, к), which need fast, noisy toggling.
//
// The first byte of every segment (after the optional $FF) is a repeat count:
// the segment body is played that many times, which is how the longer voiced
// sounds (ж, з) get their length.
//
// One delay unit is the inner loop at $09F0: LDX #speed / DEX / BNE / DEY /
// BNE = 5*speed+6 CPU cycles. The "speed" immediate operand is self-modified
// by the program: digit commands set its default (1-9 -> 4-12, default 8) and
// the in-text characters '/' and ':' decrement/increment it for the next
// letter only.

const cpuHz = 1020484.0 // effective Apple II / Pravetz 82 CPU clock

type segment struct {
	start, end uint16 // offsets into soundData, [start, end)
}

// Synth accumulates speaker toggle events measured in CPU cycles and renders
// them to PCM samples.
type Synth struct {
	SampleRate int

	defaultSpeed int // operand reloaded into the delay loop after every letter
	speed        int // current delay-loop operand ($09F1)

	cycles  uint64   // running CPU cycle count
	toggles []uint64 // cycle timestamps of speaker toggles
}

// NewSynth creates a synthesizer. speed is the original program's 1-9 scale
// (the &1...&9 commands); 5 is the default.
func NewSynth(sampleRate, speed int) *Synth {
	if speed < 1 {
		speed = 1
	}
	if speed > 9 {
		speed = 9
	}
	s := &Synth{SampleRate: sampleRate}
	// The digit command at $082C computes char - $2D: '1'..'9' -> 4..12.
	s.defaultSpeed = speed + 3
	s.speed = s.defaultSpeed
	return s
}

// unit is the duration in cycles of one delay count (one DEY iteration of the
// loop at $09F0, including its inner LDX #speed countdown).
func (s *Synth) unit() uint64 {
	return uint64(5*s.speed + 6)
}

// delay models the delay subroutine for a count n (a stored value of 0 means
// 256). The final BNE falls through, costing one cycle less.
func (s *Synth) delay(n int) {
	if n == 0 {
		n = 256
	}
	s.cycles += uint64(n)*s.unit() - 1
}

// toggle records a speaker flip (the STX $C030 / STX $C020 pair).
func (s *Synth) toggle() {
	s.cycles += 4
	s.toggles = append(s.toggles, s.cycles)
	s.cycles += 4
}

// top models the pointer-vs-end comparison at $09CF. When only the low bytes
// match, the high-byte compare runs too.
func (s *Synth) top(ptr, end uint16) {
	if byte(ptr) == byte(end) {
		s.cycles += 16
	} else {
		s.cycles += 9
	}
}

// step advances the data pointer, modelling INC $06 / BNE (+ INC $07 on page
// crossings).
func (s *Synth) step(ptr uint16) uint16 {
	ptr++
	if byte(ptr) == 0 {
		s.cycles += 15 // INC, BNE not taken, INC hi, BNE
	} else {
		s.cycles += 8 // INC, BNE taken
	}
	return ptr
}

// playSegment models one call of the routine at $09AD.
func (s *Synth) playSegment(seg segment) {
	ptr, end := seg.start, seg.end
	if ptr >= end || int(end) > len(soundData) {
		return
	}

	nibbleMode := false
	s.cycles += 21 // entry preamble through the $FF check
	if soundData[ptr] == 0xFF {
		nibbleMode = true
		ptr++
		s.cycles += 11
	} else {
		s.cycles += 3
	}
	repeat := int(soundData[ptr])
	ptr++
	s.cycles += 25 // read repeat count, save restart pointer
	if repeat == 0 {
		repeat = 256 // DEC wraps; never present in the shipped data
	}

	body := ptr
	for {
		for ptr != end {
			s.top(ptr, end)
			v := soundData[ptr]
			if v == 0 {
				// LDA, BNE not taken, shift high nibble out, BEQ taken
				s.cycles += 25
				s.delay(0) // 256 silent units, no toggle
				s.cycles += 6
			} else if nibbleMode {
				s.cycles += 23 // LDA, BNE, STA $1B, AND, 4xLSR, TAY
				if hi := int(v >> 4); hi != 0 {
					s.cycles += 2
					s.toggle()
					s.delay(hi)
				} else {
					s.cycles += 3
					s.delay(0)
				}
				s.cycles += 5 + 5 // LDA $1B, BEQ not taken, STY, AND
				if lo := int(v & 0x0F); lo != 0 {
					s.cycles += 7 // BNE, TAY, BEQ not taken
					s.toggle()
					s.delay(lo)
					s.cycles += 6 // LDA $1B (now 0), BEQ taken
				} else {
					s.cycles += 2
				}
			} else {
				s.cycles += 12 // LDA, BNE to TAY, BEQ not taken
				s.toggle()
				s.delay(int(v))
				s.cycles += 6 // LDA $1B (0), BEQ taken
			}
			ptr = s.step(ptr)
		}
		s.cycles += 17 // end-of-segment compare
		repeat--
		if repeat == 0 {
			s.cycles += 14 // DEC, BEQ, RTS
			return
		}
		s.cycles += 22 // DEC, reload pointer, JMP $09DB
		ptr = body
		// The restart jumps straight to the byte fetch, so play one byte
		// without the top-of-loop compare: emulate by rewinding its cost.
		s.cycles -= 9
	}
}

// pause models the inter-word pause for ' ': the loop at $0870 calls the
// monitor WAIT routine ($FCA8) with arguments 8*speed, 8*speed, 8*speed-1,
// ..., 2. WAIT(a) burns (26 + 27a + 5a^2)/2 cycles.
func (s *Synth) pause() {
	wait := func(a int) uint64 {
		return uint64(26+27*a+5*a*a) / 2
	}
	n := 8 * s.speed
	s.cycles += 13 + wait(n) // LDA/ASL/TAX preamble plus the first call
	for a := n; a >= 2; a-- {
		s.cycles += wait(a) + 13 // JSR/TXA/DEX/BNE around each call
	}
}

// playCode speaks a single Pravetz 82 character code, modelling $086C.
func (s *Synth) playCode(code byte) {
	switch {
	case code == 0x20:
		s.pause()
		return
	case code == '/': // raise pitch of the next letter
		if s.speed > 1 {
			s.speed--
		}
		return
	case code == ':': // lower pitch of the next letter
		s.speed++
		return
	}
	seg, ok := segments[code]
	if !ok {
		return
	}
	s.cycles += 80 // dispatch and table-load overhead in $086C-$08C5
	s.playSegment(seg)
	s.speed = s.defaultSpeed // LDX #default / STX $09F1 after every letter
}

// letterCodes maps Bulgarian Cyrillic letters to the Pravetz 82 character
// codes the original program indexes its table with.
var letterCodes = map[rune]byte{
	'а': 0x61, 'б': 0x62, 'в': 0x77, 'г': 0x67, 'д': 0x64, 'е': 0x65,
	'ж': 0x76, 'з': 0x7A, 'и': 0x69, 'й': 0x6A, 'к': 0x6B, 'л': 0x6C,
	'м': 0x6D, 'н': 0x6E, 'о': 0x6F, 'п': 0x70, 'р': 0x72, 'с': 0x73,
	'т': 0x74, 'у': 0x75, 'ф': 0x66, 'х': 0x68, 'ц': 0x63, 'ч': 0x5E,
	'ш': 0x5B, 'щ': 0x5D, 'ъ': 0x79, 'ь': 0x78, 'ю': 0x60, 'я': 0x71,
}

// Speak processes a text the way the original handles a quoted string:
// Cyrillic letters play their sound segments, spaces pause, digits and the
// symbols * + , - . = are spelled out as words, '/' and ':' tweak the pitch
// of the following letter, and everything else is ignored.
func (s *Synth) Speak(text string) {
	for _, r := range text {
		switch r {
		case ' ', '\t', '\n', '\r':
			s.playCode(0x20)
			continue
		case '/', ':':
			s.playCode(byte(r))
			continue
		case '_':
			s.playCode(0x5F)
			continue
		case '=': // the original remaps '=' to the "равно" slot
			r = '/'
		case 'ѝ': // stressed и, common in Bulgarian text
			r = 'и'
		}
		if word, ok := symbolWords[r]; ok {
			for _, c := range word {
				s.playCode(c)
			}
			continue
		}
		lower := unicode.ToLower(r)
		if code, ok := letterCodes[lower]; ok {
			s.playCode(code)
		}
		// Anything else: ignored, as in the original.
	}
}

// Render converts the recorded toggle events into 16-bit PCM samples. The
// speaker is a two-state device; a first-order high-pass filter removes the
// DC offset the raw square wave would otherwise carry, which also roughly
// matches how the real speaker cone relaxes during silence.
func (s *Synth) Render() []int16 {
	// quarter second of lead-in/lead-out silence
	pad := uint64(cpuHz / 4)
	total := s.cycles + 2*pad
	n := int(float64(total) / cpuHz * float64(s.SampleRate))

	out := make([]int16, n)
	level := 0.5
	cyclesPerSample := cpuHz / float64(s.SampleRate)

	// ~20 Hz DC-blocking high-pass
	rc := 1.0 / (2 * math.Pi * 20.0)
	dt := 1.0 / float64(s.SampleRate)
	alpha := rc / (rc + dt)

	ti := 0
	prev, hp := level, 0.0
	for i := 0; i < n; i++ {
		c := uint64(float64(i) * cyclesPerSample)
		for ti < len(s.toggles) && s.toggles[ti]+pad <= c {
			level = -level
			ti++
		}
		hp = alpha * (hp + level - prev)
		prev = level
		v := hp * 0.8
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		out[i] = int16(v * 32767)
	}
	return out
}
