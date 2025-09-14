import logging
import os
from datetime import datetime
from typing import Any

import openai
from jinja2 import Template

from .config import config

logger = logging.getLogger(__name__)


class DigestGenerator:
    """Generates AI-powered summaries of RSS content"""

    def __init__(self) -> None:
        config.validate_openai_config()
        self.client = openai.OpenAI(api_key=config.OPENAI_API_KEY)

    def _get_fallback_template_paths(self, filename: str) -> list[str]:
        """Generate standard fallback paths for a template filename"""
        return [
            f"src/templates/{filename}",
            f"templates/{filename}",
            os.path.expanduser(f"~/.config/colino/templates/{filename}"),
        ]

    def _load_prompt_template(self, config_key: str, template_filename: str) -> str:
        """Load prompt template from config or fallback file paths"""
        # First try to get prompt from config
        template_content = getattr(config, config_key, None)

        # If no prompt in config, try template files (backward compatibility)
        if not template_content:
            fallback_paths = self._get_fallback_template_paths(template_filename)
            for template_path in fallback_paths:
                if os.path.exists(template_path):
                    with open(template_path) as f:
                        template_content = f.read()
                    break

        # No fallback - fail if prompt not configured
        if not template_content:
            raise ValueError(
                f"No AI prompt configured. Add '{config_key.lower().replace('_', '.')}' to ai section in config.yaml"
            )

        return template_content

    def _call_llm(self, prompt: str) -> str:
        """Make a call to the LLM with the given prompt. Stream if config.AI_STREAM is True, else wait for full response."""
        try:
            if config.AI_STREAM:
                stream_response = self.client.chat.completions.create(
                    model=config.LLM_MODEL,
                    messages=[{"role": "user", "content": prompt}],
                    max_completion_tokens=4096,
                    stream=True,
                )
                full_text = ""
                print("\n", end="", flush=True)
                for chunk in stream_response:
                    delta = chunk.choices[0].delta
                    # Only process content, log/skip other fields
                    if hasattr(delta, "content") and delta.content is not None:
                        # Handle carriage returns for overwrite-style output
                        content = delta.content
                        if "\r" in content:
                            # Overwrite the current line (simulate terminal behavior)
                            lines = content.split("\r")
                            for i, line in enumerate(lines):
                                if i == 0:
                                    print(line, end="", flush=True)
                                    full_text += line
                                else:
                                    print("\r" + line, end="", flush=True)
                                    # Overwrite last line in buffer
                                    if "\n" in full_text:
                                        # Only replace after last newline
                                        last_nl = full_text.rfind("\n")
                                        full_text = full_text[: last_nl + 1] + line
                                    else:
                                        full_text = line
                        else:
                            print(content, end="", flush=True)
                            full_text += content
                    else:
                        # Log/skip non-content deltas (role, function_call, etc.)
                        logger.debug(f"Skipping non-content delta: {delta}")
                print("\n", flush=True)
                logger.info("Generated AI digest successfully (streamed)")
                return full_text
            else:
                nonstream_response = self.client.chat.completions.create(
                    model=config.LLM_MODEL,
                    messages=[{"role": "user", "content": prompt}],
                    max_completion_tokens=4096,
                )
                digest = nonstream_response.choices[0].message.content
                logger.info("Generated AI digest successfully (non-streamed)")
                print("\n" + (digest or "") + "\n", flush=True)
                return digest or ""
        except Exception as e:
            logger.error(f"Error generating LLM digest: {e}")
            return ""

    def _format_published_date(self, published: str) -> str:
        """Format a published date string to a readable format"""
        if isinstance(published, str):
            try:
                pub_date = datetime.fromisoformat(published.replace("Z", "+00:00"))
                return pub_date.strftime("%Y-%m-%d %H:%M")
            except (ValueError, TypeError):
                pass
        return published

    def summarize_video(self, transcript: str) -> str:
        """Generate a digest summary of a video transcript"""
        prompt = self._create_video_prompt(transcript)
        return self._call_llm(prompt)

    def _create_video_prompt(self, transcript: str) -> str:
        """Create the prompt for video digest generation"""
        template_content = self._load_prompt_template(
            "AI_PROMPT_YOUTUBE", "youtube_digest_prompt.txt"
        )
        template = Template(template_content)
        return str(template.render(transcript=transcript))

    def summarize_article(self, article: dict[str, Any]) -> str:
        """Generate a digest summary of a single article"""
        article_data = self._prepare_article_data(article)
        return self.generate_llm_article_digest(article_data)

    def _prepare_article_data(self, article: dict[str, Any]) -> dict[str, Any]:
        """Prepare article data for digest generation"""
        metadata = article.get("metadata", {})

        return {
            "title": metadata.get("entry_title", "No title"),
            "feed_title": metadata.get("feed_title", ""),
            "content": article["content"],
            "url": article.get("url", ""),
            "source": article.get("author_display_name", "Unknown source"),
            "published": article.get("created_at", ""),
        }

    def generate_llm_article_digest(self, article: dict[str, Any]) -> str:
        """Generate an LLM digest for a single article"""
        prompt = self._create_article_prompt(article)
        return self._call_llm(prompt)

    def _create_article_prompt(self, article: dict[str, Any]) -> str:
        """Create the prompt for single article digest generation"""
        template_content = self._load_prompt_template(
            "AI_ARTICLE_PROMPT_TEMPLATE", "article_digest_prompt.txt"
        )

        # Prepare article data for template
        template_article = {
            "title": article["title"],
            "source": article["source"],
            "published": self._format_published_date(article["published"]),
            "url": article["url"],
            "content": article["content"],
        }

        template = Template(template_content)
        return str(template.render(article=template_article))

    def summarize_articles(
        self, articles: list[dict[str, Any]], limit: int | None = None
    ) -> str:
        """Generate a digest summary of multiple articles"""
        # Apply limit if provided
        if limit is not None:
            articles = articles[:limit]
        logger.info(f"Generating digest for {len(articles)} articles")

        # Prepare article content for LLM
        article_summaries = []

        for i, article in enumerate(articles, 1):
            logger.info(
                f"Processing article {i}/{len(articles)}: {article.get('metadata', {}).get('entry_title', 'No title')}"
            )
            article_summaries.append(self._prepare_article_data(article))

        # Generate digest using LLM
        return self._generate_llm_digest(article_summaries)

    def _generate_llm_digest(self, articles: list[dict[str, Any]]) -> str:
        """Use LLM to generate a comprehensive digest (always streamed). If LLM call fails, show error and return empty string."""
        prompt = self._create_multi_article_prompt(articles)
        try:
            result = self._call_llm(prompt)
            if not result:
                raise RuntimeError("No response from language model.")
            return result
        except Exception as e:
            print(
                f"\n❌ Error: Failed to generate digest from the language model. Exception: {e}\n"
            )
            return ""

    def _create_multi_article_prompt(self, articles: list[dict[str, Any]]) -> str:
        """Create the prompt for multi-article digest generation"""
        template_content = self._load_prompt_template(
            "AI_PROMPT_TEMPLATE", "digest_prompt.txt"
        )

        # Prepare article data for template
        template_articles = []
        for article in articles:
            template_articles.append(
                {
                    "title": article["title"],
                    "source": article["source"],
                    "published": self._format_published_date(article["published"]),
                    "url": article["url"],
                    "content": article["content"],
                }
            )

        template = Template(template_content)
        return str(
            template.render(articles=template_articles, article_count=len(articles))
        )

    def _fallback_digest(self, articles: list[dict[str, Any]]) -> str:
        """Generate a simple fallback digest if LLM fails"""
        digest = f"# Daily Digest - {datetime.now().strftime('%Y-%m-%d')}\n\n"
        digest += f"## {len(articles)} Recent Articles\n\n"

        for article in articles:
            title = article["title"]
            source = article["source"]
            url = article["url"]
            content = article["content"]

            digest += f"### {title}\n"
            digest += f"**Source:** {source}\n"
            digest += f"**Link:** {url}\n\n"
            digest += f"{content[:200]}...\n\n---\n\n"

        return digest
