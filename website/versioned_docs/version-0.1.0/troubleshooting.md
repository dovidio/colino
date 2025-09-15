# Troubleshooting Colino

If you encounter issues running or using Colino, check the following common problems and solutions.

## 1. Shell errors with URLs (zsh: no matches found)

**Problem:**
When running a command like:

```
colino digest https://www.youtube.com/watch?v=W26bBFyFaR8
```

You get:

```
zsh: no matches found: https://www.youtube.com/watch?v=W26bBFyFaR8
```

**Solution:**
This is a shell (zsh) issue, not a Colino bug. The shell interprets special characters like `?`, `&`, and `=` unless the URL is quoted or escaped. To fix:

- Wrap the URL in quotes:
  ```
  colino digest "https://www.youtube.com/watch?v=W26bBFyFaR8"
  ```
- Or escape special characters:
  ```
  colino digest https://www.youtube.com/watch\?v=W26bBFyFaR8
  ```
- Or install oh-my-zsh which escapes those characters automatically:

## 2. No content from youtube videos

**Problem:**
You don't get any summary from youtube videos.

**Solution:**
Youtube is rate limiting your requests. You can either wait a few hours and try again, or set up a proxy in your config file. See the [configuration guide](./configuration.md#webshare-proxy-configuration) for details.

## 3. OpenAI API key errors

**Problem:**
You see errors about missing or invalid OpenAI API keys.

**Solution:**
- Set your API key as an environment variable:
  ```
  export OPENAI_API_KEY="your_key_here"
  ```
- Or add it to your config file (not recommended for production).

## 4. Still stuck?

- Check the [documentation](https://colino.pages.dev)
- Open an issue on [GitHub](https://github.com/dovidio/colino)
