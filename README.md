# AI Note Taker

AI Note Taker is a local HTTP service that turns an uploaded recording into a timestamped, speaker-labelled transcript.

The service:

1. Accepts an audio or video recording.
2. Extracts mono 16 kHz WAV audio with FFmpeg.
3. Transcribes the audio with `whisper.cpp`.
4. Identifies speaker segments with `pyannote.audio`.
5. Merges the transcription and diarization results and returns the combined transcript.

## Requirements

- Windows
- Go 1.26.5 or later
- CMake and a C/C++ build toolchain for building `whisper.cpp`
- FFmpeg available on `PATH`
- Python 3 with `torch`, `soundfile`, and `pyannote.audio`
- A Hugging Face access token with access to `pyannote/speaker-diarization-3.1`

The repository includes `whisper.cpp` and `llama.cpp` as Git submodules. LLaMA integration is currently disabled in the Go entrypoint; `whisper.cpp` is required for the recording endpoint.

## Setup

Clone the repository and its submodules:

```powershell
git clone --recurse-submodules <repository-url>
cd ai_note_taker
```

If the repository was cloned without submodules, initialize them with:

```powershell
git submodule update --init --recursive
```

Install the Python dependencies in the Python environment you will use for diarization:

```powershell
py -m pip install torch soundfile pyannote.audio
```

Create a `.env` file in the project root:

```dotenv
HF_TOKEN=hf_your_token_here
RECORDINGS_DIR=./recordings
MAX_RECORDING_BYTES=104857600
```

`HF_TOKEN` is required for speaker diarization. Do not commit `.env`; it is ignored by Git.

## Build Whisper

Build the Windows CLI from the `whisper.cpp` submodule:

```powershell
cd whisper\whisper.cpp
cmake -B build
cmake --build build --config Release
```

Download the model used by the API (`tiny.en`):

```powershell
.\models\download-ggml-model.cmd tiny.en
```

The service expects these paths relative to the project root:

- Executable: `whisper/whisper.cpp/build/bin/Release/whisper-cli.exe`
- Model: `whisper/whisper.cpp/ggml-tiny.en.bin`

Return to the project root before running the Go service, because these paths are relative to the current working directory.

## Run

From the project root:

```powershell
go run .\cmd
```

The server listens on `http://localhost:8080`.

Check that it is running:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## API

### `GET /health`

Returns the service health status.

### `POST /api/v1/recordings`

Upload a recording as multipart form data using the field name `recording`:

```powershell
curl.exe -X POST http://localhost:8080/api/v1/recordings `
  -F "recording=@C:\path\to\recording.webm"
```

On success, the endpoint returns HTTP `201` with the recording metadata, raw Whisper JSON, diarization segments, and merged transcript segments. A recording larger than `MAX_RECORDING_BYTES` is rejected; the default limit is 100 MiB.

## Generated files

- Uploaded recordings are stored in `recordings/`.
- Extracted WAV files are stored in `audio/`.
- Merged transcript output is stored in `transcripts/` as `<audio-id>_transcript.json`.

The uploaded source file is removed after audio extraction completes. Extracted audio and transcript files are retained.

## Troubleshooting

- **Whisper executable not found:** build `whisper.cpp` with the Release configuration and run the service from the repository root.
- **Model not found:** download `tiny.en` into `whisper/whisper.cpp` using the model script above.
- **FFmpeg extraction failed:** verify that `ffmpeg.exe` is installed and available on `PATH`.
- **Diarization failed:** verify the Python environment, `HF_TOKEN`, and access to the gated pyannote model.
- **No recording file:** use the multipart field name `recording` exactly.

## Project layout

```text
cmd/                 Go application entrypoint
internal/handlers/   HTTP request handlers
internal/router/     HTTP routes
internal/services/   File, audio, and transcription services
internal/diarization Diarization process integration
python/              Python speaker-diarization script
whisper/whisper.cpp  Whisper inference submodule
llama/llama.cpp      LLaMA submodule for future/disabled integration
audio/               Extracted audio output
recordings/          Uploaded recording storage
transcripts/         Merged transcript output
```