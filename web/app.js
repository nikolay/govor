"use strict";

const speakBtn = document.getElementById("speak");
const textEl = document.getElementById("text");
const speedEl = document.getElementById("speed");
const speedVal = document.getElementById("speedval");
const player = document.getElementById("player");
const download = document.getElementById("download");
const wave = document.getElementById("wave");
const transport = document.getElementById("transport");
const playpause = document.getElementById("playpause");
const timeEl = document.getElementById("time");
const seek = document.getElementById("seek");

const RATE = 44100;
let blobURL = null;
let waveImage = null; // cached waveform pixels, replayed under the playhead
let duration = 0; // seconds; from the sample count, available before metadata

// Strings the script sets at runtime; static text is swapped by CSS via
// body[data-lang] and the .lang-bg/.lang-en classes.
const STR = {
  bg: {
    loading: "зареждане…", speak: "говори", error: "грешка — презареди страницата",
    play: "пусни", pause: "пауза", download: "⤓ свали .wav", seek: "позиция в записа",
    placeholder: "здравей, аз съм правец 82",
  },
  en: {
    loading: "loading…", speak: "speak", error: "error — reload the page",
    play: "play", pause: "pause", download: "⤓ download .wav", seek: "playback position",
    placeholder: "type Bulgarian text, e.g. здравей свят",
  },
};
let lang = location.hash === "#en" ? "en" : "bg";
let speakState = "loading"; // loading | ready | error

function setSpeakState(state) {
  speakState = state;
  speakBtn.disabled = state !== "ready";
  speakBtn.textContent = STR[lang][state === "ready" ? "speak" : state];
}

function applyLang(l, updateHash) {
  lang = l;
  document.documentElement.lang = l;
  document.body.dataset.lang = l;
  if (updateHash) history.replaceState(null, "", "#" + l);
  const bgBtn = document.getElementById("lang-bg");
  const enBtn = document.getElementById("lang-en");
  bgBtn.classList.toggle("active", l === "bg");
  enBtn.classList.toggle("active", l === "en");
  bgBtn.setAttribute("aria-pressed", String(l === "bg"));
  enBtn.setAttribute("aria-pressed", String(l === "en"));
  textEl.placeholder = STR[l].placeholder;
  download.textContent = STR[l].download;
  seek.setAttribute("aria-label", STR[l].seek);
  playpause.setAttribute("aria-label", STR[l][player.paused ? "play" : "pause"]);
  setSpeakState(speakState);
}

async function loadWasm() {
  const go = new Go();
  let result;
  try {
    result = await WebAssembly.instantiateStreaming(fetch("govor.wasm"), go.importObject);
  } catch (e) {
    // Some servers (e.g. older local dev ones) don't send application/wasm,
    // which makes instantiateStreaming throw; fall back to a buffered load.
    const buf = await (await fetch("govor.wasm")).arrayBuffer();
    result = await WebAssembly.instantiate(buf, go.importObject);
  }
  go.run(result.instance); // registers govorSynth, then parks in select{}
  setSpeakState("ready");
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
    setSpeakState("error");
    return;
  }

  if (blobURL) URL.revokeObjectURL(blobURL);
  blobURL = URL.createObjectURL(new Blob([wav], { type: "audio/wav" }));

  player.src = blobURL;
  player.play();

  download.href = blobURL;
  transport.hidden = false;

  drawWave(wav);
}

// Draw the synthesized waveform: 16-bit mono PCM starts after the 44-byte
// WAV header.
function drawWave(wav) {
  // wav.go always writes a canonical 44-byte header with the "data" chunk id
  // at byte 36; if that ever changes, drop the (purely cosmetic) waveform —
  // including any previous one — rather than render or keep stale pixels.
  if (String.fromCharCode(wav[36], wav[37], wav[38], wav[39]) !== "data") {
    waveImage = null;
    duration = 0;
    wave.hidden = true;
    return;
  }
  const samples = new Int16Array(wav.buffer, wav.byteOffset + 44, (wav.length - 44) >> 1);
  duration = samples.length / RATE;
  seek.max = duration;
  const off = document.createElement("canvas");
  off.width = wave.width;
  off.height = wave.height;
  const ctx = off.getContext("2d");
  const { width: w, height: h } = off;
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
  waveImage = off;
  wave.hidden = false;
  paint();
}

// Repaint the waveform with the playhead at the current position.
function paint() {
  timeEl.textContent = player.currentTime.toFixed(1) + " s";
  seek.value = player.currentTime;
  if (!waveImage) return;
  const ctx = wave.getContext("2d");
  ctx.clearRect(0, 0, wave.width, wave.height);
  ctx.drawImage(waveImage, 0, 0);
  if (duration > 0) {
    const x = Math.min(wave.width - 1, (player.currentTime / duration) * wave.width);
    ctx.fillStyle = "#aaffc3";
    ctx.fillRect(x, 0, 2, wave.height);
  }
}

let rafId = 0;
function tick() {
  paint();
  if (!player.paused) rafId = requestAnimationFrame(tick);
}

playpause.addEventListener("click", () => {
  if (player.paused) player.play(); else player.pause();
});
player.addEventListener("play", () => {
  playpause.textContent = "❚❚";
  playpause.setAttribute("aria-label", STR[lang].pause);
  playpause.setAttribute("aria-pressed", "true");
  // Reassigning player.src pauses without firing a pause event, so an old
  // tick loop may still be scheduled; cancel it to keep a single loop.
  cancelAnimationFrame(rafId);
  rafId = requestAnimationFrame(tick);
});
player.addEventListener("pause", () => {
  playpause.textContent = "►";
  playpause.setAttribute("aria-label", STR[lang].play);
  playpause.setAttribute("aria-pressed", "false");
  paint();
});
player.addEventListener("ended", () => {
  player.currentTime = 0;
  paint();
});
wave.addEventListener("click", (e) => {
  // Seeking needs loaded metadata (readyState >= HAVE_METADATA); setting
  // currentTime earlier throws in some browsers.
  if (!duration || player.readyState < 1) return;
  const rect = wave.getBoundingClientRect();
  const t = ((e.clientX - rect.left) / rect.width) * duration;
  player.currentTime = Math.min(Math.max(t, 0), duration);
  paint();
});
seek.addEventListener("input", () => {
  // Keyboard/screen-reader seek path; same metadata guard as the canvas.
  if (!duration || player.readyState < 1) return;
  player.currentTime = Math.min(Math.max(parseFloat(seek.value), 0), duration);
  paint();
});

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

document.getElementById("lang-bg").addEventListener("click", () => applyLang("bg", true));
document.getElementById("lang-en").addEventListener("click", () => applyLang("en", true));
window.addEventListener("hashchange", () => {
  applyLang(location.hash === "#en" ? "en" : "bg", false);
});
applyLang(lang, false);

loadWasm().catch((err) => {
  setSpeakState("error");
  console.error(err);
});
