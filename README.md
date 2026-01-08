# Vyom Middleware - Download Center

This repository hosts the **One-Line Installer** and release artifacts for the Vyom Middleware.

## 🚀 Quick Install

Run this command on your Raspberry Pi:

```bash
curl -fsSL https://code-execute-rishi.github.io/VG_Cloud_Middleware_Download/install.sh | sudo bash
```

## 📂 Repository Structure

*   `docs/`: Contains the Website and Installer Script.
*   **Releases**: Download `.deb` packages directly from the [Releases Page](../../releases).

## 🛠 For Maintainers (Manual Release)

Since this repository does not check in source code, releases must be built privately and uploaded manually.

1.  **Build** the `.deb` file on your local machine or private repo.
2.  **Create a New Release** in this repo.
3.  **Upload** `vyom-middleware_x.x.x_arm64.deb` (and `armhf`/`amd64` variants).
4.  **Publish**.
