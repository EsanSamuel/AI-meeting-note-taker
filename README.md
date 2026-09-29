# AI Note Taker

**AI Note Taker** (also referred to as *Afterword*) is an intelligent, local-first meeting intelligence and transcription platform. It ingests audio or video recordings, extracts and standardizes audio, transcribes speech with hardware-accelerated local models, identifies distinct speakers, and leverages AI to generate timestamped transcripts, structured action items, decisions, vector embeddings, and executive meeting summaries.

> For an in-depth architectural breakdown, sequence diagrams, and database entity relationships, see [architecture.md](architecture.md).

---

## Key Features

- **Local High-Performance Speech-to-Text:** Transcribes audio locally using `whisper.cpp` (C++ GGML), eliminating cloud transcription latency and per-minute transcription costs.
- **Neural Speaker Diarization:** Identifies speaker boundaries and turns using `pyannote.audio` 3.1.
- **Temporal Alignment:** Matches Whisper timestamp offsets with Pyannote intervals to assign speaker labels to every segment.
- **Runtime Inference Profiling:** Live telemetry measuring Whisper CPU usage, RAM RSS footprint, inference duration, and Real-Time Factor (RTF).
- **Vector Embeddings & Semantic Search:** Generates 384-dimensional embeddings via a local `llama.cpp` embedding server (`bge-small-en-v1.5`) stored in PostgreSQL using `pgvector` for semantic transcript search and RAG Q&A.
- **Action Items & Decisions Extraction:** Analyzes transcript chunks in parallel using Google Gemini to extract concrete decisions and action items with speaker attribution and timestamps.
- **Executive Meeting Synthesis:** Combines chunk-level summaries into a structured, executive summary covering Overview, Discussion Points, Decisions, Action Items, Open Questions, and Final Outcomes.
- **Multi-Tenant Authentication & RBAC:** Organization-scoped multi-tenancy with secure `HttpOnly` session cookies and role-based permissions (`owner`, `admin`, `member`).
- **Modern Desktop App:** Cross-platform desktop interface built with [Wails v2](https://wails.io/) and React/Vite, featuring synchronized audio playback, transcript search, speaker renaming, and task completion.

---

## Requirements

- **Operating System:** Windows 10/11 (x64)
- **Go:** 1.26+ installed
- **C/C++ Build Toolchain:** CMake and Visual Studio C++ Build Tools (or MinGW) for compiling `whisper.cpp` and `llama.cpp`
- **FFmpeg:** Installed and added to system `PATH` (`ffmpeg -version`)
- **Python:** Python 3.10+ with `torch`, `soundfile`, and `pyannote.audio`
- **PostgreSQL:** Version 15+ with the `pgvector` extension enabled
- **Hugging Face Account:** User Access Token with accepted terms for [pyannote/speaker-diarization-3.1](https://huggingface.co/pyannote/speaker-diarization-3.1) and [pyannote/segmentation-3.0](https://huggingface.co/pyannote/segmentation-3.0)
- **Google Gemini API Key:** For meeting summarization, decision extraction, and RAG Q&A ([Google AI Studio](https://aistudio.google.com/))
- **Wails CLI (Optional for desktop client):** `go install github.com/wailsapp/wails/v2/cmd/wails@latest` and Node.js 18+

---

## Quick Start Setup

### 1. Clone Repository and Submodules

Clone the repository recursively to fetch `whisper.cpp` and `llama.cpp`:

```powershell
git clone --recurse-submodules <repository-url>
cd ai_note_taker
```

If already cloned without submodules, initialize them:

```powershell
git submodule update --init --recursive
```

### 2. Configure Environment Variables

Copy the example configuration file:

```powershell
copy env.example .env
```

Open `.env` and fill in your values (see [env.example](env.example) for documentation on every parameter):

```dotenv
SERVER_PORT=8080
SERVER_ENV=development
DB_URL=postgresql://postgres:postgres@localhost:5432/ai_note_taker?sslmode=disable
RECORDINGS_DIR=./recordings
MAX_RECORDING_BYTES=104857600
HF_TOKEN=hf_your_huggingface_token
GEMINI_API_KEY=your_gemini_api_key
```

### 3. Install Python Dependencies

Install the packages required for neural speaker diarization:

```powershell
py -m pip install torch soundfile pyannote.audio
```

### 4. Build Whisper.cpp and Download Model

Build the `whisper-cli` executable:

```powershell
cd whisper\whisper.cpp
cmake -B build
cmake --build build --config Release
.\models\download-ggml-model.cmd tiny.en
cd ..\..
```

The service expects:
- Executable: `whisper/whisper.cpp/build/bin/Release/whisper-cli.exe`
- Model: `whisper/whisper.cpp/ggml-tiny.en.bin`

### 5. Build LLaMA.cpp Submodule

Build the `llama-server` binary used for local embeddings and completions:

```powershell
cd llama\llama.cpp
cmake -B build
cmake --build build --config Release
cd ..\..
```

The service expects:
- Executable: `llama/llama.cpp/build/bin/llama-server.exe`
- Embedding model: `llama/llama.cpp/models/embedding/bge-small-en-v1.5-q4_k_m.gguf`
- Chat model: `llama/llama.cpp/models/qwen/qwen2.5-1.5b-instruct-q4_k_m.gguf`

### 6. Database Setup & Migrations

Ensure PostgreSQL has the `vector` extension installed. Run the migrations in `internal/db/migrations/` sequentially (e.g. using `golang-migrate`):

```powershell
# Using golang-migrate
migrate -database "postgresql://postgres:postgres@localhost:5432/ai_note_taker?sslmode=disable" -path internal/db/migrations up
```

---

## Running the Application

### Start the Backend Service

Always run from the project root so relative binary and model paths resolve correctly:

```powershell
go run .\cmd
```

On startup, the backend automatically:
1. Connects to PostgreSQL and verifies the connection pool.
2. Spawns the local `llama.cpp` Chat Server on `http://localhost:8083`.
3. Spawns the local `llama.cpp` Embedding Server on `http://localhost:8082`.
4. Starts the Gin HTTP server on `http://localhost:8080`.

Verify server health:

```powershell
Invoke-RestMethod http://localhost:8080/health
# Returns: {"status":"ok"}
```

### Run the Desktop Client (Wails)

In a separate terminal:

```powershell
cd desktop\client
wails dev
```

Alternatively, run the web frontend in standalone Vite dev mode:

```powershell
cd desktop\client\frontend
npm install
npm run dev
```

The Vite dev server will run on `http://localhost:5173`.

---

## REST API Overview

All protected routes require an active session cookie set upon login.

### Authentication (`/api/v1/auth`)
- `POST /api/v1/auth/setup`: Create the initial organization and owner account.
- `POST /api/v1/auth/login`: Authenticate and receive `session` cookie.
- `POST /api/v1/auth/logout`: Invalidate session and clear cookie.
- `GET /api/v1/auth/me`: Current authenticated user information.
- `POST /api/v1/auth/invitations`: Invite new members (`owner`/`admin` only).
- `POST /api/v1/auth/invitations/accept`: Accept invitation token and set account password.
- `GET /api/v1/auth/organization/members`: List organization team members.
- `PATCH /api/v1/auth/users/:id/role`: Change user role (`admin`, `member`).

### Recordings & Processing (`/api/v1/recordings`)
- `POST /api/v1/recordings`: Upload audio/video recording (multipart field: `recording`). Runs audio extraction, Whisper STT, Pyannote diarization, chunk embeddings, and Gemini summarization.
- `POST /api/v1/recordings/:id/generate-ai-results`: Re-trigger AI analysis, chunk embeddings, decisions, and action item generation on an existing meeting.

### Meetings & Outcomes (`/api/v1/meetings`)
- `GET /api/v1/meetings`: List all meetings for the current user's organization.
- `GET /api/v1/meetings/:id`: Get meeting details and summary.
- `GET /api/v1/meetings/:id/audio`: Stream meeting audio.
- `PUT /api/v1/meetings/:id`: Update meeting details.
- `DELETE /api/v1/meetings/:id`: Delete meeting and cascaded data.
- `GET /api/v1/meetings/:id/decisions`: List decisions extracted from the meeting.
- `GET /api/v1/meetings/:id/action-items`: List action items for the meeting.
- `PATCH /api/v1/action-items/:id/complete`: Mark an action item as completed.

### Transcripts (`/api/v1/meetings/:id/transcript-segments`)
- `GET /api/v1/meetings/:id/transcript-segments`: List timestamped speaker segments.
- `PUT /api/v1/meetings/:id/transcript-segments/update-speakers`: Batch rename speaker labels (e.g., replace `SPEAKER_00` with `Alice`).
- `PUT /api/v1/transcript-segments/:id`: Update a transcript segment's text.

---

## Project Structure

```text
cmd/                     # Go application entrypoint
internal/
  auth/                  # Password hashing & token utilities
  config/                # Config loader, Gemini AI client, Sensory logger
  db/                    # PostgreSQL pool, SQL migrations, sqlc queries
  diarization/           # Python pyannote.audio runner
  handlers/              # HTTP request handlers (Auth, Meeting, Recording, Transcript)
  inference/             # Whisper runtime telemetry (CPU, RAM, RTF via gopsutil)
  llama/                 # llama.cpp server management (embedding & LLM endpoints)
  middleware/            # Session auth & role-based access control
  repository/            # Repository pattern wrapping sqlc queries
  router/                # Gin router setup and CORS configuration
  services/              # Core business services (Audio, File, Transcribe, DB services)
  transcription/         # Speech-to-text runners (Whisper & AssemblyAI)
desktop/
  client/                # Wails v2 desktop application & React/Vite frontend
python/                  # Python speaker diarization script (diarize.py)
whisper/whisper.cpp/     # Whisper C++ submodule
llama/llama.cpp/         # LLaMA & embedding C++ submodule
audio/                   # Extracted 16kHz WAV audio files
recordings/              # Incoming uploaded media storage
transcripts/             # Merged transcript JSON caches
summaries/               # Meeting analysis and summary JSON outputs
architecture.md          # Comprehensive architecture and technical documentation
env.example              # Documented environment variable template
```

---

## Troubleshooting

- **Whisper executable not found:** Build `whisper.cpp` with the `Release` configuration and always start the backend service from the repository root.
- **Whisper model not found:** Ensure `ggml-tiny.en.bin` was downloaded into `whisper/whisper.cpp/`.
- **Diarization failed:** Verify Python 3 is installed, packages (`torch`, `soundfile`, `pyannote.audio`) are present in your active environment, and your `HF_TOKEN` has accepted terms for `pyannote/speaker-diarization-3.1`.
- **Database connection error:** Ensure PostgreSQL is running, the database exists, and the `vector` extension is enabled (`CREATE EXTENSION IF NOT EXISTS vector;`).
- **Embedding server error:** Verify that `llama/llama.cpp/build/bin/llama-server.exe` was compiled and `llama/llama.cpp/models/embedding/bge-small-en-v1.5-q4_k_m.gguf` exists.
- **FFmpeg extraction failed:** Verify `ffmpeg.exe` is installed and reachable on system `PATH`.