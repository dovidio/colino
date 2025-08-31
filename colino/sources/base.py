import logging
from abc import ABC, abstractmethod
from datetime import datetime
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from ..db import Database

logger = logging.getLogger(__name__)


class BaseSource(ABC):
    """Abstract base class for all content sources"""

    def __init__(self, db: "Database | None") -> None:
        self.db = db

    @abstractmethod
    def get_recent_content(
        self, since_time: datetime | None = None
    ) -> list[dict[str, Any]]:
        """
        Get recent posts from this source

        Args:
            since_time: Only return posts newer than this time

        Returns:
            List of post dictionaries with standardized format
        """
        pass

    @property
    @abstractmethod
    def source_name(self) -> str:
        """Return the name of this source (e.g., 'rss', 'youtube')"""
        pass
