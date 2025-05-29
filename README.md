# 🎵 AutoMuse Discord Music Bot

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS-lightgrey?style=for-the-badge&logo=apple)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Discord](https://img.shields.io/badge/Discord-Bot-7289DA?style=for-the-badge&logo=discord)

_A high-performance Discord music bot written in Go with advanced YouTube integration_

</div>

## ✨ Features

- 🎵 **YouTube Integration** - Play individual videos and entire playlists
- 🚀 **High Performance** - Built in Go for optimal speed and memory usage
- 🔊 **Quality Audio** - Advanced DCA encoding for crystal-clear Discord streaming
- 📋 **Smart Queue Management** - Efficient playlist processing and queue controls
- 🔍 **YouTube Search** - Find and play music without leaving Discord
- 💾 **Intelligent Caching** - Stores downloaded audio for instant replays
- ⚡ **Low Latency** - Optimized voice channel handling and audio processing
- 🎛️ **Skip Controls** - Skip to any position in the queue with ease

## 🖥️ Platform Support

| Platform    | Status           | Notes                                    |
| ----------- | ---------------- | ---------------------------------------- |
| **macOS**   | ✅ Fully Tested  | Primary development and testing platform |
| **Linux**   | ⚠️ Experimental  | May require additional configuration     |
| **Windows** | ❌ Not Supported | Currently incompatible                   |

> **Note**: AutoMuse has been extensively tested on macOS and is guaranteed to work properly. Linux support is experimental and may require troubleshooting. Windows support is not currently available.

## 🛠️ Prerequisites

### Required Software

| Tool                                       | Version | Purpose             |
| ------------------------------------------ | ------- | ------------------- |
| [Go](https://golang.org/dl/)               | 1.22+   | Runtime environment |
| [FFmpeg](https://ffmpeg.org/)              | Latest  | Audio processing    |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | Latest  | YouTube downloading |

### Installation Commands

#### macOS (Recommended)

```bash
# Install Homebrew (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install all dependencies
brew install go ffmpeg yt-dlp
```

#### Linux (Ubuntu/Debian)

```bash
# Update package manager
sudo apt update && sudo apt upgrade -y

# Install Go
sudo apt install golang-go -y

# Install FFmpeg
sudo apt install ffmpeg -y

# Install yt-dlp
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

## 🔑 Environment Variables Setup

AutoMuse requires two essential tokens to function properly:

### Discord Bot Token

1. **Create a Discord Application**

   - Visit the [Discord Developer Portal](https://discord.com/developers/applications)
   - Click "New Application" and give it a name
   - Navigate to the "Bot" section in the sidebar
   - Click "Add Bot" and confirm

2. **Get Your Bot Token**

   - In the Bot section, click "Reset Token"
   - Copy the generated token immediately (you won't see it again!)
   - **Keep this token secret** - never share it publicly

3. **Set Bot Permissions**

   - In the Bot section, enable these permissions:
     - `Send Messages`
     - `Connect` (voice)
     - `Speak` (voice)
     - `Use Voice Activity`

4. **Invite Bot to Server**
   - Go to OAuth2 → URL Generator
   - Select scopes: `bot`
   - Select permissions: `Send Messages`, `Connect`, `Speak`, `Use Voice Activity`
   - Use generated URL to invite bot to your server

### YouTube API Token

1. **Create a Google Cloud Project**

   - Visit [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable the [YouTube Data API v3](https://console.cloud.google.com/apis/library/youtube.googleapis.com)

2. **Generate API Key**
   - Go to [Credentials](https://console.cloud.google.com/apis/credentials)
   - Click "Create Credentials" → "API Key"
   - Copy the generated API key
   - **Optional**: Restrict the key to YouTube Data API v3 for security

### Setting Environment Variables

#### Method 1: Export Commands (Temporary)

```bash
export BOT_TOKEN="your_discord_bot_token_here"
export YT_TOKEN="your_youtube_api_key_here"
```

#### Method 2: .env File (Recommended)

```bash
# Create .env file in project root
echo "BOT_TOKEN=your_discord_bot_token_here" > .env
echo "YT_TOKEN=your_youtube_api_key_here" >> .env
```

#### Method 3: Shell Profile (Permanent)

```bash
# Add to ~/.zshrc or ~/.bashrc
echo 'export BOT_TOKEN="your_discord_bot_token_here"' >> ~/.zshrc
echo 'export YT_TOKEN="your_youtube_api_key_here"' >> ~/.zshrc
source ~/.zshrc
```

## 🚀 Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/Wolfenheimm/automuse.git
cd automuse
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Set Environment Variables

```bash
# Replace with your actual tokens
export BOT_TOKEN="your_discord_bot_token"
export YT_TOKEN="your_youtube_api_key"
```

### 4. Build and Run

```bash
# Build the application
go build -o automuse

# Run the bot
./automuse
```

## 🎮 Commands

| Command            | Description                   | Example                                |
| ------------------ | ----------------------------- | -------------------------------------- |
| `play [URL]`       | Play a YouTube video          | `play https://youtube.com/watch?v=...` |
| `play [search]`    | Search and select music       | `play never gonna give you up`         |
| `play [number]`    | Play from search results      | `play 3`                               |
| `skip`             | Skip current song             | `skip`                                 |
| `skip [number]`    | Skip to specific position     | `skip 5`                               |
| `skip to [number]` | Alternative skip syntax       | `skip to 5`                            |
| `queue`            | Show current queue            | `queue`                                |
| `stop`             | Stop playback and clear queue | `stop`                                 |
| `remove [number]`  | Remove song from queue        | `remove 3`                             |

### Playlist Support

```bash
# Play entire YouTube playlists
play https://youtube.com/playlist?list=...

# The bot will process all songs and start playback once ready
```

## 🏗️ Architecture

### Audio Processing Pipeline

```
YouTube URL → yt-dlp → MP3 Cache → FFmpeg → Opus Encoding → Discord
```

### Key Components

- **Queue Manager**: Thread-safe playlist and queue handling
- **Audio Cache**: Local MP3 storage for faster repeated playback
- **Voice Manager**: Persistent voice connections between songs
- **Search Engine**: YouTube API integration for music discovery

## 🔧 Configuration

### Audio Quality Settings

The bot automatically optimizes audio quality:

- **Bitrate**: 128kbps (optimal for Discord)
- **Sample Rate**: 48kHz
- **Channels**: Stereo (2 channels)
- **Format**: Opus (Discord native)

### Performance Tuning

- **Concurrent Processing**: 3 parallel downloads for playlists
- **Buffer Size**: 17,000 frames for stable playback
- **Voice Timeout**: 5-second connection retry limit

## 🐛 Troubleshooting

<details>
<summary><strong>🔴 Bot Won't Start</strong></summary>

**Possible Causes:**

- Missing or invalid environment variables
- Incorrect file permissions
- Missing dependencies

**Solutions:**

```bash
# Verify tokens are set
echo $BOT_TOKEN
echo $YT_TOKEN

# Check Go installation
go version

# Rebuild application
go clean && go build
```

</details>

<details>
<summary><strong>🔴 No Audio Playback</strong></summary>

**Possible Causes:**

- FFmpeg not installed or not in PATH
- yt-dlp outdated or missing
- Network connectivity issues

**Solutions:**

```bash
# Test FFmpeg
ffmpeg -version

# Update yt-dlp
yt-dlp --update

# Test YouTube access
yt-dlp --simulate "https://youtube.com/watch?v=dQw4w9WgXcQ"
```

</details>

<details>
<summary><strong>🔴 Bot Can't Join Voice Channel</strong></summary>

**Possible Causes:**

- Missing Discord permissions
- Voice channel restrictions
- Bot not in same server

**Solutions:**

- Verify bot has `Connect` and `Speak` permissions
- Check voice channel user limit
- Ensure bot is invited to correct server
</details>

## 📝 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.
