# govor

Говореща програма от Борислав Захариев — a 1980s Bulgarian text-to-speech
program for the Apple II / Правец-82, recovered from an original DOS 3.3 disk
image and ported to Go.

The original is ~6 KB of 6502 machine code hiding behind a one-line Applesoft
BASIC program (`0 CALL 2347`). It speaks through the Apple II's 1-bit speaker
by replaying stored delay streams between speaker clicks — the Go port models
that playback loop cycle-for-cycle at the original 1.02 MHz clock and renders
the square wave to a WAV file, using the exact waveform data extracted from
the disk. See [docs/REVERSING.md](docs/REVERSING.md) for the full
reverse-engineering story.

## Usage

```sh
go build .
./govor "здравей свят"                  # writes govor.wav
./govor -play "проба едно две три"      # also play it (afplay/aplay/...)
./govor -speed 9 -o slow.wav "бавно"    # speed 1 (fast/high) … 9 (slow/low)
echo "от диск към говор" | ./govor      # text from stdin
```

The input handling matches the original program:

| Input         | Effect |
|---------------|--------|
| а-я, А-Я      | spoken with the original sampled sounds |
| space         | word pause, `,` is a double pause, `_` a short one |
| `0`-`9`       | spelled out: нула, едно, две, … девет |
| `+ - = .`     | плюс, минус, равно, точка |
| `*`           | хахаха |
| `/` `:`       | raise / lower the pitch of the next letter (intonation) |
| anything else | ignored |

## Repository layout

* `main.go`, `synth.go`, `wav.go` — the Go port (no dependencies)
* `data.go` — waveform data and tables, generated from the disk image by
  `tools/extract.py`
* `original/GOVOR.dsk` — the Apple DOS 3.3 disk image, with extracted BASIC
  listings
* `docs/` — reverse-engineering notes and the 6502 disassembly
* `tools/` — the extractor and the small 6502 disassembler used for the port
