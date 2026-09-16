from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from app.config import Settings
from app.agents.router import RouteDecision
from app.llm.gigachat import GigaChatClient


def make_settings() -> Settings:
    return Settings(
        rabbitmq_url="amqp://guest:guest@localhost/",
        gigachat_credentials="secret",
    )


@pytest.mark.asyncio
async def test_generate_uses_async_gigachat_api() -> None:
    client = GigaChatClient(make_settings())
    create = AsyncMock(
        return_value=SimpleNamespace(
            messages=[
                SimpleNamespace(
                    content=[SimpleNamespace(text="Первая "), SimpleNamespace(text="часть")]
                )
            ]
        )
    )
    client._client = SimpleNamespace(achat=SimpleNamespace(create=create))  # type: ignore[assignment]
    client._started = True

    result = await client.generate("Вопрос")

    assert result == "Первая часть"
    create.assert_awaited_once_with("Вопрос")


@pytest.mark.asyncio
async def test_generate_passes_separate_system_prompt() -> None:
    client = GigaChatClient(make_settings())
    create = AsyncMock(
        return_value=SimpleNamespace(
            messages=[SimpleNamespace(content=[SimpleNamespace(text="Ответ")])]
        )
    )
    client._client = SimpleNamespace(achat=SimpleNamespace(create=create))  # type: ignore[assignment]
    client._started = True

    result = await client.generate("Вопрос", system_prompt="Системные правила")

    assert result == "Ответ"
    create.assert_awaited_once_with(
        {
            "messages": [
                {"role": "system", "content": "Системные правила"},
                {"role": "user", "content": "Вопрос"},
            ]
        }
    )


@pytest.mark.asyncio
async def test_empty_gigachat_response_is_an_error() -> None:
    client = GigaChatClient(make_settings())
    create = AsyncMock(return_value=SimpleNamespace(messages=[]))
    client._client = SimpleNamespace(achat=SimpleNamespace(create=create))  # type: ignore[assignment]
    client._started = True

    with pytest.raises(RuntimeError, match="empty response"):
        await client.generate("Вопрос")


@pytest.mark.asyncio
async def test_parse_structured_uses_native_gigachat_api() -> None:
    client = GigaChatClient(make_settings())
    decision = RouteDecision(
        agent="negotiation",
        intent="handle_objection",
        confidence=0.9,
    )
    parse = AsyncMock(return_value=(SimpleNamespace(), decision))
    client._client = SimpleNamespace(achat=SimpleNamespace(parse=parse))  # type: ignore[assignment]
    client._started = True

    result = await client.parse_structured("Маршрутизируй", RouteDecision)

    assert result == decision
    parse.assert_awaited_once_with(
        "Маршрутизируй",
        response_format=RouteDecision,
        strict=True,
    )
