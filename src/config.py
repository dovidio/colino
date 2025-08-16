import os
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

class Config:
    # Database Configuration
    DATABASE_PATH = os.getenv('DATABASE_PATH', 'colino.db')
    
    # General Settings
    MAX_POSTS_PER_FEED = int(os.getenv('MAX_POSTS_PER_FEED', '100'))
    DEFAULT_LOOKBACK_HOURS = int(os.getenv('DEFAULT_LOOKBACK_HOURS', '24'))
    
    # RSS Settings
    RSS_FEEDS = os.getenv('RSS_FEEDS', '').split(',') if os.getenv('RSS_FEEDS') else []
    RSS_USER_AGENT = os.getenv('RSS_USER_AGENT', 'Colino RSS Reader 1.0.0')
    RSS_TIMEOUT = int(os.getenv('RSS_TIMEOUT', '30'))
    
    # Content filtering
    FILTER_KEYWORDS = os.getenv('FILTER_KEYWORDS', '').split(',') if os.getenv('FILTER_KEYWORDS') else []
    EXCLUDE_KEYWORDS = os.getenv('EXCLUDE_KEYWORDS', '').split(',') if os.getenv('EXCLUDE_KEYWORDS') else []
    
    # LLM/AI Configuration
    OPENAI_API_KEY = os.getenv('OPENAI_API_KEY')
    LLM_MODEL = os.getenv('LLM_MODEL', 'gpt-3.5-turbo')
    LLM_MAX_ARTICLES = int(os.getenv('LLM_MAX_ARTICLES', '10'))
    LLM_SUMMARIZE_LINKS = os.getenv('LLM_SUMMARIZE_LINKS', 'true').lower() == 'true'
    
    @classmethod
    def validate_openai_config(cls):
        """Validate OpenAI API credentials"""
        if not cls.OPENAI_API_KEY:
            raise ValueError("Missing OPENAI_API_KEY. Get one from https://platform.openai.com/api-keys")
        return True 