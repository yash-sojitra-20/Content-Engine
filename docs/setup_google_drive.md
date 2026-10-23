# Google Drive Storage Setup Guide

This guide walks you through setting up **Google Drive** automated uploads for Content-Engine using a Google Cloud **Service Account**.

---

## 1. Overview

Content-Engine runs as an autonomous, scheduled background service. To upload generated videos to Google Drive without requiring manual browser sign-ins, it uses a **Google Cloud Service Account** with standard Service-to-Service authentication.

---

## 2. Prerequisites

- A standard Google Account with access to [Google Drive](https://drive.google.com).
- Access to the [Google Cloud Console](https://console.cloud.google.com/).

---

## 3. Step-by-Step Setup Instructions

### Step 3.1: Open Google Cloud Console & Select Project
1. Open the **[Google Cloud Console](https://console.cloud.google.com/)**.
2. Sign in with your Google account.
3. At the top-left (next to the "Google Cloud" logo), click the **Project Selector** dropdown.
4. Select your existing project (or click **New Project** to create one, e.g. `Content-Engine-Project`).

---

### Step 3.2: Enable the Google Drive API
1. In the search bar at the top, type **"Google Drive API"**.
2. Select **Google Drive API** from the Marketplace/APIs search results.
3. Click the blue **"Enable"** button.
   *(Wait a few seconds until the API status shows as enabled).*

---

### Step 3.3: Create a Service Account
1. Open the left sidebar menu (☰) and navigate to **IAM & Admin > Service Accounts** (or [open directly](https://console.cloud.google.com/iam-admin/serviceaccounts)).
2. Click **"+ Create Service Account"** at the top.
3. Fill in the basic details:
   - **Service account name:** `content-engine-uploader`
   - **Service account ID:** (auto-generated, e.g. `content-engine-uploader`)
   - **Description:** `Service account for Content-Engine automated video uploads`
4. Click **Create and Continue**.
5. Click **Done** on the optional role grants (roles are granted on the Drive folder itself in Step 3.5).

---

### Step 3.4: Generate and Download the JSON Key
1. In the Service Accounts list, locate your newly created service account.
2. Click on the **Service Account Email** (e.g. `content-engine-uploader@<your-project-id>.iam.gserviceaccount.com`).
3. Click on the **"Keys"** tab at the top.
4. Click **"Add Key" > "Create new key"**.
5. Ensure the key type is set to **JSON**, then click **Create**.
6. A `.json` file will automatically download to your computer.
7. Move or copy this file into the `Content-Engine` root directory and name it `credentials.json`:

```bash
# Example command (adjust the path to your downloaded file):
cp ~/Downloads/gen-lang-client-*.json path-to-Content-Engine/credentials.json
```

> [!NOTE]
> `credentials.json` is already included in [`.gitignore`](../.gitignore), so your private keys and credentials will never be committed to Git.

---

### Step 3.5: Create & Share Destination Folder (Crucial!)

> [!IMPORTANT]
> **A Service Account does NOT have access to your personal Google Drive by default.**
> You **must** share the destination folder with the Service Account email, just like sharing with another person!

1. Open **[Google Drive](https://drive.google.com/)** in your browser.
2. Click **New > New folder**, name it (e.g., `AI Generated Videos`), and click **Create**.
3. Right-click the folder and choose **Share > Share**.
4. Copy your Service Account email address:
   ```text
   content-engine-uploader@<your-project-id>.iam.gserviceaccount.com
   ```
5. Paste this email address into the **"Add people and groups"** field.
6. Make sure the permission role is set to **"Editor"**.
7. *(Optional)* Uncheck **"Notify people"** since service accounts do not receive email.
8. Click **Send** or **Share**.

---

### Step 3.6: Get Your Google Drive Folder ID
1. Double-click to open your newly created folder in Google Drive.
2. Look at the browser address/URL bar:
   ```text
   https://drive.google.com/drive/folders/1wcB2xiz6DxxxxxxxxxxxxX1JBf15
                                          └───────────────────────────────┘
                                                This is your FOLDER ID
   ```
3. Copy only the alphanumeric string at the end of the URL.

---

## 4. Configuring Your `.env` File

Open `path-to-Content-Engine/.env` and update the storage section:

```dotenv
# ------------------------------------------------------------------------------
# Storage Provider Configuration (Google Drive)
# ------------------------------------------------------------------------------
STORAGE_PROVIDER=gdrive

# Path to your downloaded Service Account credentials JSON file:
GDRIVE_CREDENTIALS_FILE=credentials.json

# The Folder ID copied from your Google Drive folder URL:
GDRIVE_FOLDER_ID=1wcB2xiz6DxxxxxxxxxxxxX1JBf15
```

---

## 5. Verification & Testing

To test that Google Drive upload works end-to-end, run:

```bash
cd path-to-Content-Engine
make run-once
```

### Expected Output in Logs:
```text
level=INFO msg="Uploading video to storage" stage=video_upload provider=gdrive
level=INFO msg="Video stored successfully" file_id=1A2B3C... link=https://drive.google.com/file/d/1A2B3C.../view
```

You can then open Google Drive or click the generated link to verify the uploaded video file.

---

## 6. Troubleshooting & Common Pitfalls

### Issue 1: `google api returned status 404: File not found: <FOLDER_ID>`
- **Cause:** The Service Account was not given access to the folder, or the `GDRIVE_FOLDER_ID` is mistyped.
- **Solution:** Re-verify that you shared the folder with the exact Service Account email (`content-engine-uploader@...`) and gave it **Editor** permissions.

### Issue 2: `google api returned status 403: The caller does not have permission`
- **Cause:** The Service Account was given **Viewer** access instead of **Editor**, or the Google Drive API was not enabled on the Google Cloud project.
- **Solution:**
  1. In Google Drive, check folder sharing settings and ensure the Service Account role is set to **Editor**.
  2. In Google Cloud Console, ensure **Google Drive API** shows as **Enabled**.

### Issue 3: `google credentials file not found at credentials.json`
- **Cause:** The JSON key file is missing from the directory or the file path is incorrect.
- **Solution:** Confirm the file exists at `path-to-Content-Engine/credentials.json`.
