#!/usr/bin/env python3
"""Publish a local diagnostic offer approval/rejection event."""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from uuid import uuid4

import aio_pika
from aio_pika import ExchangeType
from aio_pika.abc import AbstractRobustConnection

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.config import get_settings  # noqa: E402


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("decision", choices=("approved", "rejected"))
    parser.add_argument("offer_id", type=int)
    parser.add_argument("deal_id", type=int)
    parser.add_argument(
        "--reason",
        default="Предложение отклонено руководителем",
        help="Причина отклонения для rejected event",
    )
    return parser.parse_args()


async def publish(args: argparse.Namespace) -> None:
    settings = get_settings()
    connection: AbstractRobustConnection | None = None
    event_id = f"event_{uuid4().hex}"
    routing_key = f"event.offer.{args.decision}"
    payload: dict[str, str | int] = {
        "offer_id": args.offer_id,
        "deal_id": args.deal_id,
    }
    if args.decision == "rejected":
        payload["reason"] = args.reason

    event = {
        "event_id": event_id,
        "event_type": f"offer.{args.decision}",
        "occurred_at": datetime.now(timezone.utc).isoformat(),
        "payload": payload,
    }

    try:
        connection = await aio_pika.connect_robust(
            settings.rabbitmq_url.get_secret_value()
        )
        channel = await connection.channel()
        exchange = await channel.declare_exchange(
            settings.rabbitmq_exchange,
            ExchangeType.TOPIC,
            durable=True,
            passive=True,
        )
        await exchange.publish(
            aio_pika.Message(
                body=json.dumps(event, ensure_ascii=False).encode("utf-8"),
                content_type="application/json",
                delivery_mode=aio_pika.DeliveryMode.PERSISTENT,
            ),
            routing_key=routing_key,
        )
        print(f"Published {routing_key}: event_id={event_id}")
    finally:
        if connection is not None and not connection.is_closed:
            await connection.close()


if __name__ == "__main__":
    asyncio.run(publish(parse_args()))
