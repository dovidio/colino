
# Colino 🌱

Reclaim your attention from algorithmic feeds. Build a personal knowledge garden with high-quality content that serves your goals, not engagement metrics.

Colino is a privacy-first tool that helps you consume information intentionally. It gathers content from RSS feeds, articles, and YouTube videos, then makes it available to your AI assistant for deep analysis and understanding.

## ✨ Key Features

- **Privacy-First**: Everything stays local on your device. No accounts, no tracking, no cloud services.
- **Curated Sources**: Choose exactly what content enters your knowledge base.
- **AI Integration**: Analyze and summarize your content using your preferred AI assistant.
- **Automated Ingestion**: Set it up once and let Colino keep your library current.

## 🚀 Quick Start

```bash
# Build from source
git clone https://github.com/dovidio/colino.git
cd colino
go build -o colino ./cmd/colino

# Interactive setup
./colino setup

# Start using it
./colino list                # See what's new
./colino server              # Connect with AI
```

## 🧠 Who is Colino For?

- **Researchers** staying current without drowning in noise
- **Lifelong Learners** building expertise in specific domains
- **Professionals** needing industry insights without social media distractions
- **Anyone** looking to reclaim their attention from algorithmic feeds

## 📖 Documentation

Full documentation is available at [colino.pages.dev](https://colino.pages.dev):

- **[Introduction](https://colino.pages.dev/docs/introduction)** - Understand the philosophy
- **[Installation](https://colino.pages.dev/docs/installation)** - Get Colino running
- **[Usage Guide](https://colino.pages.dev/docs/usage)** - Daily workflows and examples
- **[Configuration](https://colino.pages.dev/docs/configuration)** - Customize your setup

## 🤝 Community

We're building a community of mindful information consumers. Join us!

- **[Contributing Guide](https://colino.pages.dev/docs/contributing)** - Help us improve Colino
- **[GitHub Issues](https://github.com/dovidio/colino/issues)** - Report bugs and request features
- **[FAQ](https://colino.pages.dev/docs/faq)** - Common questions and answers

## 🏗️ Development

Colino is written in Go and focuses on simplicity and reliability. macOS is our primary platform (with automatic scheduling), but it works on Linux and Windows too.

**Requirements:**
- Go 1.23+
- Git

**Development Setup:**
```bash
git clone https://github.com/dovidio/colino.git
cd colino
go build -o colino ./cmd/colino
```

**Run tests:**
```bash
go test ./...
```

**Install git hooks (auto-format + vet):**
```bash
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

## 📄 License

[MIT License](LICENSE) - feel free to use, modify, and contribute.

## 🙏 Acknowledgments

Built for everyone who believes in intentional information consumption over endless scrolling. Thank you for being part of this journey toward a more mindful relationship with information.

---

**Development Note**: This repo includes git hooks for automatic code formatting and quality checks. Enable them with `git config core.hooksPath .githooks && chmod +x .githooks/pre-commit`.
