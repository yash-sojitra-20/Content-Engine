# Google Gemini API Setup Guide

This guide walks you through setting up the **Google Gemini API** for both **creative content brief generation** (LLM text, scripts, scene prompts) and **AI video generation** (Google Veo / Gemini video).

---

## 1. Overview

Content-Engine uses Google Gemini in two phases:
1. **LLM Content Brief Generator:** Automatically creates viral concepts, voice-over scripts, scene-by-scene descriptions, and optimized visual generation prompts in structured JSON.
2. **Video Generation Engine:** Submits the visual scene prompt to Google's video models (e.g. Veo), monitors long-running generation jobs, and downloads the resulting MP4.

---

## 2. Prerequisites

- A standard Google Account (personal Gmail or Google Workspace).
- A web browser.

---

## 3. Step-by-Step API Key Setup

### Step 3.1: Open Google AI Studio
1. Navigate to **[Google AI Studio API Keys](https://aistudio.google.com/app/apikey)**.
2. Sign in with your Google account.

### Step 3.2: Create Your API Key
1. Click the blue **"Create API key"** button.
2. You will be prompted to:
   - **Create API key in new project**, OR
   - **Create API key in existing project** (recommended if you already have a Google Cloud project).
3. Click **Create API key**.
4. A modal will display your new API key (starts with `AIzaSy...` or similar).
5. Copy the API key to a secure clipboard.

---

## 4. Selecting the Right Model

In your `.env` file, you configure the model using `GEMINI_MODEL`:

| Model Name | Type | Best For | Speed | Cost |
| :--- | :--- | :--- | :--- | :--- |
| `gemini-1.5-flash` / `gemini-2.0-flash` | Multimodal Flash | Standard brief generation, fast structured JSON | **Fast (< 2s)** | Free Tier Available (15 RPM) |
| `gemini-1.5-pro` | Flagship Pro | Complex storylines, nuanced creative scripts | Moderate (3-5s) | Free Tier Available (2 RPM) |
| `gemini-3.8-flash` | Next-Gen Flash | Advanced reasoning and high-fidelity output | Variable | Free Tier Available |
| `veo-2.0-generate-001` | Video Model | High-definition text-to-video rendering | Async job (1-3 min) | Pay-as-you-go / Google Cloud Credits |

> [!TIP]
> For content brief generation, **`gemini-1.5-flash`** or **`gemini-2.0-flash`** is the recommended default because it delivers instant JSON output and has a generous free tier.

---

## 5. Configuring Your `.env` File

Open (or create) `/Path-to-Content-Engine/.env`:

```dotenv
# ------------------------------------------------------------------------------
# LLM Provider Configuration (Google Gemini)
# ------------------------------------------------------------------------------
LLM_PROVIDER=gemini

# Paste your API Key obtained from Google AI Studio:
GEMINI_API_KEY=AIzaSyYourGeminiApiKeyHere

# Model to use for generating briefs:
GEMINI_MODEL=gemini-1.5-flash

# Custom topic or theme for your video content:
CONTENT_TOPIC_PROMPT="Create a breathtaking, cinematic 10-second nature documentary scene of a misty redwood forest with morning golden sunlight rays piercing through ancient emerald trees and a crystal clear mountain stream."

# ------------------------------------------------------------------------------
# Video Generation Provider Configuration
# ------------------------------------------------------------------------------
# Set to "gemini" for Google Veo video generation, or "mock" for free offline testing:
VIDEO_PROVIDER=gemini

# Video API Key (uses your GEMINI_API_KEY if identical):
VIDEO_API_KEY=AIzaSyYourGeminiApiKeyHere

# Video generation model:
VIDEO_MODEL=veo-2.0-generate-001

# Polling parameters for asynchronous video jobs:
VIDEO_POLL_INTERVAL=15s
VIDEO_MAX_WAIT=10m
```

> [!NOTE]
> If you want to test prompt generation without incurring video generation charges, set `VIDEO_PROVIDER=mock`. The engine will call Gemini for real prompts, but generate a local mock `.mp4` for storage testing.

---

## 6. How to Verify Your API Key

You can test that your API key is active and view the models available on your account using `curl`:

```bash
# Replace with your actual API key:
API_KEY="AIzaSyYourGeminiApiKeyHere"

# Query the Gemini API models endpoint:
curl -s "https://generativelanguage.googleapis.com/v1beta/models?key=${API_KEY}" | grep '"name": "models/gemini'
```

If your key is valid, you will see a list of available Gemini models.

---

## 7. Troubleshooting & Common Issues

### Issue 1: `404 NOT_FOUND: models/<model> is not found`
- **Cause:** The model specified in `GEMINI_MODEL` might be misspelled, deprecated, or not enabled in your region.
- **Solution:** Check the available models on your key using the verification `curl` command above, and use an active model like `gemini-1.5-flash` or `gemini-2.0-flash`.

### Issue 2: `403 PERMISSION_DENIED / API key not valid`
- **Cause:** The API key was copied with leading/trailing whitespace, or the key was restricted to specific IPs/APIs.
- **Solution:** In [Google AI Studio](https://aistudio.google.com/app/apikey), verify that the API key has no restrictive IP restrictions and that the Generative Language API is enabled.

### Issue 3: `429 RESOURCE_EXHAUSTED / Quota exceeded`
- **Cause:** Exceeded the free tier rate limit (e.g. >15 requests per minute).
- **Solution:** The Content-Engine has built-in exponential backoff retry logic that automatically waits and retries. For production volume, link a billing account to your Google Cloud project.

### Issue 4: `Client.Timeout exceeded while awaiting headers`
- **Cause:** A reasoning/thinking model took longer than the default HTTP timeout to finish deep thought tokens.
- **Solution:** Increase `HTTP_TIMEOUT=120s` in `.env` or use a fast flash model like `gemini-1.5-flash`.
