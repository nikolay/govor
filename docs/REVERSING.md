# Reverse-engineering notes: GOVOR.dsk

`original/GOVOR.dsk` is a 140 KB Apple DOS 3.3 disk image (35 tracks × 16
sectors × 256 bytes) of „Говореща програма“ — a Bulgarian text-to-speech
program for the Apple II / Правец-82 by **Борислав Захариев** (credited on the
program's help screen: „говореща програма от Борислав Захариев“).

## Disk contents

The DOS 3.3 catalog (track 17) lists two Applesoft BASIC files:

| File    | Size       | Contents |
|---------|------------|----------|
| `HELLO` | 23 bytes   | boot program: `10 PRINT CHR$(4);"RUNgowor"` |
| `gowor` | 6147 bytes | one BASIC line `0 CALL 2347` + ~6 KB of 6502 machine code |

`gowor` is a classic hybrid: Applesoft loads it at `$0801`, the single line
`CALL 2347` jumps into the machine code at `$092B` that rides along after the
tokenized line. Extracted listings are in `original/*.bas.txt`; a disassembly
of the code portion is in `gowor-disassembly.txt`.

## Memory map of `gowor` (loaded at $0801)

| Range           | Contents |
|-----------------|----------|
| `$0801`–`$080A` | BASIC stub `0 CALL 2347` |
| `$0810`–`$092A` | `&` argument parser, per-character dispatch, variable lookup |
| `$092B`–`$09AC` | startup: hooks the Applesoft `&` vector (`$03F5`) to `$0A1B`; cassette save/restore helpers |
| `$09AD`–`$0A1A` | **the sound player** (see below) |
| `$0A1B`–`$0AEF` | `&` command handler (`D`, `I`, `V`, `H`, `S`, `C`, `SAVE`), screen echo |
| `$0AF0`–`$0B6F` | spoken words for `* + , - . /` and digits `0`–`9`, 8 bytes/slot |
| `$0B70`–`$0C03` | letter table: 4 bytes per char code `$5B`–`$7F` = little-endian (start, end) of the sound segment |
| `$0C00`–`$1DFF` | **waveform data** (speaker delay streams) |
| `$1E01`–`$1FFE` | help screen (`&I`), title screen, cassette-save UI |

## The sound player ($09AD)

The Apple II has a 1-bit speaker: any read/write of `$C030` flips the cone.
The player walks a byte stream where every value is a *delay count*; after
each delay it toggles the speaker (`STX $C030`, plus `STX $C020` for the
cassette out). Speech is therefore stored as the time intervals between
square-wave edges.

Two encodings:

* **byte mode** — each byte is one delay of 1–255 units; `0` means a silent
  256-unit gap with no toggle. Used for voiced sounds (vowels, б, д, г, л,
  м, н, р, в, ж, з, ъ...).
* **nibble mode** — the segment starts with an `$FF` marker; each following
  byte packs two delays of 1–15 units (high nibble first, `0` = silent gap).
  Used for unvoiced noise (с, ш, щ, ф, х, ц, ч, т, к), which needs edge
  rates up to ~10 kHz.

The first byte of each segment (after the optional `$FF`) is a **repeat
count**: the body plays that many times (ж and з use 2; everything else 1).

One delay unit is the loop at `$09F0`:

```asm
09F0: LDX #$08      ; operand = "speed", self-modified
09F2: DEX
09F3: BNE $09F2
09F5: DEY
09F6: BNE $09F0     ; Y = delay count
```

i.e. `5*speed + 6` CPU cycles per unit (≈45 µs at the default speed 8).

## Speed and intonation

The speed operand at `$09F1` is self-modified from three places:

* a digit `1`–`9` in the `&` argument sets the default: `char - $2D` → 4–12
  (so `&5` is the default 8);
* `/` in spoken text **decrements** it (higher pitch, faster) and `:`
  **increments** it (lower, slower) — for the *next letter only*, because
* after every letter the dispatch code reloads the default (`LDX #def`,
  `STX $09F1` at `$08C8`).

A space pauses by calling the monitor `WAIT` (`$FCA8`) in a loop at `$0870`
with arguments `8*speed, 8*speed, 8*speed-1, …, 2`; `WAIT(A)` burns
`(26 + 27·A + 5·A²)/2` cycles, ≈260 ms total at default speed.

## Character set

Text uses the Правец-82 (KOI-7 N2 / МИК) character codes, where `$60`–`$7A`
display as Cyrillic. The letter table covers codes `$5B`–`$7F`:

```
$5B ш  $5D щ  $5E ч  $5F _ (short pause)
$60 ю  $61 а  $62 б  $63 ц  $64 д  $65 е  $66 ф  $67 г  $68 х  $69 и
$6A й  $6B к  $6C л  $6D м  $6E н  $6F о  $70 п  $71 я  $72 р  $73 с
$74 т  $75 у  $76 ж  $77 в  $78 ь  $79 ъ  $7A з
```

Three economies in the table are worth noticing:

* **щ** points at ш's data extended through т's — Bulgarian щ = «шт»;
* **ю** = й's segment extended into a у-like tail, **я** = a glide + а's data
  (я ends exactly where а ends);
* **ь** simply reuses й (correct for Bulgarian, where ь only marks the /j/
  glide in -ьо).

The `$5C` (э) entry has start = end and would run off the end of its data —
a latent bug that Bulgarian text never triggers.

Digits and arithmetic symbols are spelled out as words from the `$0AF0`
table: нула…девет, плюс, минус, равно (`=`), точка (`.`), and `*` speaks
„хахаха“. `,` is a double pause.

## The & interface

The handler hooked at `$03F5` implements, per the built-in `&I` help screen:

```
&I          informaciq            (this help)
&4          skorost 4 (ot 1 do 9) (speed)
&C          ^isti ekrana          (clear screen)
&A$  &"  "  izgowarq stringa      (speak a variable / literal)
&S,A$       pokazwa go            (display it)
&5V9,H6,A$  skorost 5, red 9, 6a poziciq  (speed/VTAB/HTAB)
" / : "     promenq intonaciqta   (intonation)
&D          direkten revim        (speak keys as typed)
&SAVE       zapis na kaseta       (cassette save)
```

## What the Go port keeps and drops

Kept: the exact waveform data, the segment table, the delay/toggle timing
model (cycle-accurate, including the WAIT pauses), the repeat counts, speed
1–9, the `/` `:` intonation marks, the digit/symbol words, and the
unknown-character behaviour. The square wave is rendered at the effective
Apple II clock (1.0205 MHz) and passed through a 20 Hz DC-blocking filter in
place of the physical speaker.

Dropped: the Apple-specific shell — `&` vector, screen echo, VTAB/HTAB,
cassette save, and the self-relocation helpers.
