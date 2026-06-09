// govor is a Go port of „Говореща програма“ — a Bulgarian text-to-speech
// program for the Apple II / Правец-82 by Борислав Захариев, recovered from
// an original DOS 3.3 disk image (see original/GOVOR.dsk and docs/).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	out := flag.String("o", "govor.wav", "output WAV file ('-' for stdout)")
	speed := flag.Int("speed", 5, "speech speed 1-9 (1 fastest/highest, 9 slowest/lowest), as the original &1..&9 commands")
	rate := flag.Int("rate", 44100, "output sample rate in Hz")
	play := flag.Bool("play", false, "play the result through a system audio player")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `govor — Bulgarian text-to-speech (Apple II / Правец-82, 1980s) ported to Go

Usage: govor [flags] [text ...]

Text is read from the arguments, or from standard input if none are given.
Recognized input, exactly as in the original:

  а-я А-Я    spoken with the original sampled sounds
  space      word pause; ',' is a double pause
  0-9        spelled out (нула, едно, две, ...)
  + - = * .  spelled out (плюс, минус, равно, хахаха, точка)
  / :        raise/lower the pitch of the next letter (intonation)
  _          short pause

Flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	var text string
	if flag.NArg() > 0 {
		text = strings.Join(flag.Args(), " ")
	} else {
		var sb strings.Builder
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			sb.WriteString(sc.Text())
			sb.WriteByte('\n')
		}
		text = sb.String()
	}
	if strings.TrimSpace(text) == "" {
		flag.Usage()
		os.Exit(2)
	}

	s := NewSynth(*rate, *speed)
	s.Speak(text)
	samples := s.Render()

	if *out == "-" {
		if err := writeWAV(os.Stdout, *rate, samples); err != nil {
			fatal(err)
		}
		return
	}
	f, err := os.Create(*out)
	if err != nil {
		fatal(err)
	}
	if err := writeWAV(f, *rate, samples); err != nil {
		fatal(err)
	}
	if err := f.Close(); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%.2fs)\n", *out, float64(len(samples))/float64(*rate))

	if *play {
		if err := playFile(*out); err != nil {
			fatal(err)
		}
	}
}

func playFile(path string) error {
	for _, p := range []string{"afplay", "paplay", "aplay", "ffplay"} {
		if _, err := exec.LookPath(p); err != nil {
			continue
		}
		args := []string{path}
		if p == "ffplay" {
			args = []string{"-nodisp", "-autoexit", "-loglevel", "quiet", path}
		}
		cmd := exec.Command(p, args...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("no audio player found (tried afplay, paplay, aplay, ffplay)")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "govor:", err)
	os.Exit(1)
}
