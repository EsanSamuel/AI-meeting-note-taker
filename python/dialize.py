import sys
import json
import torch
import soundfile as sf
from pyannote.audio import Pipeline

def main():
    audio_path = sys.argv[1]
    hf_token = sys.argv[2]

    pipeline = Pipeline.from_pretrained(
        "pyannote/speaker-diarization-3.1",
        token=hf_token
    )

    data, sample_rate = sf.read(audio_path, dtype="float32")
    waveform = torch.from_numpy(data)
    if waveform.ndim == 1:
        waveform = waveform.unsqueeze(0)  # pyannote expects (channels, samples)
    else:
        waveform = waveform.T  # soundfile gives (samples, channels); pyannote wants (channels, samples)

    result = pipeline({"waveform": waveform, "sample_rate": sample_rate})
    annotation = result.speaker_diarization

    segments = []
    for turn, _, speaker in annotation.itertracks(yield_label=True):
        segments.append({
            "start": round(turn.start, 2),
            "end": round(turn.end, 2),
            "speaker": speaker
        })

    print(json.dumps(segments))

if __name__ == "__main__":
    main()