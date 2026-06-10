"use strict";

const speakBtn = document.getElementById("speak");
const textEl = document.getElementById("text");
const speedEl = document.getElementById("speed");
const speedVal = document.getElementById("speedval");
const player = document.getElementById("player");
const download = document.getElementById("download");
const wave = document.getElementById("wave");

const RATE = 44100;
let blobURL = null;

async function loadWasm() {
  const go = new Go();
  const src = fetch("govor.wasm");
  const result = WebAssembly.instantiateStreaming
    ? await WebAssembly.instantiateStreaming(src, go.importObject)
    : await WebAssembly.instantiate(await (await src).arrayBuffer(), go.importObject);
  go.run(result.instance); // registers govorSynth, then parks in select{}
  speakBtn.disabled = false;
  speakBtn.textContent = "говори";
}

function speak() {
  const text = textEl.value;
  if (!text.trim()) {
    textEl.focus();
    return;
  }
  let wav = null;
  try {
    wav = govorSynth(text, parseInt(speedEl.value, 10), RATE);
  } catch (err) {
    // Thrown when the Go runtime has exited; only a reload restarts it.
    console.error(err);
  }
  if (!wav) {
    speakBtn.disabled = true;
    speakBtn.textContent = "грешка — презареди страницата";
    return;
  }

  if (blobURL) URL.revokeObjectURL(blobURL);
  blobURL = URL.createObjectURL(new Blob([wav], { type: "audio/wav" }));

  player.src = blobURL;
  player.hidden = false;
  player.play();

  download.href = blobURL;
  download.hidden = false;

  drawWave(wav);
}

// Draw the synthesized waveform: 16-bit mono PCM starts after the 44-byte
// WAV header.
function drawWave(wav) {
  const samples = new Int16Array(wav.buffer, wav.byteOffset + 44, (wav.length - 44) >> 1);
  const ctx = wave.getContext("2d");
  const { width: w, height: h } = wave;
  wave.hidden = false;
  ctx.clearRect(0, 0, w, h);
  ctx.fillStyle = "#33ff66";
  const step = samples.length / w;
  for (let x = 0; x < w; x++) {
    let min = 32767, max = -32768;
    const from = Math.floor(x * step), to = Math.max(from + 1, Math.floor((x + 1) * step));
    for (let i = from; i < to; i++) {
      if (samples[i] < min) min = samples[i];
      if (samples[i] > max) max = samples[i];
    }
    const y1 = (0.5 - max / 65536) * h;
    const y2 = (0.5 - min / 65536) * h;
    ctx.fillRect(x, y1, 1, Math.max(1, y2 - y1));
  }
}

speakBtn.addEventListener("click", speak);
textEl.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    if (!speakBtn.disabled) speak();
  }
});
speedEl.addEventListener("input", () => { speedVal.textContent = speedEl.value; });
for (const b of document.querySelectorAll(".ex")) {
  b.addEventListener("click", () => {
    textEl.value = b.textContent;
    if (!speakBtn.disabled) speak();
  });
}

loadWasm().catch((err) => {
  speakBtn.textContent = "грешка при зареждане";
  console.error(err);
});
