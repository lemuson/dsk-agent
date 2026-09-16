import logging
from typing import TypeVar

from gigachat import GigaChat
from pydantic import BaseModel

from app.config import Settings

logger = logging.getLogger(__name__)

ModelT = TypeVar("ModelT", bound=BaseModel)


class GigaChatClient:
    def __init__(self, settings: Settings) -> None:
        client_options: dict[str, object] = {
            "credentials": settings.gigachat_credentials.get_secret_value(),
            "scope": settings.gigachat_scope,
            "model": settings.gigachat_model,
            "verify_ssl_certs": settings.gigachat_verify_ssl_certs,
            "timeout": settings.gigachat_timeout,
        }
        if settings.gigachat_ca_bundle_file:
            client_options["ca_bundle_file"] = settings.gigachat_ca_bundle_file

        self._client = GigaChat(**client_options)
        self._started = False

    async def start(self) -> None:
        if not self._started:
            await self._client.__aenter__()
            self._started = True
            logger.info("GigaChat client started")

    async def close(self) -> None:
        if self._started:
            await self._client.__aexit__(None, None, None)
            self._started = False
            logger.info("GigaChat client closed")

    async def generate(
        self,
        message: str,
        *,
        system_prompt: str | None = None,
    ) -> str:
        if not self._started:
            raise RuntimeError("GigaChat client is not started")

        payload: str | dict[str, object] = message
        if system_prompt is not None:
            payload = {
                "messages": [
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": message},
                ]
            }

        response = await self._client.achat.create(payload)
        content = "".join(
            part.text or ""
            for response_message in response.messages
            for part in (response_message.content or [])
        ).strip()
        if not content:
            raise RuntimeError("GigaChat returned an empty response")
        return content

    async def parse_structured(
        self,
        prompt: str,
        response_model: type[ModelT],
    ) -> ModelT:
        if not self._started:
            raise RuntimeError("GigaChat client is not started")

        _, parsed = await self._client.achat.parse(
            prompt,
            response_format=response_model,
            strict=True,
        )
        return parsed
