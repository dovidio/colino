import requests
from bs4 import BeautifulSoup
from typing import Optional
import logging
from .config import Config
from readability import Document

logger = logging.getLogger(__name__)

class ArticleScraper:
    """Scrapes and extracts full content from web articles"""
    
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (compatible; Colino RSS Reader/1.0)'
        })
    
    def scrape_article_content(self, url: str) -> Optional[str]:
        """Scrape and extract main content from a web page"""
        try:
            logger.info(f"Scraping content from: {url}")
            
            response = self.session.get(url, timeout=Config.RSS_TIMEOUT)
            response.raise_for_status()
            
            # Use readability to extract main content
            doc = Document(response.text)
            
            # Parse with BeautifulSoup for cleaning
            soup = BeautifulSoup(doc.content(), 'html.parser')
            
            # Remove unwanted elements
            for element in soup(['script', 'style', 'nav', 'footer', 'header', 'aside']):
                element.decompose()
            
            # Get text content
            content = soup.get_text(separator=' ', strip=True)
            
            # Clean up whitespace
            content = ' '.join(content.split())
            
            if len(content) > 100:  # Only use if we got substantial content
                logger.info(f"Scraped {len(content)} characters from {url}")
                return content
            else:
                logger.debug(f"Scraped content too short from {url}, keeping RSS content")
                return None
            
        except requests.exceptions.RequestException as e:
            logger.warning(f"Network error scraping {url}: {e}")
            return None
        except Exception as e:
            logger.warning(f"Could not scrape content from {url}: {e}")
            return None
