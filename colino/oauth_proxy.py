"""
OAuth Proxy Client for handling Google YouTube authentication via external OAuth server
"""

import logging
import time
import webbrowser
from typing import Any

import requests

logger = logging.getLogger(__name__)


class OAuthProxyClient:
    """Client for handling OAuth flow via external proxy server"""

    def __init__(self, base_url: str, timeout: int = 300):
        """
        Initialize OAuth proxy client

        Args:
            base_url: Base URL of the OAuth proxy server
            timeout: Maximum time to wait for OAuth completion (seconds)
        """
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.session = requests.Session()

    def authenticate(self) -> tuple[str, str]:
        """
        Perform complete OAuth flow and return access/refresh tokens

        Returns:
            Tuple of (access_token, refresh_token)

        Raises:
            Exception: If authentication fails
        """
        logger.info("Starting OAuth flow via proxy server...")

        # Step 1: Initiate authentication
        session_id, auth_url = self._initiate_auth()

        # Step 2: Open browser for user authentication
        logger.info("Opening browser for authentication...")
        webbrowser.open(auth_url)

        # Step 3: Poll for completion
        tokens = self._poll_for_tokens(session_id)

        logger.info("OAuth authentication completed successfully")
        return tokens["access_token"], tokens["refresh_token"]

    def refresh_token(self, refresh_token: str) -> str:
        """
        Refresh access token using refresh token

        Args:
            refresh_token: The refresh token

        Returns:
            New access token

        Raises:
            Exception: If token refresh fails
        """
        logger.info("Refreshing access token...")

        url = f"{self.base_url}/auth/refresh"
        payload = {"refresh_token": refresh_token}

        try:
            response = self.session.post(
                url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=30,
            )
            response.raise_for_status()

            data = response.json()
            logger.info("Access token refreshed successfully")
            return data["access_token"]

        except requests.RequestException as e:
            logger.error(f"Failed to refresh token: {e}")
            raise Exception(f"Token refresh failed: {e}") from e

    def _initiate_auth(self) -> tuple[str, str]:
        """
        Initiate OAuth flow and get authorization URL

        Returns:
            Tuple of (session_id, authorization_url)
        """
        url = f"{self.base_url}/auth/initiate"

        try:
            response = self.session.get(url, timeout=30)
            response.raise_for_status()

            data = response.json()
            session_id = data["session_id"]
            auth_url = data["authorization_url"]

            logger.info(f"OAuth session initiated: {session_id}")
            return session_id, auth_url

        except requests.ConnectionError as e:
            logger.error(f"Cannot connect to OAuth proxy server at {self.base_url}")
            raise Exception(f"OAuth proxy connection failed: {e}") from e
        except requests.Timeout as e:
            logger.error(f"OAuth proxy server timeout: {e}")
            raise Exception(f"OAuth proxy timeout: {e}") from e
        except requests.HTTPError as e:
            logger.error(f"OAuth proxy HTTP error: {e}")
            raise Exception(f"OAuth proxy HTTP error: {e}") from e
        except (KeyError, ValueError) as e:
            logger.error(f"Invalid response from OAuth proxy: {e}")
            raise Exception(f"OAuth proxy response error: {e}") from e
        except requests.RequestException as e:
            logger.error(f"Failed to initiate OAuth: {e}")
            raise Exception(f"OAuth initiation failed: {e}") from e

    def _poll_for_tokens(self, session_id: str) -> dict[str, Any]:
        """
        Poll for OAuth completion and retrieve tokens

        Args:
            session_id: The session ID from initiation

        Returns:
            Dictionary containing tokens and metadata
        """
        url = f"{self.base_url}/auth/poll/{session_id}"
        start_time = time.time()
        poll_interval = 2  # Start with 2 seconds
        max_interval = 10  # Max 10 seconds between polls

        logger.info(f"Polling for OAuth completion at: {url}")
        logger.info(f"Timeout: {self.timeout} seconds")

        while time.time() - start_time < self.timeout:
            try:
                response = self.session.get(url, timeout=30)
                logger.debug(f"Poll response: {response.status_code}")

                if response.status_code == 200:
                    # Success - tokens available
                    tokens = response.json()
                    logger.info("Authentication completed successfully")
                    return tokens

                elif response.status_code == 202:
                    # Still pending - continue polling
                    elapsed = int(time.time() - start_time)
                    logger.info(f"Authentication still pending... ({elapsed}s elapsed)")

                elif response.status_code == 404:
                    raise Exception(
                        "Session not found - authentication may have expired"
                    )

                else:
                    logger.warning(
                        f"Unexpected response status: {response.status_code}"
                    )
                    response.raise_for_status()

                # Wait before next poll with exponential backoff
                time.sleep(poll_interval)
                poll_interval = min(poll_interval * 1.2, max_interval)

            except requests.RequestException as e:
                logger.warning(f"Polling error: {e}")
                time.sleep(poll_interval)
                continue

        raise Exception(f"OAuth timeout after {self.timeout} seconds")


class TokenManager:
    """Manages OAuth tokens with automatic refresh using database storage"""

    def __init__(
        self, oauth_client: OAuthProxyClient, db=None, service: str = "youtube"
    ):
        """
        Initialize token manager

        Args:
            oauth_client: OAuth proxy client instance
            db: Database instance (if None, will create one)
            service: Service name for token storage (default: "youtube")
        """
        self.oauth_client = oauth_client
        self.service = service
        self._access_token = None
        self._refresh_token = None
        self._expires_at = None

        # Initialize database if not provided
        if db is None:
            from .db import Database

            self.db = Database()
        else:
            self.db = db

        self._load_tokens()

    def get_access_token(self) -> str:
        """
        Get valid access token, refreshing if necessary

        Returns:
            Valid access token
        """
        if self._access_token and self._is_token_valid():
            return self._access_token

        if self._refresh_token:
            try:
                self._refresh_access_token()
                return self._access_token
            except Exception as e:
                logger.warning(f"Token refresh failed: {e}")

        # Need to re-authenticate
        logger.info("Re-authentication required")
        self._authenticate()
        return self._access_token

    def authenticate(self) -> str:
        """
        Force re-authentication and return access token

        Returns:
            New access token
        """
        self._authenticate()
        return self._access_token

    def _authenticate(self):
        """Perform full OAuth authentication"""
        access_token, refresh_token = self.oauth_client.authenticate()

        self._access_token = access_token
        self._refresh_token = refresh_token
        # Assume 1 hour expiry if not provided
        self._expires_at = time.time() + 3600

        self._save_tokens()

    def _refresh_access_token(self):
        """Refresh the access token"""
        new_access_token = self.oauth_client.refresh_token(self._refresh_token)

        self._access_token = new_access_token
        # Assume 1 hour expiry if not provided
        self._expires_at = time.time() + 3600

        self._save_tokens()

    def _is_token_valid(self) -> bool:
        """Check if current access token is valid"""
        if not self._access_token or not self._expires_at:
            return False

        # Consider token expired 5 minutes before actual expiry
        return time.time() < (self._expires_at - 300)

    def _load_tokens(self):
        """Load tokens from database"""
        try:
            tokens = self.db.get_oauth_tokens(self.service)

            if tokens:
                self._access_token = tokens.get("access_token")
                self._refresh_token = tokens.get("refresh_token")
                self._expires_at = tokens.get("expires_at")

                logger.info(f"OAuth tokens loaded from database for {self.service}")
            else:
                logger.info(f"No OAuth tokens found in database for {self.service}")

        except Exception as e:
            logger.error(f"Error loading tokens from database: {e}")

    def _save_tokens(self):
        """Save tokens to database"""
        try:
            success = self.db.save_oauth_tokens(
                service=self.service,
                access_token=self._access_token,
                refresh_token=self._refresh_token,
                expires_at=self._expires_at,
                token_type="Bearer",
                scope="https://www.googleapis.com/auth/youtube.readonly",
            )

            if success:
                logger.info(f"OAuth tokens saved to database for {self.service}")
            else:
                logger.error(
                    f"Failed to save OAuth tokens to database for {self.service}"
                )

        except Exception as e:
            logger.error(f"Error saving tokens to database: {e}")

    def clear_tokens(self):
        """Clear stored tokens (for logout/reset)"""
        try:
            self.db.delete_oauth_tokens(self.service)
            self._access_token = None
            self._refresh_token = None
            self._expires_at = None
            logger.info(f"OAuth tokens cleared for {self.service}")
        except Exception as e:
            logger.error(f"Error clearing tokens: {e}")
