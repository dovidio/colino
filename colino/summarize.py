import openai
from typing import List, Dict, Any, Optional
import logging
from datetime import datetime
from .config import Config
from jinja2 import Template
import os
        
logger = logging.getLogger(__name__)

class DigestGenerator:
    """Generates AI-powered summaries of RSS content"""
    
    def __init__(self):
        Config.validate_openai_config()
        self.client = openai.OpenAI(api_key=Config.OPENAI_API_KEY)

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
        # Use the content that was already scraped during ingestion
        content = article['content']

        metadata = article.get('metadata', {})
        title = metadata.get('entry_title', 'No title')
        feed_title = metadata.get('feed_title', '')
        url = article.get('url', '')
        source = article.get('author_display_name', 'Unknown source')

        article_data = {
            'title': title,
            'feed_title': feed_title,
            'content': content,
            'url': url,
            'source': source,
            'published': article.get('created_at', '')
        } 
        return self.generate_llm_article_digest(article_data)

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
            
            # Use the content that was already scraped during ingestion
            content = article['content']

            metadata = article.get('metadata', {})
            title = metadata.get('entry_title', 'No title')
            feed_title = metadata.get('feed_title', '')
            url = article.get('url', '')
            source = article.get('author_display_name', 'Unknown source')
            
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
