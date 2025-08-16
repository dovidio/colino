import os
import yaml
from pathlib import Path
from typing import Dict, Any, List

class Config:
    def __init__(self):
        self._config = self._load_config()
    
    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from YAML files"""
        # Check config locations in order of preference
        config_paths = [
            Path.home() / '.config' / 'colino' / 'config.yaml',
            Path('config.yaml')
        ]
        
        for config_path in config_paths:
            if config_path.exists():
                print(f"Loading config from: {config_path}")
                with open(config_path, 'r') as f:
                    return yaml.safe_load(f)
                
        raise ValueError("No config file found. Please create a config.yaml file in the current directory or in ~/.config/colino/")
        
    # RSS Properties
    @property
    def RSS_FEEDS(self) -> List[str]:
        return self._config.get('rss', {}).get('feeds', [])
    
    @property
    def RSS_USER_AGENT(self) -> str:
        return self._config.get('rss', {}).get('user_agent', 'Colino RSS Reader 1.0.0')
    
    @property
    def RSS_TIMEOUT(self) -> int:
        return self._config.get('rss', {}).get('timeout', 30)
    
    @property
    def MAX_POSTS_PER_FEED(self) -> int:
        return self._config.get('rss', {}).get('max_posts_per_feed', 100)
    
    # Filter Properties
    @property
    def FILTER_KEYWORDS(self) -> List[str]:
        return self._config.get('filters', {}).get('include_keywords', [])
    
    @property
    def EXCLUDE_KEYWORDS(self) -> List[str]:
        return self._config.get('filters', {}).get('exclude_keywords', [])
    
    # AI Properties  
    @property
    def OPENAI_API_KEY(self) -> str:
        # Always prioritize environment variable for security
        return os.getenv('OPENAI_API_KEY') or self._config.get('ai', {}).get('openai_api_key')
    
    @property
    def LLM_MODEL(self) -> str:
        return self._config.get('ai', {}).get('model', 'gpt-3.5-turbo')
    
    @property
    def LLM_MAX_ARTICLES(self) -> int:
        return self._config.get('ai', {}).get('max_articles', 10)
    
    @property
    def LLM_SUMMARIZE_LINKS(self) -> bool:
        return self._config.get('ai', {}).get('extract_web_content', True)
    
    @property
    def AI_AUTO_SAVE(self) -> bool:
        return self._config.get('ai', {}).get('auto_save', True)
    
    @property
    def AI_SAVE_DIRECTORY(self) -> str:
        return self._config.get('ai', {}).get('save_directory', 'digests')
    
    @property
    def AI_PROMPT_TEMPLATE(self) -> str:
        return self._config.get('ai', {}).get('prompt', '')
    
    # Database Properties
    @property
    def DATABASE_PATH(self) -> str:
        return self._config.get('database', {}).get('path', 'colino.db')
    
    # General Properties
    @property
    def DEFAULT_LOOKBACK_HOURS(self) -> int:
        return self._config.get('general', {}).get('default_lookback_hours', 24)
    
    def validate_openai_config(self):
        """Validate OpenAI API credentials"""
        if not self.OPENAI_API_KEY:
            raise ValueError("Missing OPENAI_API_KEY environment variable. Get one from https://platform.openai.com/api-keys")
        return True

# Create global config instance
Config = Config() 