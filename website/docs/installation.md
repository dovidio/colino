# Installation

Get Colino running on your system in minutes. The easiest way is to download pre-built binaries from our GitHub releases.

## Quick Start - Download Binaries

### 1. Download the Latest Release

Visit our [GitHub Releases page](https://github.com/dovidio/colino/releases) and download the appropriate binary for your system:

- **macOS Intel**: `colino-darwin-amd64`
- **macOS Apple Silicon**: `colino-darwin-arm64`
- **Linux x64**: `colino-linux-amd64`
- **Windows x64**: `colino-windows-amd64.exe`

### 2. Install the Binary

**macOS/Linux:**
```bash
# Make the binary executable
chmod +x colino-*

# Rename for convenience
mv colino-darwin-* colino  # or colino-linux-*
```

**Windows:**
The `.exe` file is ready to use as-is.

### 3. Make it Available Everywhere (Optional)

**macOS/Linux:**
```bash
# Move to a directory on your PATH
sudo mv colino /usr/local/bin/colino

# Or create a local bin directory
mkdir -p ~/bin
mv colino ~/bin/
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
```

**Windows:**
Add the directory containing `colino.exe` to your PATH in System Settings.

### 4. Run the Setup Wizard
```bash
colino setup
```

The setup wizard will guide you through:
- Adding your first RSS feeds
- Configuring automatic content fetching
- Setting up YouTube transcript access (optional)
- Running your first content ingestion


## Need Help?

If you run into issues, please open an issue on [GitHub](https://github.com/dovidio/colino/issues) with details about your system and the error you're seeing.
