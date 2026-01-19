# Marcus Discord Bot

_Hello Robert_

## What is this?

Marcus is a Discord bot written in Go. It does TTS with multiple voices (ElevenLabs and a built-in cached "marcus" voice; TikTok support is currently disabled), has AI chat integration via OpenRouter, and plays custom audio memes. There's also some entertainment commands thrown in for good measure.

### What it does

- Multiple TTS voices you can pick from
- AI chat with two modes: regular assistant and Marcus personality (he's kind of an ass)
- Random jokes, facts, and insults that get read aloud
- Custom audio meme system - just drop .wav files in a folder
- Caches audio so it doesn't spam the APIs
- Can target specific voice channels

---

## Commands

### TTS Commands

#### Basic TTS
- `!marcus <message>` or `!m <message>`
  - Converts text to speech using the default Marcus voice
  - Example: `!marcus Hello everyone!`

- `v!<voice> <message>`
  - Uses a specific voice for TTS
  - Example: `v!liam Hello there!`
  - Example with emotion: `v!alice [energetic] Let's go!`

#### Voice-Specific Subcommands
- `v!<voice>-insult` - Generates a random insult in the specified voice
- `v!<voice>-joke` - Tells a random joke in the specified voice
- `v!<voice>-fact` - Shares a random fact in the specified voice
- `v!<voice>-slur` - Plays a cached slur entry if it exists; does not generate new audio

#### Channel Targeting
You can target a specific voice channel by prefixing your message with `<channel_name>`:
- `!marcus <general> Hello from another channel!`
- `v!sarah <music> This will play in the music channel`

### Voice Management

- `v!voices` (alias: `!list-voices`)
  - Lists all available voices grouped by TTS provider
  - Shows example usage patterns
  - Note: ElevenLabs voices require ELEVEN_LABS_API_KEY to be set. Without it, only the built-in cached "marcus" voice is available.

### AI Question & Answer

- `!ask-ai <question>`
  - Ask a question to a general AI assistant (requires OPEN_ROUTER_KEY)
  - Uses web search for enhanced responses
  - Can be used outside of voice channels
  - The response is also spoken via TTS in your current or targeted voice channel
  - Example: `!ask-ai What is the capital of France?`

- `!ask-marcus <question>`
  - Ask a question to Marcus (with personality) (requires OPEN_ROUTER_KEY)
  - Marcus responds with his signature irreverent, funny, and blunt style
  - The response is also spoken via TTS in your current or targeted voice channel
  - Example: `!ask-marcus How are you today?`

### Entertainment Commands

- `!marcus-insult` or `v!<voice>-insult`
  - Random insult from the Evil Insult API
  
- `!marcus-joke` or `v!<voice>-joke`
  - Random joke (mostly terrible ones)
  
- `!marcus-fact` or `v!<voice>-fact`
  - Random useless fact

### Meme Commands

- `!list-memes`
  - Displays all available audio memes
  - Can be used outside of voice channels

- `!<meme-name>`
  - Plays a specific audio meme
  - If multiple files exist for a meme, plays a random one
  - Example: `!airhorn`

- `!<meme-name>-<variant>`
  - Plays a specific variant of a meme
  - Example: `!dracula-laugh`

- `!addmeme <command-name>`
  - Reply to a message that has exactly one .wav attachment, then run `!addmeme <command-name>` (no spaces or emojis in the name)
  - The file will be saved under MEMES_LOCATION and become playable as `!<command-name>` after the periodic rescan

---

## Environment Variables

### Required

- **DISCORD_BOT_TOKEN** - Your Discord bot token from the [Discord Developer Portal](https://discord.com/developers/applications)
- **magic_key** - Required session cookie for TikTok TTS (still required by the app even if TikTok TTS is currently disabled)

### Optional

- **AUDIO_DIR** - Where to cache TTS files (default: `./audio`). Uses a provider/voice/hash structure.
- **OPEN_ROUTER_KEY** - API key for AI commands from [OpenRouter](https://openrouter.ai/). Without it, AI may fail or be rate-limited.
- **ELEVEN_LABS_API_KEY** - Enables ElevenLabs TTS voices and loads your available ElevenLabs voices.
- **MEMES_LOCATION** - Where your .wav meme files are (default: `./memes`). Bot scans this and makes commands automatically.

---

## Building & Running

### Prerequisites

- Go 1.24 or higher
- FFmpeg (audio playback via dgvoice)

### Local Development

1. Clone the repository:
```bash
git clone <repository-url>
cd marcus
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
export DISCORD_BOT_TOKEN="your-token-here"
export magic_key="your-secret-key"
export OPEN_ROUTER_KEY="your-openrouter-key"  # optional
export AUDIO_DIR="./audio"
export MEMES_LOCATION="./memes"
```

4. Run the bot:
```bash
go run main.go
```

### Docker Deployment

Dockerfiles are in the `package/` directory. There's a base image and architecture-specific variants:
- `Dockerfile` - Standard build
- `Dockerfile-base` - Base image
- `Dockerfile-base-arm` - ARM architecture

Note: Compose and Kubernetes manifests live under `deploy/` (the old `depoy/` path was corrected).

There's also a `docker-compose.yaml` in the `deploy/` directory if you want to spin things up quickly.

Build and run:
```bash
docker build -f package/Dockerfile -t marcus-bot .
docker run -d \
  -e DISCORD_BOT_TOKEN="your-token" \
  -e magic_key="your-key" \
  -e OPEN_ROUTER_KEY="your-openrouter-key" \
  -e AUDIO_DIR="/audio" \
  -e MEMES_LOCATION="/memes" \
  -v /host/path/to/audio:/audio \
  -v /host/path/to/memes:/memes \
  marcus-bot
```

Or use docker-compose from the deploy directory:
```bash
cd deploy
docker-compose up -d
```

Mount volumes for `/audio` and `/memes` to keep your cached audio and custom memes between restarts.

---

## Project Structure

```
marcus/
├── main.go                 # Entry point and Discord session setup
├── pkg/
│   ├── command.go         # Command routing and parsing
│   ├── ask.go             # AI question & answer functionality
│   ├── fact.go            # Random facts command
│   ├── joke.go            # Random jokes command
│   ├── insult.go          # Random insults command
│   ├── meme.go            # Meme audio indexing and playback
│   ├── addmeme.go         # Add new meme by replying with a .wav
│   ├── slur.go            # Slur command (plays cached only, no new generation)
│   ├── tts/
│   │   ├── tts.go         # TTS manager and interface
│   │   ├── elevenlabs.go  # ElevenLabs TTS provider
│   │   ├── tiktok.go      # TikTok TTS provider (currently disabled in code)
│   │   ├── cache.go       # Audio caching (provider/voice/hash)
│   │   ├── cache_hash.go  # Cache path + hashing helpers
│   │   └── cache_metadata.go # Cache metadata (per-voice, master index)
│   └── util/
│       └── util.go        # Utility functions
├── audio/                 # Cached TTS audio files
├── memes/                 # Custom audio meme files
├── go.mod                 # Go module dependencies
└── README.md             # This file
```

---

## How It Works

### Command Parsing

There's a few ways to use commands:

1. `v!<voice>[-subcommand] [<channel>] [content]` - Pick a specific voice. Can't use this with `!marcus` at the same time.
2. `!marcus[-subcommand] [<channel>] [content]` - Uses the default Marcus voice. `!m` is shorthand.
3. `!<command>[-subcommand]` - Other stuff like `!ask-ai`, `!list-memes`

### TTS Caching

Generated audio gets cached so we're not hitting the APIs every time. Files are hashed by content, so if you say the same thing twice it just plays the cached version. Mount a volume if you're using Docker or you'll lose it all on restart.

### Meme System

Scans the memes folder for .wav files and makes commands out of them. Checks every 10 seconds for new files. You can organize stuff in subdirectories and it'll create variant commands. Just drop a .wav file in there and it's good to go.

---

## Contributing

### How to contribute

1. Fork it
2. Make a branch: `git checkout -b feature/whatever`
3. Do your thing (run `go fmt` and add comments for weird stuff)
4. Test it
5. Commit with a decent message
6. Push and make a PR

### Code style

- Use `go fmt`
- Name things clearly
- Comment complex parts
- Use `slog` for logging
- Handle errors properly

### Ideas for contributions

**Stuff that would be useful:**
- Unit tests for command parsing
- Better error handling
- More TTS providers
- Rate limiting
- Integration tests

**Features:**
- More entertainment commands
- Different AI personalities
- Web dashboard
- Voice cloning
- Multi-language support

**Docs:**
- More examples
- Tutorials
- API docs
- Deployment guides

**Infrastructure:**
- Better Docker setup
- K8s manifests
- CI/CD
- Monitoring

### Bugs and requests

If something's broken or you want a feature:
1. Check if someone already reported it
2. Say how to reproduce it
3. Include logs/errors
4. Mention your setup (OS, Go version, etc.)

---

## Tips

- Mount the audio directory if you're using Docker, otherwise you'll regenerate everything on restart
- Put memes in subdirectories to organize variants - the bot figures it out
- You need to be in a voice channel for TTS unless you target a specific channel
- Watch API rate limits
- Get an OpenRouter key if you plan to use the AI commands a lot

---

## Dependencies

- **[discordgo](https://github.com/bwmarrin/discordgo)** - Discord API library for Go
- **[dgvoice](https://github.com/bwmarrin/dgvoice)** - Voice connection support for Discord
- **[go-openrouter](https://github.com/revrost/go-openrouter)** - OpenRouter API client
- **[zerolog](https://github.com/rs/zerolog)** - Fast and structured logging
- **[gopus](https://layeh.com/gopus)** - Opus audio codec bindings

