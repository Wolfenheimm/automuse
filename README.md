# AutoMuse Discord Music Bot

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS-lightgrey?style=for-the-badge&logo=apple)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Discord](https://img.shields.io/badge/Discord-Bot-7289DA?style=for-the-badge&logo=discord)

A high-performance Discord music bot written in Go with YouTube integration and age-restricted content bypass

</div>

## Features

- **YouTube Integration** - Play videos, playlists, and search with age-restricted content bypass
- **High Performance** - Go-based with 128kbps streaming and intelligent caching
- **Queue Management** - Move, shuffle, remove songs with 500-song capacity
- **Pre-Download Buffer** - 5-song lookahead for instant skipping
- **Smart Caching** - Metadata management with automatic cleanup and duplicate detection
- **Comprehensive Controls** - Skip, pause, resume, stop, and emergency reset
- **History Tracking** - Playback history with persistence
- **Structured Logging** - Professional logging with metrics collection

## Platform Support

| Platform | Status | Notes |
| -------- | ------ | ----- |
| macOS | Fully Tested | Primary development platform |
| Linux | Experimental | May require additional configuration |
| Windows | Not Supported | Currently incompatible |

## Prerequisites

| Tool | Version | Purpose |
| ---- | ------- | ------- |
| [Go](https://golang.org/dl/) | 1.23+ | Runtime environment |
| [FFmpeg](https://ffmpeg.org/) | Latest | Audio processing |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | Latest | YouTube downloading |

### Installation

**macOS:**
```bash
brew install go ffmpeg yt-dlp
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install golang-go ffmpeg -y
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
sudo chmod +x /usr/local/bin/yt-dlp
```

## Environment Variables

### Required
- `BOT_TOKEN` - Discord bot token
- `YT_TOKEN` - YouTube Data API v3 key

### Optional
- `DEBUG` - Enable debug logging (`true`/`false`)
- `LOG_LEVEL` - Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`)
- `ENABLE_METRICS` - Enable metrics collection
- `MAX_QUEUE_SIZE` - Maximum queue size (default: 500)
- `CACHE_DIR` - Cache directory path (default: downloads)
- `ENABLE_CACHING` - Enable audio caching
- `ENABLE_BUFFERING` - Enable pre-download buffer

### Setup

1. **Discord Bot**: Create at [Discord Developer Portal](https://discord.com/developers/applications), get token from Bot section
2. **YouTube API**: Enable YouTube Data API v3 in [Google Cloud Console](https://console.cloud.google.com/), create API key
3. **Set Environment Variables**:
```bash
export BOT_TOKEN="your_discord_bot_token"
export YT_TOKEN="your_youtube_api_key"
```

## Quick Start

```bash
git clone https://github.com/Wolfenheimm/automuse.git
cd automuse
go mod download
export BOT_TOKEN="your_discord_bot_token"
export YT_TOKEN="your_youtube_api_key"
go build -o automuse
./automuse
```

## Commands

### Playback
- `play [URL/search]` - Play YouTube video/playlist or search
- `skip [position]` - Skip current song or to position
- `stop` - Stop playback and clear queue
- `pause` / `resume` - Pause/resume playback

### Queue Management
- `queue` - Show current queue
- `remove [number]` - Remove song from queue
- `move [from] [to]` - Move song between positions
- `shuffle` - Shuffle queue

### System
- `cache` - Show cache statistics
- `cache-clear` - Clear old cached songs
- `buffer-status` - Show buffer status
- `history` - Show playback history
- `emergency-reset` - Reset all systems

## Architecture

### Audio Pipeline
```
YouTube URL → yt-dlp (age-restricted bypass) → MP3 Cache → FFmpeg → DCA → Discord
```

### Key Components
- **Queue Manager** - Thread-safe queue handling with 500-song capacity
- **Buffer Manager** - Pre-downloads next 5 songs for instant skipping
- **Cache System** - Metadata-driven storage with duplicate detection
- **Age-Restricted Bypass** - Multiple methods for accessing restricted content
- **History Manager** - Persistent playback history tracking

## Technical Details

### Configuration
- Centralized configuration management with environment variable support
- Structured logging using zerolog with automatic rotation
- Dependency validation with health checks on startup
- Optional metrics collection for performance monitoring

### Audio Processing
- 128kbps Opus streaming with DCA encoding
- Configurable audio buffer (200 frames for memory safety)
- Multiple age-restricted bypass methods for comprehensive access
- Thread-safe operations with proper resource cleanup

## Configuration

### Audio Settings
- Download: 256kbps MP3, Stream: 128kbps Opus
- Buffer: 200 frames (~4 seconds) for memory safety
- Pre-download: 5 songs ahead for instant skipping

### Performance Settings
```bash
export MAX_QUEUE_SIZE="500"
export CACHE_DIR="downloads"
export ENABLE_CACHING="true"
export ENABLE_BUFFERING="true"
export ENABLE_METRICS="true"
```

## Troubleshooting

### Common Issues
- **Bot won't start**: Check `BOT_TOKEN` and `YT_TOKEN` environment variables
- **No audio**: Verify FFmpeg installation (`ffmpeg -version`)
- **Age-restricted videos fail**: Update yt-dlp (`yt-dlp --update`)
- **Performance issues**: Check logs and adjust `MAX_QUEUE_SIZE`
- **Dependencies**: Bot validates dependencies on startup with detailed reports

## License

MIT License - see [LICENSE](LICENSE) file for details.
