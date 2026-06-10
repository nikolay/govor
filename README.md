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

**Try it in the browser at [govor.app](https://govor.app)** — the same Go
code compiled to WebAssembly, with playback and WAV download.

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

## The website (govor.app)

`web/` is a static page that loads the synthesizer compiled to WebAssembly
(`GOOS=js GOARCH=wasm`, see `wasm.go`) and lets you type text, hear it, and
download the WAV. `.github/workflows/deploy.yml` builds it and deploys to
GitHub Pages on every push to `main`.

To run it locally:

```sh
GOOS=js GOARCH=wasm go build -o web/govor.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
python3 -m http.server -d web 8000    # open http://localhost:8000
```

One-time hosting setup (repository **Settings → Pages**):

1. Set **Source** to “GitHub Actions”.
2. Enter `govor.app` as the **Custom domain** and enable **Enforce HTTPS**
   (the `web/CNAME` file matches it).
3. At your DNS provider, point the apex `govor.app` at GitHub Pages:
   * `A` records: `185.199.108.153`, `185.199.109.153`, `185.199.110.153`,
     `185.199.111.153`
   * `AAAA` records: `2606:50c0:8000::153`, `2606:50c0:8001::153`,
     `2606:50c0:8002::153`, `2606:50c0:8003::153`
   * optionally `www` as a `CNAME` to `nikolay.github.io`

## Repository layout

* `main.go`, `synth.go`, `wav.go` — the Go port (no dependencies)
* `wasm.go` — the WebAssembly entry point used by the website
* `data.go` — waveform data and tables, generated from the disk image by
  `tools/extract.py`
* `web/` — the [govor.app](https://govor.app) page (HTML/CSS/JS, WASM built
  in CI)
* `original/GOVOR.dsk` — the Apple DOS 3.3 disk image, with extracted BASIC
  listings
* `docs/` — reverse-engineering notes and the 6502 disassembly
* `tools/` — the extractor and the small 6502 disassembler used for the port
