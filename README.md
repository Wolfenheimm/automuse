# AutoMuse Discord Music Bot

A high-performance Discord music bot written in Go that supports YouTube playback with advanced audio handling and playlist management.

## Features

- 🎵 YouTube video and playlist playback
- 🔊 High-quality audio streaming using DCA (Discord Compressed Audio)
- 📋 Queue management system
- 🔍 YouTube search functionality
- 💾 Audio caching for faster playback
- ⚡ Low-latency voice channel handling

## Prerequisites

### Required Software

- Go 1.19 or higher
- FFmpeg
- yt-dlp
- Git

### Installation Instructions

#### For macOS:

```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required packages
brew install go
brew install ffmpeg
brew install yt-dlp
```

#### For Debian/Ubuntu:

```bash
# Update package list
sudo apt update

# Install Go
sudo apt install golang-go

# Install FFmpeg
sudo apt install ffmpeg

# Install yt-dlp
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

## Setup

1. Clone the repository:

```bash
git clone https://github.com/Wolfenheimm/automuse.git
cd automuse
```

2. Install Go dependencies:

```bash
go mod download
```

3. Set up your environment variables:

```bash
export DISCORD_TOKEN="your_discord_bot_token"
export YT_TOKEN="your_youtube_api_token"
```

4. Build the bot:

```bash
go build -o automuse
```

## Running the Bot

```bash
./automuse
```

## Commands

- `play [URL]` - Play a YouTube video
- `play -pl [URL]` - Play a YouTube playlist
- `play [number]` - Play a song from the queue or search results
- `skip` - Skip the current song
- `queue` - Show the current queue
- `search [query]` - Search for a song on YouTube

## Dependencies

The bot uses several Go packages:

- `github.com/bwmarrin/discordgo` - Discord API wrapper
- `github.com/jonas747/dca` - Discord audio encoding
- `github.com/kkdai/youtube/v2` - YouTube video processing
- `google.golang.org/api/youtube/v3` - YouTube API client

## Audio Processing

AutoMuse uses DCA (Discord Compressed Audio) for optimal audio streaming:

- Converts YouTube videos to MP3 format
- Encodes audio with optimal settings for Discord
- Implements keep-alive mechanisms for stable playback
- Caches downloaded audio for faster repeated playback

## Troubleshooting

### Common Issues

1. **Bot not joining voice channel:**

   - Ensure the bot has proper permissions
   - Check if the bot token is correct
   - Verify the voice channel is accessible

2. **Audio not playing:**

   - Check if FFmpeg is properly installed
   - Verify yt-dlp is up to date
   - Ensure the YouTube API token is valid

3. **Poor audio quality:**
   - Check your internet connection
   - Verify the voice channel region
   - Ensure the bot has sufficient bandwidth

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Discord Go community for their excellent documentation
- YouTube-DL project for video processing capabilities
- FFmpeg team for audio processing tools
