package main

import "testing"

func TestSegmentsCoverAlphabet(t *testing.T) {
	for r, code := range letterCodes {
		seg, ok := segments[code]
		if !ok {
			t.Fatalf("letter %c (code %02X) has no segment", r, code)
		}
		if seg.start >= seg.end || int(seg.end) > len(soundData) {
			t.Errorf("letter %c: bad segment [%04X, %04X)", r, seg.start, seg.end)
		}
	}
}

func TestNibbleSegmentsStartWithFF(t *testing.T) {
	// The unvoiced sounds use the packed nibble encoding, marked by $FF.
	for _, r := range "сшщфхцчтк" {
		seg := segments[letterCodes[r]]
		if soundData[seg.start] != 0xFF {
			t.Errorf("%c: expected nibble-mode marker, got %02X", r, soundData[seg.start])
		}
	}
}

func TestSpeakProducesAudio(t *testing.T) {
	s := NewSynth(44100, 5)
	s.Speak("проба едно две три")
	if len(s.toggles) == 0 {
		t.Fatal("no speaker toggles generated")
	}
	sec := float64(s.cycles) / cpuHz
	if sec < 1 || sec > 10 {
		t.Errorf("unexpected duration %.2fs", sec)
	}
	samples := s.Render()
	peak := int16(0)
	for _, v := range samples {
		if v > peak {
			peak = v
		}
	}
	if peak < 8000 {
		t.Errorf("output too quiet, peak %d", peak)
	}
}

func TestSpeedChangesDuration(t *testing.T) {
	dur := func(speed int) uint64 {
		s := NewSynth(44100, speed)
		s.Speak("а")
		return s.cycles
	}
	if !(dur(1) < dur(5) && dur(5) < dur(9)) {
		t.Errorf("durations not monotonic: %d %d %d", dur(1), dur(5), dur(9))
	}
	// One delay unit is 5*speed+6 cycles, so durations scale roughly with it.
	ratio := float64(dur(9)) / float64(dur(1))
	if ratio < 2.0 || ratio > 3.0 {
		t.Errorf("speed 9 / speed 1 duration ratio %.2f, want ~2.5", ratio)
	}
}

func TestIntonationMarks(t *testing.T) {
	dur := func(text string) uint64 {
		s := NewSynth(44100, 5)
		s.Speak(text)
		return s.cycles
	}
	if !(dur("/а") < dur("а") && dur("а") < dur(":а")) {
		t.Error("'/' should shorten (raise) and ':' lengthen (lower) the next letter")
	}
	// The effect resets after one letter, as in the original.
	if d, want := dur("/аа")-dur("/а"), dur("а"); d != want {
		t.Errorf("second letter after '/' lasted %d cycles, want %d", d, want)
	}
}

func TestUnknownRunesIgnored(t *testing.T) {
	s := NewSynth(44100, 5)
	s.Speak("AZ?!;()xyz")
	if len(s.toggles) != 0 {
		t.Error("non-Cyrillic letters should be silent")
	}
}
