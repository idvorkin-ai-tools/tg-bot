# /// script
# requires-python = ">=3.11"
# dependencies = ["kokoro-onnx", "soundfile"]
# ///
"""Generate speech from text using Kokoro TTS."""
import subprocess
import sys
import tempfile
from pathlib import Path

import kokoro_onnx
import soundfile as sf

MODEL_DIR = Path.home() / ".local" / "share" / "tg-bot" / "kokoro"
ONNX_PATH = MODEL_DIR / "kokoro-v1.0.onnx"
VOICES_PATH = MODEL_DIR / "voices-v1.0.bin"

def ensure_models():
    if ONNX_PATH.exists() and VOICES_PATH.exists():
        return
    MODEL_DIR.mkdir(parents=True, exist_ok=True)
    base = "https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0"
    for name in ["kokoro-v1.0.onnx", "voices-v1.0.bin"]:
        if not (MODEL_DIR / name).exists():
            print(f"Downloading {name}...")
            subprocess.run(["wget", "-q", f"{base}/{name}", "-O", str(MODEL_DIR / name)], check=True)

def speak(text: str, output_path: str, voice: str = "am_puck", speed: float = 1.0):
    ensure_models()
    tts = kokoro_onnx.Kokoro(str(ONNX_PATH), str(VOICES_PATH))
    samples, sr = tts.create(text, voice=voice, speed=speed)

    # Write wav then convert to ogg
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        wav_path = tmp.name
    sf.write(wav_path, samples, sr)
    result = subprocess.run(
        ["ffmpeg", "-i", wav_path, "-c:a", "libopus", output_path, "-y"],
        capture_output=True,
    )
    if result.returncode != 0:
        Path(wav_path).unlink(missing_ok=True)
        raise RuntimeError(f"ffmpeg encoding failed: {result.stderr.decode()}")
    Path(wav_path).unlink(missing_ok=True)

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: speak.py <text> <output.ogg> [voice] [speed]")
        sys.exit(1)
    text = sys.argv[1]
    output = sys.argv[2]
    voice = sys.argv[3] if len(sys.argv) > 3 else "am_puck"
    speed = float(sys.argv[4]) if len(sys.argv) > 4 else 1.0
    speak(text, output, voice, speed)
    print(f"Generated: {output}")
