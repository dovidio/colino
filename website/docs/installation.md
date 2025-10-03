# Installation

Get Colino running on your system in just a few minutes. Currently, we support building from source, with pre-built binaries coming soon.

## Quick Start (macOS & Linux)

### Prerequisites
- **Go 1.23+** - Download from [go.dev](https://go.dev/dl/)
- **Git** - For cloning the repository

### Installation Steps

1. **Clone the repository:**
   ```bash
   git clone https://github.com/dovidio/colino.git
   cd colino
   ```

2. **Build Colino:**
   ```bash
   go build -o colino ./cmd/colino
   ```

3. **Make it available everywhere (optional but recommended):**
   ```bash
   # Move to a directory on your PATH
   sudo mv colino /usr/local/bin/colino

   # Or create a local bin directory
   mkdir -p ~/bin
   mv colino ~/bin/
   echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
   ```

4. **Verify installation:**
   ```bash
   colino --version
   ```

5. **Run the setup wizard:**
   ```bash
   colino setup
   ```

The setup wizard will guide you through:
- Adding your first RSS feeds
- Configuring automatic content fetching
- Setting up YouTube transcript access (optional)
- Running your first content ingestion

## Platform-Specific Information

### macOS (Primary Platform)
- **Full support** including automatic background fetching via launchd
- **Optimized integration** with macOS notifications and logging
- **Recommended setup** for the best experience

### Linux
- **Full core functionality** works perfectly
- **Manual scheduling** required (use systemd or cron)
- **Same commands and features** as macOS

### Windows
- **Basic functionality** should work (not extensively tested)
- **Manual scheduling** via Task Scheduler
- **Community feedback welcome** to improve Windows support

## Next Steps After Installation

### 1. Configure Your Sources
The setup wizard will help you add RSS feeds. Start with 3-5 high-quality sources you trust.

### 2. Set Up Automatic Fetching (macOS)
If you're on macOS, the setup can install automatic background fetching:
```bash
colino daemon schedule
```

### 3. Connect with Your AI Assistant
Configure your AI client to use Colino as an MCP server:

```bash
colino server
```

Example for Claude Desktop:
```json
{
  "mcpServers": {
    "colino": {
      "command": "/usr/local/bin/colino",
      "args": ["server"]
    }
  }
}
```

## Troubleshooting Installation

### "command not found: colino"
- Make sure the binary is in your PATH
- Try using the full path: `/path/to/colino`
- Restart your terminal after modifying PATH

### "go: command not found"
- Install Go from [go.dev](https://go.dev/dl/)
- Verify installation: `go version`
- Restart your terminal

### Build Issues
- Ensure you have Go 1.23 or later
- Check that you're in the correct directory
- Try: `go clean && go build -o colino ./cmd/colino`

### Permission Issues
```bash
# If you can't write to system directories
chmod +x colino
mkdir -p ~/bin
mv colino ~/bin/
```

## Verification

Test your installation with these commands:

```bash
# Check version
colino --version

# See available commands
colino --help

# Test basic functionality
colino list --hours 1
```

## Need Help?

If you run into issues:

1. **Check the troubleshooting guide** for common problems
2. **Search existing GitHub issues** - someone may have already solved it
3. **Open an issue** with details about your system and the error you're seeing

## Future Installation Options

We're working on:
- **Pre-built binaries** for easier installation
- **Homebrew package** for macOS
- **Snap package** for Linux
- **Windows installer** with proper integration

Stay tuned to our [releases page](https://github.com/dovidio/colino/releases) for updates!
