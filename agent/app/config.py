from functools import lru_cache

from pydantic import Field, SecretStr
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables and .env file."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
        populate_by_name=True,
    )

    app_name: str = Field(default="agent-service", validation_alias="APP_NAME")
    log_level: str = Field(default="INFO", validation_alias="LOG_LEVEL")

    rabbitmq_url: SecretStr = Field(validation_alias="RABBITMQ_URL")
    rabbitmq_exchange: str = Field(
        default="app.topic",
        validation_alias="RABBITMQ_EXCHANGE",
    )
    rabbitmq_queue: str = Field(
        default="agent-service",
        validation_alias="RABBITMQ_QUEUE",
    )
    rabbitmq_routing_key: str = Field(
        default="agent.chat.request",
        validation_alias="RABBITMQ_ROUTING_KEY",
    )
    rabbitmq_prefetch_count: int = Field(
        default=10,
        ge=1,
        validation_alias="RABBITMQ_PREFETCH_COUNT",
    )

    gigachat_credentials: SecretStr = Field(
        validation_alias="GIGACHAT_CREDENTIALS"
    )
    gigachat_scope: str = Field(
        default="GIGACHAT_API_PERS",
        validation_alias="GIGACHAT_SCOPE",
    )
    gigachat_model: str = Field(
        default="GigaChat",
        validation_alias="GIGACHAT_MODEL",
    )
    gigachat_verify_ssl_certs: bool = Field(
        default=True,
        validation_alias="GIGACHAT_VERIFY_SSL_CERTS",
    )
    gigachat_ca_bundle_file: str | None = Field(
        default=None,
        validation_alias="GIGACHAT_CA_BUNDLE_FILE",
    )
    gigachat_timeout: float = Field(
        default=60.0,
        gt=0,
        validation_alias="GIGACHAT_TIMEOUT",
    )


@lru_cache
def get_settings() -> Settings:
    return Settings()