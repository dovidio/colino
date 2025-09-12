# Privacy Policy for Colino

**Last Updated:** September 12, 2025

## Our Commitment to Your Privacy

At Colino, we believe your privacy is fundamental. We have built Colino with privacy-by-design principles, ensuring that your personal data remains under your control at all times.

## Zero Tracking, Zero Data Sales

- **No tracking:** We do not track your usage, behavior, or reading habits
- **No data sales:** We never sell, rent, or monetize your personal data
- **No analytics:** We do not collect analytics or usage statistics
- **No advertising:** We do not serve ads or work with advertising networks

## Local-First Architecture

Colino is designed as a **local-first application**, which means:

- All your RSS feeds, articles, and digests are stored locally on your device
- Your reading preferences and configurations remain on your machine
- No content data is transmitted to our servers
- You maintain complete control over your data

## Limited Backend Usage

The only backend component we operate is for **Google YouTube authentication**, and even this follows strict privacy principles:

### Google Authentication Server
- **Purpose:** Facilitates secure OAuth authentication with Google for fetching your YouTube subscriptions locally
- **Data stored:** Temporary authentication tokens only
- **Retention period:** All authentication data is automatically deleted from our servers after **10 minutes**
- **Access:** No human access to authentication data
- **Location:** Secure cloud infrastructure with encryption in transit and at rest

### What We Store (Temporarily)
During the 10-minute authentication window:
- OAuth session identifiers (non-personally identifiable)
- Temporary access tokens for YouTube API
- No personal information, email addresses, or account details

### What We Never Store
- Your Google account information
- Your YouTube viewing history
- Your subscriptions list
- Your personal content or preferences
- Any identifying information beyond the authentication session

## Data You Control

All persistent data in Colino is stored locally on your device:

- **RSS feed URLs and content**
- **YouTube subscription information**
- **Generated AI digests**
- **Application settings and preferences**
- **Local SQLite database**

You can export, backup, or delete this data at any time by managing the local files.

## Third-Party Services

Colino may interact with third-party services as configured by you:

- **RSS feeds:** Direct connections to RSS sources you specify
- **YouTube API:** For fetching your subscriptions (when authenticated)
- **OpenAI API:** For generating digests (using your API key)
- **Proxy services:** If you configure rotating proxies (using your credentials)

**Important:** Your API keys and credentials for these services are stored locally and never transmitted to our servers.

## Your Data Rights

Since Colino is local-first, you have complete control:

- **Access:** All your data is accessible in local files
- **Portability:** Export your data at any time
- **Deletion:** Delete the application and all data is removed
- **Modification:** Edit configurations and data as needed

## Security Measures

- **Local encryption:** Sensitive data can be stored with local encryption
- **Secure transmission:** All network communications use HTTPS/TLS
- **Minimal attack surface:** No user accounts or cloud storage to compromise
- **Open source:** Code is auditable for security verification

## Changes to This Policy

We will update this privacy policy if our practices change. The "Last Updated" date will reflect any modifications. Given our local-first approach, changes are unlikely to affect data handling significantly.

## Contact Us

If you have questions about this privacy policy or Colino's privacy practices:

- **GitHub Issues:** [github.com/dovidio/colino/issues](https://github.com/dovidio/colino/issues)
- **Email:** [colinosupport@fastmail.com](mailto:colinosupport@fastmail.com)

## Legal Compliance

This privacy policy complies with applicable privacy laws including GDPR, CCPA, and other regional privacy regulations. Our minimal data collection approach ensures we meet the highest privacy standards globally.

---

**Summary:** Colino respects your privacy by keeping everything local. We only handle authentication briefly and securely, then forget about it. Your data is yours, always.
