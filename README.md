# Content-Engine

Autonomous AI Video Generation & Publishing Pipeline (Phase 1).

Content-Engine is a scheduled, headless pipeline that orchestrates the end-to-end lifecycle of short-form AI video generation:
1. **Brief Generation:** Calls an LLM (Google Gemini API) to invent creative concepts, punchy scripts, and visual scene prompts.
2. **Video Generation:** Submits asynchronous generation requests to video models (Google Veo / Luma Dream Machine), handles rate limits, and polls status.
3. **Asset Retrieval:** Streams generated MP4 files directly to disk without memory buffering.
4. **Cloud Storage:** Uploads completed videos with metadata to Google Drive (or local storage).
5. **Run History:** Persists full execution history, stage status, and links in SQLite, PostgreSQL, or JSONL.
6. **Resilience:** Prevents overlapping runs, retries transient failures with exponential backoff, and guarantees graceful shutdown.

---

## Architecture & Workflow

```text
       Cron Schedule (or CLI --mode=once)
                       │
                       ▼
             Create Pipeline Run
             (status: started)
                       │
                       ▼
       Generate Content Brief (Gemini LLM)
             (status: brief_generated)
                       │
                       ▼
        Submit Video Generation Job
     (Google Veo / Luma Dream Machine / Mock)
      (status: video_generation_submitted)
                       │
                       ▼
          Poll Job Status with Backoff
            (status: video_generating)
                       │
                       ▼
        Download Video (Streamed to Disk)
            (status: video_downloaded)
                       │
                       ▼
        Upload to Google Drive / Storage
             (status: video_uploaded)
                       │
                       ▼
             Record Run Metadata
            (status: succeeded)
```

---

## Prerequisites

- **Go 1.22+** (Linux / macOS / Windows)
- **Make** (optional, for convenience scripts)

---

## Quickstart

### 1. Clone & Setup Configuration

Copy the example configuration to `.env`:

```bash
cp configs/.env.example .env
```

To test immediately without any external API keys or credentials, the default configuration uses the built-in **mock providers** and **SQLite**:

```bash
RUN_MODE=once
LLM_PROVIDER=mock
VIDEO_PROVIDER=mock
STORAGE_PROVIDER=local
DB_TYPE=sqlite
```

### 2. Run in One-Shot Mode (`once`)

Executes the full pipeline once and exits with status 0 on success:

```bash
# Using Make
make run-once

# Or directly with Go
go run ./cmd/content-engine --mode=once
```

### 3. Run on Scheduled Cron (`schedule`)

Starts the autonomous background scheduler:

```bash
# Using Make
make run-schedule

# Or directly with Go
go run ./cmd/content-engine --mode=schedule
```

To stop, send `Ctrl+C` (`SIGINT`) or `SIGTERM`. Active generation runs will complete within the configured grace period (`SHUTDOWN_GRACE_PERIOD=30s`).

---

## Detailed Setup Guides

Step-by-step guides for setting up external provider credentials:

- 📖 **[Google Gemini API Setup Guide (Text & Video)](docs/setup_gemini_api.md)**: How to get an API key, model selection (`gemini-1.5-flash`, `veo-2.0-generate-001`), verification, and quotas.
- 📖 **[Google Drive Storage Setup Guide](docs/setup_google_drive.md)**: How to create a Google Cloud Service Account, download `credentials.json`, share a destination folder, and get the `GDRIVE_FOLDER_ID`.

---

## Provider Configuration

### 1. LLM Provider (Google Gemini API)
*(See complete walkthrough in [docs/setup_gemini_api.md](docs/setup_gemini_api.md))*

In `.env`:
```dotenv
LLM_PROVIDER=gemini
GEMINI_API_KEY=AIzaSy...your_gemini_api_key
GEMINI_MODEL=gemini-1.5-flash
CONTENT_TOPIC_PROMPT="Create a captivating 10-second tech micro-fact video about AI neural networks."
```
*Get your API key at [Google AI Studio](https://aistudio.google.com/).*

### 2. Video Generation Provider

#### Option A: Google Veo / Gemini Video
```dotenv
VIDEO_PROVIDER=gemini
VIDEO_API_KEY=AIzaSy...your_gemini_api_key
VIDEO_MODEL=veo-2.0-generate-001
VIDEO_POLL_INTERVAL=10s
VIDEO_MAX_WAIT=10m
```

#### Option B: Luma Dream Machine
```dotenv
VIDEO_PROVIDER=luma
VIDEO_API_KEY=luma-api-key-...
VIDEO_POLL_INTERVAL=10s
VIDEO_MAX_WAIT=10m
```

#### Option C: Mock Video Provider (Offline / Testing)
```dotenv
VIDEO_PROVIDER=mock
```
Simulates full asynchronous polling and produces a valid ISO MP4 video without external API calls.

### 3. Storage Provider

#### Option A: Google Drive
```dotenv
STORAGE_PROVIDER=gdrive
GDRIVE_CREDENTIALS_FILE=path/to/service-account.json
GDRIVE_FOLDER_ID=1A2B3C4D5E6F7G8H9
```
*Requires a Google Cloud service account with Drive API enabled and editor permissions shared on the destination folder.*

#### Option B: Local Disk Storage
```dotenv
STORAGE_PROVIDER=local
LOCAL_STORAGE_DIR="data/videos"
```

### 4. Run Persistence Database

#### Option A: SQLite (Default)
Embedded, zero CGO, zero-config:
```dotenv
DB_TYPE=sqlite
SQLITE_PATH="data/runs.db"
```

#### Option B: PostgreSQL
```dotenv
DB_TYPE=postgres
DATABASE_URL="postgres://user:password@localhost:5432/content_engine?sslmode=disable"
```

#### Option C: JSONL
Append-only flat-file:
```dotenv
DB_TYPE=jsonl
JSONL_PATH="data/runs.jsonl"
```

---

## Project Structure

```text
Content-Engine/
├── cmd/
│   └── content-engine/
│       └── main.go          # CLI entrypoint & signal handling
├── internal/
│   ├── config/              # Environment loading & validation
│   ├── content/             # Gemini LLM & brief schema validation
│   ├── video/               # Veo, Luma & Mock video providers
│   ├── storage/             # Google Drive & local disk storage
│   ├── store/               # SQLite, Postgres & JSONL run persistence
│   ├── retry/               # Exponential backoff & Retry-After logic
│   ├── pipeline/            # End-to-end execution coordinator
│   ├── scheduler/           # robfig/cron & overlap prevention
│   └── observability/       # slog structured logging & secret redaction
├── configs/
│   └── .env.example         # Documented environment template
├── data/                    # Database and output storage
├── tmp/                     # Temporary video download workspace
├── Makefile                 # Build and test shortcuts
├── go.mod
└── README.md
```

---

## Testing

Run the full automated test suite covering unit tests, retry backoffs, overlap prevention, mock providers, and SQLite/JSONL persistence:

```bash
make test
# or
go test -v ./...
```

---

## Phase 2 Roadmap

Once the autonomous pipeline is running reliably in Phase 1:
- [ ] Social Media Publishing (YouTube Shorts API, Instagram Graph API, Facebook Pages API, X API v2)
- [ ] Content calendar and niche strategy planner
- [ ] Automated subtitles, audio mastering, and watermarking via FFmpeg
- [ ] Webhook notifications (Slack/Discord) for run summaries and failure alerts
