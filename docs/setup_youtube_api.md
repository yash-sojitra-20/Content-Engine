# YouTube Data API v3 & YouTube Shorts Setup Guide

This guide walks you through setting up automated **YouTube Shorts** and video uploads for Content-Engine using the official **YouTube Data API v3**.

---

## 1. Overview

Content-Engine can automatically publish generated vertical videos directly to your YouTube channel as **YouTube Shorts** (or standard YouTube videos).

Key capabilities:
- **Automatic Shorts Recognition**: Formats titles with `#Shorts` tag and sets description hashtags so YouTube automatically indexes the video in the Shorts feed.
- **Configurable Privacy**: Choose `public`, `unlisted`, or `private` uploads (e.g. upload as `unlisted` for review before going public).
- **Topic Categorization**: Categorizes videos (default: Category ID `28` - *Science & Technology*).
- **Bounded Retries**: Handles network interruptions and transient rate limits gracefully.
- **Offline / Mock Mode**: Test pipeline execution without burning API quota or needing credentials (`SOCIAL_PLATFORM=mock`).

---

## 2. Prerequisites

1. A standard **Google / YouTube account** with an active YouTube channel.
2. A project in the **[Google Cloud Console](https://console.cloud.google.com/)**.

---

## 3. Step-by-Step Setup Instructions

### Step 3.1: Enable the YouTube Data API v3
1. Open the **[Google Cloud Console](https://console.cloud.google.com/)**.
2. Select your Google Cloud project (the same project used for Gemini or Google Drive, or a new one).
3. In the top search bar, search for **"YouTube Data API v3"**.
4. Click on **YouTube Data API v3** from Marketplace/APIs.
5. Click the blue **"Enable"** button.

---

### Step 3.2: Configure the OAuth Consent Screen
YouTube video uploads require channel owner consent via OAuth 2.0.

1. Navigate to **APIs & Services > OAuth consent screen** in the left sidebar.
2. Select **External** user type and click **Create**.
3. Fill in the required fields:
   - **App name:** `Content-Engine`
   - **User support email:** (select your Google email)
   - **Developer contact information:** (enter your Google email)
4. Click **Save and Continue**.
5. Under **Scopes**, click **Add or Remove Scopes**:
   - Filter or manually add the upload scope:
     `https://www.googleapis.com/auth/youtube.upload`
   - Click **Update**, then **Save and Continue**.
6. Under **Test Users**:
   - Click **+ Add Users** and enter your YouTube channel's Google email address.
   - Click **Add**, then **Save and Continue**.

---

### Step 3.3: Create OAuth 2.0 Client Credentials
1. Navigate to **APIs & Services > Credentials**.
2. Click **+ Create Credentials** at the top, and select **OAuth client ID**.
3. In **Application type**, select **Web application**.
4. Set **Name** to: `Content-Engine Uploader`.
5. Under **Authorized redirect URIs**, click **+ Add URI** and add:
   `https://developers.google.com/oauthplayground`
   *(This enables generating a long-lived refresh token in the next step).*
6. Click **Create**.
7. Copy your **Client ID** and **Client Secret**.

---

### Step 3.4: Generate a Refresh Token via OAuth Playground
A refresh token allows Content-Engine to autonomously generate fresh access tokens without requiring manual browser logins during scheduled runs.

1. Open the **[Google OAuth 2.0 Playground](https://developers.google.com/oauthplayground)**.
2. Click the gear icon (**OAuth 2.0 configuration**) in the top right:
   - Check the box: **"Use your own OAuth credentials"**.
   - Paste your **OAuth Client ID**.
   - Paste your **OAuth Client Secret**.
3. In the left panel under **Step 1: Select & authorize APIs**:
   - Scroll down to **YouTube Data API v3**.
   - Check `https://www.googleapis.com/auth/youtube.upload`.
   - Click the blue **"Authorize APIs"** button.
4. Sign in with your YouTube Google account and click **Allow** on the consent screen.
5. You will be redirected back to the Playground under **Step 2: Exchange authorization code for tokens**.
6. Click the blue **"Exchange authorization code for tokens"** button.
7. Locate the **Refresh token** in the response panel. Copy this value.

---

## 4. Configure Content-Engine Environment

Open your `.env` file and configure the social publishing section:

```dotenv
# Enable social media publishing
SOCIAL_PUBLISH_ENABLED=true
SOCIAL_PLATFORM=youtube

# Google OAuth2 Credentials
YOUTUBE_CLIENT_ID=your-client-id.apps.googleusercontent.com
YOUTUBE_CLIENT_SECRET=your-client-secret
YOUTUBE_REFRESH_TOKEN=your-refresh-token

# Video Visibility: "public", "unlisted", or "private"
YOUTUBE_PRIVACY_STATUS=unlisted

# Category ID (28 = Science & Technology, 22 = People & Blogs, 27 = Education)
YOUTUBE_CATEGORY_ID=28

# Default tags for Shorts indexing
YOUTUBE_DEFAULT_TAGS="shorts,tech,ai,contentengine"
```

> **Tip:** Start with `YOUTUBE_PRIVACY_STATUS=unlisted` so you can verify video quality and formatting in your YouTube Studio before making them public.

---

## 5. Verifying Social Publishing

### Option A: Test with Mock / Dry-Run (Zero Cost & Offline)
Verify the pipeline flow without touching the YouTube API or burning quota:

```bash
SOCIAL_PUBLISH_ENABLED=true SOCIAL_PLATFORM=mock go run cmd/content-engine/main.go --mode once
```

Output:
```text
=======================================================
 [SUCCESS] Content-Engine Pipeline Finished
 Run ID:      5d7ca0d7-2c96-4967-b5e0-6b2fb9941694
 Status:      succeeded
 Title:       How Generative AI Actually Works
 Topic:       Tech Explainer...
 Duration:    10s (9:16)
 Provider Job:mock-job-56363321
 File Link:   file:///path/to/data/videos/content_5d7ca0d7.mp4
 Social Post: https://youtube.com/shorts/mock_post_5d7ca0d7 (mock)
=======================================================
```

### Option B: Live Upload to YouTube
Run a one-shot pipeline with your live credentials:

```bash
go run cmd/content-engine/main.go --mode once
```

When completed, the video will be uploaded and accessible directly via:
`https://youtube.com/shorts/<video_id>` (for Shorts) or `https://youtu.be/<video_id>`

---

## 6. YouTube API Quotas & Best Practices

- **Default Quota:** Google allocates **10,000 units per day** for YouTube Data API v3 per project for free.
- **Upload Cost:** Each video upload (`videos.insert`) costs **1,600 units**.
- **Daily Capacity:** You can upload roughly **6 videos per day** under the free quota.
- If you need higher volume, you can request an audit/quota extension in Google Cloud Console under **APIs & Services > YouTube Data API v3 > Quotas**.
