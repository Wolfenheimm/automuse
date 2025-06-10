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
- 🔊 **Premium Audio Quality** - 256kbps audio with advanced DCA encoding
- 📋 **Advanced Queue Management** - Move, shuffle, and organize your music queue
- 🔍 **YouTube Search** - Find and play music without leaving Discord
- 💾 **Intelligent Caching** - Smart metadata management with automatic cleanup
- ⚡ **Instant Skip Performance** - 5-song pre-download buffer for zero-latency skipping
- 🎛️ **Comprehensive Controls** - Skip, move, shuffle, and manage your music experience
- 📊 **Cache Analytics** - Track usage statistics and manage storage efficiently
- 🔄 **Parallel Processing** - 4x concurrent downloads for faster playlist loading
- 🛡️ **Memory Safety** - Optimized audio buffer settings prevent memory exhaustion
- 🔧 **Smart Configuration** - Environment variable validation and safe defaults

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

### Required Environment Variables

| Variable          | Required    | Description              | Example                       |
| ----------------- | ----------- | ------------------------ | ----------------------------- |
| `BOT_TOKEN`       | ✅ Yes      | Discord bot token        | `MTIzNDU2Nzg5MDEyMzQ1Njc4...` |
| `YT_TOKEN`        | ✅ Yes      | YouTube Data API v3 key  | `AIzaSyA1B2C3D4E5F6G7H8I9...` |
| `GUILD_ID`        | ⚠️ Optional | Discord server ID        | `123456789012345678`          |
| `GENERAL_CHAT_ID` | ⚠️ Optional | Default voice channel ID | `987654321098765432`          |
| `DEBUG`           | ⚠️ Optional | Enable debug logging     | `true` or `false`             |

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
export DEBUG="false"  # Optional: Enable debug logging
```

#### Method 2: .env File (Recommended)

```bash
# Create .env file in project root
echo "BOT_TOKEN=your_discord_bot_token_here" > .env
echo "YT_TOKEN=your_youtube_api_key_here" >> .env
echo "DEBUG=false" >> .env
```

#### Method 3: Shell Profile (Permanent)

```bash
# Add to ~/.zshrc or ~/.bashrc
echo 'export BOT_TOKEN="your_discord_bot_token_here"' >> ~/.zshrc
echo 'export YT_TOKEN="your_youtube_api_key_here"' >> ~/.zshrc
echo 'export DEBUG="false"' >> ~/.zshrc
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

### Core Playback Commands

| Command            | Description                   | Example                                |
| ------------------ | ----------------------------- | -------------------------------------- |
| `play [URL]`       | Play a YouTube video          | `play https://youtube.com/watch?v=...` |
| `play [search]`    | Search and select music       | `play never gonna give you up`         |
| `play [number]`    | Play from search results      | `play 3`                               |
| `skip`             | Skip current song             | `skip`                                 |
| `skip [number]`    | Skip to specific position     | `skip 5`                               |
| `skip to [number]` | Alternative skip syntax       | `skip to 5`                            |
| `stop`             | Stop playback and clear queue | `stop`                                 |

### Queue Management Commands

| Command            | Description                 | Example    |
| ------------------ | --------------------------- | ---------- |
| `queue`            | Show current queue          | `queue`    |
| `remove [number]`  | Remove song from queue      | `remove 3` |
| `move [from] [to]` | Move song between positions | `move 2 5` |
| `shuffle`          | Shuffle the current queue   | `shuffle`  |

### Cache & Performance Commands

| Command         | Description                      | Example         |
| --------------- | -------------------------------- | --------------- |
| `cache`         | Show detailed cache statistics   | `cache`         |
| `cache-clear`   | Clear old cached songs (7+ days) | `cache-clear`   |
| `buffer-status` | Show buffer manager status       | `buffer-status` |

### Playlist Support

```bash
# Play entire YouTube playlists
play https://youtube.com/playlist?list=...

# The bot will process all songs and start playback once ready
```

## 🚀 Advanced Features

### Smart Caching System

AutoMuse features an intelligent caching system that:

- **Prevents Duplicate Downloads**: Detects similar song titles to avoid redundant downloads
- **Metadata Management**: Stores detailed song information including play counts and timestamps
- **Automatic Cleanup**: Removes old cached files to prevent disk space issues
- **Usage Analytics**: Tracks most played songs and provides detailed statistics

### Pre-Download Buffer

The buffer system ensures smooth playback by:

- **5-Song Lookahead**: Pre-downloads the next 5 songs in the queue
- **Instant Skipping**: Skip to pre-downloaded songs with zero latency
- **Background Downloads**: Maintains buffer automatically during playback
- **Parallel Processing**: Downloads up to 4 songs simultaneously

### Queue Management

Advanced queue controls include:

- **Position Moving**: Reorganize queue by moving songs between positions
- **Shuffle Mode**: Randomize your queue for variety
- **Smart Removal**: Remove specific songs by position number
- **Real-time Updates**: All changes reflect immediately during playback

## 🏗️ Architecture

### Audio Processing Pipeline

```
YouTube URL → yt-dlp → MP3 Cache → FFmpeg → Opus Encoding → Discord
```

### Key Components

- **Queue Manager**: Thread-safe playlist and queue handling with move/shuffle support
- **Buffer Manager**: Pre-downloads next 5 songs for instant skipping
- **Smart Cache System**: Metadata-driven MP3 storage with automatic cleanup
- **Voice Manager**: Persistent voice connections between songs
- **Search Engine**: YouTube API integration for music discovery
- **Performance Analytics**: Usage tracking and cache statistics

## 🔧 Configuration

### Audio Quality Settings

The bot automatically optimizes audio quality:

- **Download Quality**: 256kbps MP3 (premium quality)
- **Streaming Bitrate**: 128kbps Opus (optimal for Discord)
- **Sample Rate**: 48kHz
- **Channels**: Stereo (2 channels)
- **Format**: Opus (Discord native)

### Performance Optimizations

- **Concurrent Downloads**: 4 parallel downloads for faster playlist processing
- **Pre-Download Buffer**: 5-song lookahead for instant skipping
- **Smart Caching**: Metadata-driven duplicate detection and storage management
- **DCA Buffer Size**: 17,000 frames for stable playback
- **Auto Cache Cleanup**: Removes songs older than 7 days to save disk space

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
