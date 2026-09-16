#!/usr/bin/env python3
"""Publish a local diagnostic construction delay event."""

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

ROUTING_KEY = "event.construction.delay_detected"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("building_id", type=int)
    parser.add_argument("--delay-days", type=int, required=True)
    parser.add_argument(
        "--risk-level",
        choices=("low", "medium", "high"),
        required=True,
    )
    args = parser.parse_args()
    if args.delay_days <= 0:
        parser.error("--delay-days must be greater than zero")
    return args


async def publish(args: argparse.Namespace) -> None:
    settings = get_settings()
    connection: AbstractRobustConnection | None = None
    event_id = f"event_{uuid4().hex}"
    event = {
        "event_id": event_id,
        "event_type": "construction.delay_detected",
        "occurred_at": datetime.now(timezone.utc).isoformat(),
        "payload": {
            "building_id": args.building_id,
            "delay_days": args.delay_days,
            "risk_level": args.risk_level,
        },
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
                body=json.dumps(event).encode("utf-8"),
                content_type="application/json",
                delivery_mode=aio_pika.DeliveryMode.PERSISTENT,
            ),
            routing_key=ROUTING_KEY,
        )
        print(f"Published {ROUTING_KEY}: event_id={event_id}")
    finally:
        if connection is not None and not connection.is_closed:
            await connection.close()


if __name__ == "__main__":
    asyncio.run(publish(parse_args()))
