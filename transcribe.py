# /// script
# requires-python = ">=3.11"
# dependencies = ["onnx-asr[cpu,hub]"]
# ///
"""Transcribe a voice message using NVIDIA Parakeet."""
import subprocess
import sys
import tempfile
from pathlib import Path

from onnx_asr import load_model

model = None

def get_model():
    global model
    if model is None:
        model = load_model("nemo-parakeet-tdt-0.6b-v3")
    return model

def transcribe(audio_path: str) -> str:
    path = Path(audio_path)
    # Convert to wav if needed
    if path.suffix != ".wav":
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
            wav_path = tmp.name
        conv = subprocess.run(
            ["ffmpeg", "-i", str(path), "-ar", "16000", "-ac", "1", wav_path, "-y"],
            capture_output=True,
        )
        if conv.returncode != 0:
            Path(wav_path).unlink(missing_ok=True)
            raise RuntimeError(f"ffmpeg conversion failed: {conv.stderr.decode()}")
        result = get_model().recognize(wav_path)
        Path(wav_path).unlink(missing_ok=True)
        return result
    return get_model().recognize(audio_path)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: transcribe.py <audio_file>")
        sys.exit(1)
    print(transcribe(sys.argv[1]))
