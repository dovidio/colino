# Frequently Asked Questions

Got questions about Colino? You're in the right place! Here are answers to common questions from our community.

## General Questions

### What is Colino?
Colino is a privacy-first tool that helps you build a personal knowledge garden from RSS feeds, articles, and YouTube videos. It stores everything locally on your device and makes it available to your AI assistant for analysis and summarization.

### How is Colino different from RSS readers?
Unlike traditional RSS readers that show you endless feeds, Colino focuses on **intentional consumption**:
- **No notifications or distractions**
- **AI-powered analysis** instead of manual scrolling
- **Local storage** instead of cloud services
- **Knowledge building** rather than consumption

### Do I need to be technical to use Colino?
Not at all! While we currently require building from source, we're working on pre-built binaries. The day-to-day usage is simple:
- `colino setup` - One-time configuration
- `colino list` - See what's new
- `colino server` - Connect with AI

## Privacy and Security

### Where does my data go?
**Nowhere!** Everything stays on your device:
- Content is stored in a local SQLite database
- No accounts or cloud services required
- No tracking or analytics
- You control your data completely

### Can Colino access my AI conversations?
No. Colino only provides content to your AI assistant when you explicitly ask for it. It cannot see your conversations or influence your AI in any way.

### Is my RSS feed usage private?
Yes. RSS fetching is done directly from your device. We don't track which feeds you subscribe to or what content you read.

## Usage and Features

### How many RSS feeds should I add?
Start with **3-5 high-quality feeds** that provide real value to you. Quality is more important than quantity. You can always add more later.

### Can I add social media feeds?
Colino works best with traditional RSS feeds (blogs, news sites, publications). While some social platforms provide RSS, they're often designed for engagement rather than information quality.

### What about YouTube videos?
When RSS items link to YouTube videos, Colino automatically fetches the transcript. This lets you evaluate video content without watching it, saving you time and attention.

### How does AI integration work?
Colino uses the **Model Context Protocol (MCP)** to connect with AI assistants like Claude. This means:
- Your AI can search and analyze your collected content
- You can ask questions about trends in your feeds
- Content stays local while AI processing happens in your AI client

### Can I use Colino without an AI assistant?
Yes! You can use the `colino list` command to see recent content and browse your feeds manually. AI integration is optional but powerful for deeper analysis.

## Technical Questions

### Why is it written in Go?
Go provides excellent performance for content processing, easy cross-platform compilation, and reliable background processing. It also means we can ship everything in a single binary.

### How much space does Colino use?
Content is stored as plain text, so it's quite efficient:
- 1,000 articles ≈ 10-50 MB
- YouTube transcripts are also text-only
- Database grows slowly over time

### Can I run Colino on a server?
Yes! You can run Colino on any machine and access it remotely via SSH or by syncing the database file. Many users run it on home servers or cloud instances.

### What happens if the database gets corrupted?
Colino uses SQLite, which is very reliable. If issues occur, you can:
1. Run `colino daemon` to rebuild from your feeds
2. Restore from a backup of the database file
3. Contact us for help debugging

## Troubleshooting

### Why isn't new content appearing?
Check these common issues:
- Run `colino daemon` manually to see any errors
- Verify your RSS feeds are still active
- Check your internet connection
- Try a different time window with `colino list --hours 72`

### My AI can't find any content
Make sure:
- You've run `colino daemon` at least once
- The MCP server is running (`colino server`)
- Try a longer time window when asking your AI
- Check your AI client's MCP configuration

### Content quality is poor
Consider:
- Removing low-quality RSS feeds
- Adding more specialized, high-quality sources
- Using your AI to filter and summarize content
- Focusing on sources that provide full-text content

## Getting Help

### Where can I get support?
- **Documentation**: Check our guides for detailed information
- **GitHub Issues**: Report bugs and request features
- **GitHub Discussions**: Ask questions and share ideas
- **Community**: Join our growing community of mindful information consumers

### How can I request features?
We love hearing ideas! Please:
1. Check existing issues and discussions
2. Consider how it aligns with our philosophy of intentional consumption
3. Open a feature request with details about your use case

### Can I contribute to Colino?
Absolutely! We welcome all kinds of contributions:
- **Code**: Fix bugs, add features, improve documentation
- **Testing**: Try on different platforms and configurations
- **Feedback**: Share your experience and suggestions
- **Community**: Help others in discussions and issues

See our [Contributing Guide](./contributing) for more details.

## Future Plans

### What's coming next?
We're working on:
- **Pre-built binaries** for easier installation
- **Web interface** for browsing content
- **Enhanced AI capabilities** and tools
- **Mobile apps** for on-the-go access
- **Community features** for sharing curated feeds

### How can I stay updated?
- **Star our GitHub repository** to see updates
- **Watch our releases** for new versions
- **Join our discussions** for community news
- **Follow our progress** on the roadmap

---

Still have questions? [Open an issue](https://github.com/dovidio/colino/issues) or [start a discussion](https://github.com/dovidio/colino/discussions) - we're here to help!