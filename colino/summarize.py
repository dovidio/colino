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
    
    def _load_prompt_template(self, config_key: str, fallback_paths: List[str]) -> str:
        """Load prompt template from config or fallback file paths"""
        # First try to get prompt from config
        template_content = getattr(Config, config_key, None)
        
        # If no prompt in config, try template files (backward compatibility)
        if not template_content:
            for template_path in fallback_paths:
                if os.path.exists(template_path):
                    with open(template_path, 'r') as f:
                        template_content = f.read()
                    break
        
        # No fallback - fail if prompt not configured
        if not template_content:
            raise ValueError(f"No AI prompt configured. Add '{config_key.lower().replace('_', '.')}' to ai section in config.yaml")
        
        return template_content
    
    def _call_llm(self, prompt: str) -> str:
        """Make a call to the LLM with the given prompt"""
        try:
            response = self.client.chat.completions.create(
                model=Config.LLM_MODEL,
                messages=[{"role": "user", "content": prompt}],
                max_completion_tokens=4096
            )
            
            digest = response.choices[0].message.content
            logger.info("Generated AI digest successfully")
            return digest
            
        except Exception as e:
            logger.error(f"Error generating LLM digest: {e}")
            return ''
    
    def _format_published_date(self, published: str) -> str:
        """Format a published date string to a readable format"""
        if isinstance(published, str):
            try:
                pub_date = datetime.fromisoformat(published.replace('Z', '+00:00'))
                return pub_date.strftime('%Y-%m-%d %H:%M')
            except:
                pass
        return published

    def summarize_video(self, transcript: str) -> str:
        """Generate a digest summary of a video transcript"""
        prompt = self._create_video_prompt(transcript)
        return self._call_llm(prompt)

    def _create_video_prompt(self, transcript: str) -> str:
        """Create the prompt for video digest generation"""
        fallback_paths = [
            'src/templates/youtube_digest_prompt.txt',
            'templates/youtube_digest_prompt.txt',
            os.path.expanduser('~/.config/colino/templates/youtube_digest_prompt.txt')
        ]
        
        template_content = self._load_prompt_template('AI_PROMPT_YOUTUBE', fallback_paths)
        template = Template(template_content)
        return template.render(transcript=transcript)

    def summarize_article(self, article: Dict[str, Any]) -> str:
        """Generate a digest summary of a single article"""
        article_data = self._prepare_article_data(article)
        return self.generate_llm_article_digest(article_data)
    
    def _prepare_article_data(self, article: Dict[str, Any]) -> Dict[str, Any]:
        """Prepare article data for digest generation"""
        metadata = article.get('metadata', {})
        
        return {
            'title': metadata.get('entry_title', 'No title'),
            'feed_title': metadata.get('feed_title', ''),
            'content': article['content'],
            'url': article.get('url', ''),
            'source': article.get('author_display_name', 'Unknown source'),
            'published': article.get('created_at', '')
        }

    def generate_llm_article_digest(self, article: Dict[str, Any]) -> str:
        """Generate an LLM digest for a single article"""
        prompt = self._create_article_prompt(article)
        return self._call_llm(prompt)

    def _create_article_prompt(self, article: Dict[str, Any]) -> str:
        """Create the prompt for single article digest generation"""
        fallback_paths = [
            'src/templates/article_digest_prompt.txt',
            'templates/article_digest_prompt.txt',
            os.path.expanduser('~/.config/colino/templates/article_digest_prompt.txt')
        ]
        
        template_content = self._load_prompt_template('AI_ARTICLE_PROMPT_TEMPLATE', fallback_paths)
        
        # Prepare article data for template
        template_article = {
            'title': article['title'],
            'source': article['source'],
            'published': self._format_published_date(article['published']),
            'url': article['url'],
            'content': article['content']
        }
        
        template = Template(template_content)
        return template.render(article=template_article)
    
    def summarize_articles(self, articles: List[Dict[str, Any]]) -> str:
        """Generate a digest summary of multiple articles"""
        # Limit number of articles to process
        articles = articles[:Config.LLM_MAX_ARTICLES]
        logger.info(f"Generating digest for {len(articles)} articles")
        
        # Prepare article content for LLM
        article_summaries = []
        
        for i, article in enumerate(articles, 1):
            logger.info(f"Processing article {i}/{len(articles)}: {article.get('metadata', {}).get('entry_title', 'No title')}")
            article_summaries.append(self._prepare_article_data(article))
        
        # Generate digest using LLM
        return self._generate_llm_digest(article_summaries)
    
    def _generate_llm_digest(self, articles: List[Dict[str, Any]]) -> str:
        """Use LLM to generate a comprehensive digest"""
        prompt = self._create_multi_article_prompt(articles)
        result = self._call_llm(prompt)
        
        # If LLM call failed, use fallback
        if not result:
            return self._fallback_digest(articles)
        
        return result
    
    def _create_multi_article_prompt(self, articles: List[Dict[str, Any]]) -> str:
        """Create the prompt for multi-article digest generation"""
        fallback_paths = [
            'src/templates/digest_prompt.txt',
            'templates/digest_prompt.txt',
            os.path.expanduser('~/.config/colino/templates/digest_prompt.txt')
        ]
        
        template_content = self._load_prompt_template('AI_PROMPT_TEMPLATE', fallback_paths)
        
        # Prepare article data for template
        template_articles = []
        for article in articles:
            template_articles.append({
                'title': article['title'],
                'source': article['source'],
                'published': self._format_published_date(article['published']),
                'url': article['url'],
                'content': article['content']
            })
        
        template = Template(template_content)
        return template.render(
            articles=template_articles,
            article_count=len(articles)
        )
    
    def _fallback_digest(self, articles: List[Dict[str, Any]]) -> str:
        """Generate a simple fallback digest if LLM fails"""
        digest = f"# Daily Digest - {datetime.now().strftime('%Y-%m-%d')}\n\n"
        digest += f"## {len(articles)} Recent Articles\n\n"
        
        for article in articles:
            title = article['title']
            source = article['source']
            url = article['url']
            content = article['content']
            
            digest += f"### {title}\n"
            digest += f"**Source:** {source}\n"
            digest += f"**Link:** {url}\n\n"
            digest += f"{content[:200]}...\n\n---\n\n"
        
        return digest 
