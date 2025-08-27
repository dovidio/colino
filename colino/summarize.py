import requests
from bs4 import BeautifulSoup
import openai
from typing import List, Dict, Any, Optional
import logging
from datetime import datetime
from .config import Config
from readability import Document

logger = logging.getLogger(__name__)

class ContentFetcher:
    """Fetches and cleans web content from URLs"""
    
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (compatible; Colino RSS Reader/1.0)'
        })
    
    def fetch_article_content(self, url: str) -> Optional[str]:
        """Fetch and extract main content from a web page"""
            
        try:
            logger.info(f"Fetching content from: {url}")
            
            response = self.session.get(url, timeout=Config.RSS_TIMEOUT)
            response.raise_for_status()
            
            # Use readability to extract main content
            doc = Document(response.text)  # Use .text instead of .content
            
            # Parse with BeautifulSoup for cleaning
            soup = BeautifulSoup(doc.content(), 'html.parser')
            
            # Remove unwanted elements
            for element in soup(['script', 'style', 'nav', 'footer', 'header', 'aside']):
                element.decompose()
            
            # Get text content
            content = soup.get_text(separator=' ', strip=True)
            
            # Clean up whitespace
            content = ' '.join(content.split())
            
            # Limit content length (LLMs have token limits)
            max_chars = 8000  # Roughly 2000 tokens
            if len(content) > max_chars:
                content = content[:max_chars] + "..."
            
            if len(content) > 100:  # Only use if we got substantial content
                logger.info(f"Extracted {len(content)} characters from {url}")
                return content
            else:
                logger.debug(f"Extracted content too short from {url}, using RSS content instead")
                return None
            
        except requests.exceptions.RequestException as e:
            logger.warning(f"Network error fetching {url}: {e}")
            return None
        except Exception as e:
            logger.warning(f"Could not parse content from {url}: {e}")
            return None

class DigestGenerator:
    """Generates AI-powered summaries of RSS content"""
    
    def __init__(self):
        Config.validate_openai_config()
        self.client = openai.OpenAI(api_key=Config.OPENAI_API_KEY)
        self.content_fetcher = ContentFetcher()

    def summarize_video(self, transcript: str) -> str:
        prompt = self.create_digest_video_prompt(transcript)
        try:
            response = self.client.chat.completions.create(
                model=Config.LLM_MODEL,
                messages=[
                    {
                        "role": "user", 
                        "content": prompt
                    }
                ],
                max_completion_tokens=4096
            )
            digest = response.choices[0].message.content
            logger.info("Generated AI digest successfully")
            return digest
            
        except Exception as e:
            logger.error(f"Error generating LLM digest: {e}")
            return ''

    def create_digest_video_prompt(self, transcript: str) -> str:
        """Create the prompt for LLM digest generation using config template"""
        from jinja2 import Template
        from config import Config
        import os
        
        # First try to get prompt from config
        template_content = Config.AI_PROMPT_YOUTUBE
        
        # If no prompt in config, try template files (backward compatibility)
        if not template_content:
            template_paths = [
                'src/templates/youtube_digest_prompt.txt',
                'templates/youtube_digest_prompt.txt',
                os.path.expanduser('~/.config/colino/templates/article_digest_prompt.txt')
            ]
            
            for template_path in template_paths:
                if os.path.exists(template_path):
                    with open(template_path, 'r') as f:
                        template_content = f.read()
                    break
        
        # No fallback - fail if prompt not configured
        if not template_content:
            raise ValueError("No AI prompt configured. Add 'prompt' to ai section in config.yaml")
        
        # Render template
        template = Template(template_content)
        return template.render(
            transcript=transcript
        )

    def summarize_article(self, article: Dict[str, Any]) -> str:
        # Start with RSS content
        content = article['content']

        metadata = article.get('metadata', {})
        title = metadata.get('entry_title', 'No title')
        feed_title = metadata.get('feed_title', '')
        url = article.get('url', '')
        source = article.get('author_display_name', 'Unknown source')
        
        # Try to fetch full article content if enabled
        if Config.LLM_SUMMARIZE_LINKS:
            if article.get('source') == 'youtube':
                full_content = metadata.get('youtube_transcript', metadata.get('full_content', ''))
            elif url: 
                full_content = self.content_fetcher.fetch_article_content(url)
            if full_content and len(full_content) > len(content):
                content = full_content
                logger.info(f"Using full article content for: {title}")

        article = {
            'title': title,
            'feed_title': feed_title,
            'content': content,
            'url': url,
            'source': source,
            'published': article.get('created_at', '')
        } 
        return self.generate_llm_article_digest(article)

    def generate_llm_article_digest(self, article: Dict[str, Any]):
        """Use LLM to generate a comprehensive digest"""
        
        # Prepare prompt
        prompt = self._create_digest_article_prompt(article)
        
        try:
            response = self.client.chat.completions.create(
                model=Config.LLM_MODEL,
                messages=[
                    {
                        "role": "user", 
                        "content": prompt
                    }
                ],
                max_completion_tokens=4096
            )
            
            digest = response.choices[0].message.content
            logger.info("Generated AI digest successfully")
            return digest
            
        except Exception as e:
            logger.error(f"Error generating LLM digest: {e}")
            return ''

    def _create_digest_article_prompt(self, article: Dict[str, Any]) -> str:
        """Create the prompt for LLM digest generation using config template"""
        from jinja2 import Template
        from config import Config
        import os
        
        # First try to get prompt from config
        template_content = Config.AI_ARTICLE_PROMPT_TEMPLATE
        
        # If no prompt in config, try template files (backward compatibility)
        if not template_content:
            template_paths = [
                'src/templates/article_digest_prompt.txt',
                'templates/article_digest_prompt.txt',
                os.path.expanduser('~/.config/colino/templates/article_digest_prompt.txt')
            ]
            
            for template_path in template_paths:
                if os.path.exists(template_path):
                    with open(template_path, 'r') as f:
                        template_content = f.read()
                    break
        
        # No fallback - fail if prompt not configured
        if not template_content:
            raise ValueError("No AI prompt configured. Add 'prompt' to ai section in config.yaml")
        
        # Prepare article data for template
        published = article['published']
        if isinstance(published, str):
            try:
                pub_date = datetime.fromisoformat(published.replace('Z', '+00:00'))
                published = pub_date.strftime('%Y-%m-%d %H:%M')
            except:
                pass
        
        template_article = {
            'title': article['title'],
            'source': article['source'],
            'published': published,
            'url': article['url'],
            'content': article['content']
        }
        
        # Render template
        template = Template(template_content)
        return template.render(
            article=template_article
        )
    
    def summarize_articles(self, articles: List[Dict[str, Any]]) -> str:
        """Generate a digest summary of multiple articles"""
        
        # Limit number of articles to process
        articles = articles[:Config.LLM_MAX_ARTICLES]
        
        logger.info(f"Generating digest for {len(articles)} articles")
        
        # Prepare article content for LLM
        article_summaries = []
        
        for i, article in enumerate(articles, 1):
            logger.info(f"Processing article {i}/{len(articles)}: {article.get('metadata', {}).get('entry_title', 'No title')}")
            
            # Start with RSS content
            content = article['content']

            metadata = article.get('metadata', {})
            title = metadata.get('entry_title', 'No title')
            feed_title = metadata.get('feed_title', '')
            url = article.get('url', '')
            source = article.get('author_display_name', 'Unknown source')
            
            # Try to fetch full article content if enabled
            if Config.LLM_SUMMARIZE_LINKS:
                if article.get('source') == 'youtube':
                    full_content = metadata.get('youtube_transcript', metadata.get('full_content', ''))
                elif url: 
                    full_content = self.content_fetcher.fetch_article_content(url)
                if full_content and len(full_content) > len(content):
                    content = full_content
                    logger.info(f"Using full article content for: {title}")
            
            article_summaries.append({
                'title': title,
                'feed_title': feed_title,
                'content': content,
                'url': url,
                'source': source,
                'published': article.get('created_at', '')
            })
        
        # Generate digest using LLM
        return self._generate_llm_digest(article_summaries)
    
    def _generate_llm_digest(self, articles: List[Dict[str, Any]]) -> str:
        """Use LLM to generate a comprehensive digest"""
        
        # Prepare prompt
        prompt = self._create_digest_prompt(articles)
        
        try:
            response = self.client.chat.completions.create(
                model=Config.LLM_MODEL,
                messages=[
                    {
                        "role": "user", 
                        "content": prompt
                    }
                ],
                max_completion_tokens=4096
            )
            
            digest = response.choices[0].message.content
            logger.info("Generated AI digest successfully")
            return digest
            
        except Exception as e:
            logger.error(f"Error generating LLM digest: {e}")
            return self._fallback_digest(articles)
    
    def _create_digest_prompt(self, articles: List[Dict[str, Any]]) -> str:
        """Create the prompt for LLM digest generation using config template"""
        from jinja2 import Template
        from config import Config
        import os
        
        # First try to get prompt from config
        template_content = Config.AI_PROMPT_TEMPLATE
        
        # If no prompt in config, try template files (backward compatibility)
        if not template_content:
            template_paths = [
                'src/templates/digest_prompt.txt',
                'templates/digest_prompt.txt',
                os.path.expanduser('~/.config/colino/templates/digest_prompt.txt')
            ]
            
            for template_path in template_paths:
                if os.path.exists(template_path):
                    with open(template_path, 'r') as f:
                        template_content = f.read()
                    break
        
        # No fallback - fail if prompt not configured
        if not template_content:
            raise ValueError("No AI prompt configured. Add 'prompt' to ai section in config.yaml")
        
        # Prepare article data for template
        template_articles = []
        for article in articles:
            published = article['published']
            if isinstance(published, str):
                try:
                    pub_date = datetime.fromisoformat(published.replace('Z', '+00:00'))
                    published = pub_date.strftime('%Y-%m-%d %H:%M')
                except:
                    pass
            
            template_articles.append({
                'title': article['title'],
                'source': article['source'],
                'published': published,
                'url': article['url'],
                'content': article['content']
            })
        
        # Render template
        template = Template(template_content)
        return template.render(
            articles=template_articles,
            article_count=len(articles)
        )
    
    # Removed fallback template - now requires explicit configuration
    
    def _fallback_digest(self, articles: List[Dict[str, Any]]) -> str:
        """Generate a simple fallback digest if LLM fails"""
        
        digest = f"# Daily Digest - {datetime.now().strftime('%Y-%m-%d')}\n\n"
        digest += f"## {len(articles)} Recent Articles\n\n"
        
        for article in articles:
            title = article['title']
            source = article['source']
            url = article['url']
            
            digest += f"### {title}\n"
            digest += f"**Source:** {source}\n"
            digest += f"**Link:** {url}\n\n"
            digest += f"{article['content'][:200]}...\n\n---\n\n"
        
        return digest 
