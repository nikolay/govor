//go:build js && wasm

// WebAssembly entry point for the govor.app web page. It exposes a single
// global function to JavaScript:
//
//	govorSynth(text string, speed int, rate int) Uint8Array
//
// which returns a complete WAV file for the given text, speed 1-9 and
// sample rate in Hz.
package main

import (
	"bytes"
	"syscall/js"
)

func main() {
	js.Global().Set("govorSynth", js.FuncOf(func(this js.Value, args []js.Value) any {
		// This is a global JS API: validate the call instead of panicking,
		// which would kill the WASM instance for the rest of the page's life.
		if len(args) < 3 || args[0].Type() != js.TypeString ||
			args[1].Type() != js.TypeNumber || args[2].Type() != js.TypeNumber {
			return js.Null()
		}
		text := args[0].String()
		speed := args[1].Int()
		rate := args[2].Int()
		// Clamp to the range real audio hardware uses; an absurd rate would
		// just be a giant allocation that kills the WASM instance.
		if rate < 8000 || rate > 192000 {
			rate = 44100
		}
		s := NewSynth(rate, speed)
		s.Speak(text)
		samples := s.Render()
		var buf bytes.Buffer
		if err := writeWAV(&buf, rate, samples); err != nil {
			// bytes.Buffer writes cannot fail; keep the JS contract simple.
			return js.Null()
		}
		out := js.Global().Get("Uint8Array").New(buf.Len())
		js.CopyBytesToJS(out, buf.Bytes())
		return out
	}))
	select {} // keep the Go runtime alive for further calls
}
