# Troubleshooting Colino

If you encounter issues running or using Colino, check the following common problems and solutions.

## 1. “Database not found” from MCP tools

**Problem:**
`list_cache` or `get_content` returns a message like “Colino database not found …”.

**Solution:**
- Run a single ingestion to initialize the DB:
  ```bash
  ./colino daemon --once
  ```
- Optionally set a custom path in `~/.config/colino/config.yaml`:
  ```yaml
  database_path: "~/Library/Application Support/Colino/colino.db"
  ```

## 2. No transcript for YouTube links

**Problem:**
RSS items that link to YouTube are ingested, but content is empty or missing transcripts.

**Solution:**
- YouTube may be rate limiting. Try again later or configure a proxy. See [Configuration](./configuration.md) for the `youtube.proxy.webshare` settings.
- Some videos don’t have transcripts or restrict access; in those cases, no content is stored.

## 3. MCP client can’t discover tools

**Problem:**
Your MCP client doesn’t discover `list_cache`/`get_content` when running `./colino server`.

**Solution:**
- Ensure your client launches Colino over stdio and not as an HTTP server.
- Test from a terminal:
  ```bash
  ./colino server
  ```
- Restart the client so it re-handshakes with the server.

## 4. Launchd agent doesn’t run (macOS)

**Problem:**
`./colino daemon install` completed, but nothing ingests.

**Solution:**
- Check the log file path you configured; default is `~/Library/Logs/Colino/daemon.launchd.log`.
- Try unloading/reloading:
  ```bash
  ./colino daemon uninstall
  ./colino daemon install
  ```

## 5. Still stuck?

- Check the docs at https://colino.pages.dev
- Open an issue on https://github.com/dovidio/colino
