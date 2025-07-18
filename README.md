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
| `LOG_LEVEL`       | ⚠️ Optional | Set logging level        | `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `ENABLE_METRICS`  | ⚠️ Optional | Enable metrics collection | `true` or `false`             |
| `MAX_QUEUE_SIZE`  | ⚠️ Optional | Maximum queue size       | `500` (default)               |
| `MAX_PLAYLIST_SIZE` | ⚠️ Optional | Maximum playlist size   | `100` (default)               |
| `CACHE_DIR`       | ⚠️ Optional | Cache directory path     | `downloads` (default)         |
| `ENABLE_CACHING`  | ⚠️ Optional | Enable audio caching     | `true` or `false`             |
| `ENABLE_BUFFERING` | ⚠️ Optional | Enable pre-download buffer | `true` or `false`           |

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
| `pause`            | Pause current playback        | `pause`                                |
| `resume`           | Resume paused playback        | `resume`                               |

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
| `emergency-reset` | Emergency reset all systems    | `emergency-reset` |

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

## 🏛️ Professional Architecture

AutoMuse features a professionally-architected system designed for enterprise-grade reliability and maintainability:

### Core Architectural Components

#### **1. Configuration Management** (`config/config.go`)
- **Centralized Configuration**: All settings managed through a single, well-structured system
- **Environment Variable Support**: Comprehensive environment variable handling with validation
- **Type Safety**: Strongly typed configuration with proper data types and defaults
- **Feature Flags**: Enable/disable functionality through configuration
- **Performance Tuning**: Configurable parameters for optimal performance

#### **2. Structured Logging** (`pkg/logger/logger.go`)
- **High-Performance Logging**: Uses zerolog for structured, high-performance logging
- **Contextual Information**: Rich context support with user, guild, and command tracking
- **Multiple Outputs**: Console and file logging with automatic rotation
- **Performance Metrics**: Built-in memory usage and performance tracking
- **Configurable Levels**: Debug, Info, Warn, Error logging levels

#### **3. Dependency Validation** (`pkg/dependency/checker.go`)
- **System Health Checks**: Validates all required system dependencies on startup
- **Graceful Degradation**: Distinguishes between required and optional dependencies
- **Installation Guidance**: Provides detailed installation commands for missing dependencies
- **Version Validation**: Ensures minimum version requirements are met
- **Comprehensive Reporting**: Detailed health reports and recommendations

#### **4. Metrics Collection** (`pkg/metrics/metrics.go`)
- **Performance Monitoring**: Comprehensive metrics collection for system health
- **Business Analytics**: Command execution, queue size, cache usage tracking
- **Real-time Metrics**: Live performance tracking with histograms and timers
- **Resource Monitoring**: Memory, CPU, and goroutine monitoring
- **Operational Insights**: Detailed performance analytics for optimization

#### **5. Audio Service Management** (`internal/services/audio/manager.go`)
- **Per-Guild Sessions**: Isolated audio sessions for each Discord server
- **State Management**: Comprehensive playback state tracking and management
- **Resource Cleanup**: Automatic cleanup of inactive sessions
- **Thread Safety**: Concurrent-safe operations throughout
- **Error Recovery**: Robust error handling and automatic recovery

### Architecture Benefits

#### **Maintainability**
- **Modular Design**: Clear separation of concerns with defined interfaces
- **Type Safety**: Strong typing throughout the system
- **Documentation**: Comprehensive code documentation and examples
- **Testing**: Easier unit testing with dependency injection
- **Configuration**: Centralized configuration management

#### **Reliability**
- **Error Handling**: Comprehensive error handling and recovery mechanisms
- **Resource Management**: Proper cleanup and resource management
- **Graceful Degradation**: Handles missing dependencies and failures gracefully
- **State Consistency**: Consistent state management across all components
- **Input Validation**: Comprehensive input validation and sanitization

#### **Performance**
- **Efficient Operations**: High-performance structured logging and processing
- **Resource Monitoring**: Real-time resource monitoring and optimization
- **Memory Management**: Proper memory cleanup and leak prevention
- **Concurrent Safety**: Thread-safe operations with optimized locking
- **Optimized Audio**: Tuned audio processing parameters for best performance

#### **Operational Excellence**
- **Observability**: Comprehensive logging and metrics for system visibility
- **Health Monitoring**: Automatic system health checks and reporting
- **Configuration Management**: Environment-based configuration with validation
- **Dependency Tracking**: Automatic dependency validation and reporting
- **Graceful Shutdown**: Proper cleanup and resource release on shutdown

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
- **DCA Buffer Size**: 200 frames (~4 seconds) for optimized memory usage
- **Auto Cache Cleanup**: Removes songs older than 7 days to save disk space

### Advanced Configuration Options

Additional environment variables for fine-tuning:

```bash
# Logging Configuration
export LOG_LEVEL="INFO"          # DEBUG, INFO, WARN, ERROR
export LOG_FILE="logs/automuse.log"
export ENABLE_METRICS="true"     # Enable performance monitoring

# Performance Configuration  
export MAX_QUEUE_SIZE="500"      # Maximum total queue size
export MAX_PLAYLIST_SIZE="100"   # Maximum songs per playlist
export CACHE_DIR="downloads"     # Audio cache directory
export ENABLE_CACHING="true"     # Enable intelligent caching
export ENABLE_BUFFERING="true"   # Enable pre-download buffer

# Audio Configuration
export AUDIO_BITRATE="128"       # Audio bitrate (kbps)
export AUDIO_VOLUME="256"        # Discord volume level
export BUFFER_SIZE="5"           # Pre-download buffer size
```

## 📊 Monitoring & Analytics

### Structured Logging
AutoMuse provides comprehensive structured logging for debugging and monitoring:

- **Contextual Logs**: Each log entry includes user, guild, and command context
- **Performance Metrics**: Built-in memory usage and execution time tracking
- **Error Tracking**: Detailed error information with stack traces
- **Log Rotation**: Automatic log file rotation with compression

### Metrics Collection
When enabled, AutoMuse collects detailed performance metrics:

- **Command Execution**: Track success rates and execution times
- **Resource Usage**: Monitor memory, CPU, and goroutine counts  
- **Queue Analytics**: Track queue size, playlist processing times
- **Cache Performance**: Monitor cache hit rates and storage usage
- **System Health**: Track Discord connection status and errors

### Health Monitoring
Automatic system health checks include:

- **Dependency Validation**: Verify FFmpeg, yt-dlp, and other tools
- **Connection Health**: Monitor Discord and YouTube API connectivity
- **Resource Monitoring**: Track memory usage and performance
- **Error Rate Monitoring**: Alert on high error rates

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

<details>
<summary><strong>🔴 Performance Issues</strong></summary>

**Possible Causes:**

- High memory usage or CPU load
- Large queue or cache sizes
- Network connectivity problems

**Solutions:**

```bash
# Check system resources
top -p $(pgrep automuse)

# Monitor logs for performance issues
tail -f logs/automuse.log | grep -i "memory\|performance"

# Adjust performance settings
export MAX_QUEUE_SIZE="200"
export MAX_PLAYLIST_SIZE="50"
export ENABLE_METRICS="true"
```

</details>

<details>
<summary><strong>🔴 Dependency Issues</strong></summary>

**Possible Causes:**

- Missing or outdated dependencies
- Permission issues
- Path configuration problems

**Solutions:**

AutoMuse automatically validates dependencies on startup. Check the startup logs for detailed dependency reports and follow the provided installation instructions.

```bash
# Manual dependency check
ffmpeg -version
yt-dlp --version

# Update dependencies  
brew upgrade ffmpeg yt-dlp  # macOS
sudo apt update && sudo apt upgrade ffmpeg yt-dlp  # Linux
```

</details>

## 🚀 Future Enhancements

AutoMuse is continuously evolving with planned enhancements:

### **Database Integration**
- PostgreSQL/MySQL support for persistent storage
- Song metadata caching and user preferences
- Playlist management and favorites
- Historical playback analytics

### **API Integration**
- REST API for external integrations
- Webhook support for notifications
- GraphQL API for complex queries
- Rate limiting and authentication

### **Advanced Audio Features**
- Equalizer and audio effects processing
- Multiple audio source support (Spotify, SoundCloud)
- Live streaming capabilities
- Audio quality optimization

### **Scalability & Performance**
- Horizontal scaling support
- Load balancing for multiple instances
- Distributed caching systems
- Message queuing for high-load scenarios

### **Security & Operations**
- Enhanced input sanitization
- Advanced rate limiting
- User authentication and authorization
- Comprehensive audit logging

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Make your changes and test thoroughly
4. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
5. Push to the branch (`git push origin feature/AmazingFeature`)
6. Open a Pull Request

## 📝 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.
