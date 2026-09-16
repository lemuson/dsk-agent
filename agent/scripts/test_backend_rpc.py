#!/usr/bin/env python3
"""Call one backend RPC through the real RabbitMQ path."""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
import time
from pathlib import Path

import aio_pika
from aio_pika import ExchangeType
from aio_pika.abc import AbstractRobustConnection

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.broker.backend_rpc import BackendRpcClient, BackendRpcError  # noqa: E402
from app.config import get_settings  # noqa: E402
from app.schemas.backend import BackendResponse  # noqa: E402
from app.tools.apartment import ApartmentTools  # noqa: E402
from app.tools.building import BuildingTools  # noqa: E402
from app.tools.competitor import CompetitorTools  # noqa: E402
from app.tools.construction import ConstructionTools  # noqa: E402
from app.tools.deal import DealTools  # noqa: E402


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="operation", required=True)

    deal = subparsers.add_parser("deal.get", help="Get a deal")
    deal.add_argument("--deal-id", type=int, required=True)

    apartment = subparsers.add_parser("apartment.get", help="Get an apartment")
    apartment.add_argument("--apartment-id", type=int, required=True)

    building = subparsers.add_parser("building.get", help="Get a building")
    building.add_argument("--building-id", type=int, required=True)

    competitor = subparsers.add_parser(
        "competitor.list",
        help="List competitors by district",
    )
    competitor.add_argument("--district", required=True)

    construction = subparsers.add_parser(
        "construction.events.get",
        help="Get construction risk events for a building",
    )
    construction.add_argument("--building-id", type=int, required=True)
    return parser.parse_args()


def print_response(response: BackendResponse) -> None:
    print(json.dumps(response.model_dump(mode="json"), ensure_ascii=False, indent=2))


async def call_selected(args: argparse.Namespace, rpc: BackendRpcClient) -> BackendResponse:
    if args.operation == "deal.get":
        return await DealTools(rpc).get_deal(args.deal_id)
    if args.operation == "apartment.get":
        return await ApartmentTools(rpc).get_apartment(args.apartment_id)
    if args.operation == "building.get":
        return await BuildingTools(rpc).get_building(args.building_id)
    if args.operation == "competitor.list":
        return await CompetitorTools(rpc).list_competitors(args.district)
    if args.operation == "construction.events.get":
        return await ConstructionTools(rpc).get_construction_events(args.building_id)
    raise ValueError(f"Unsupported operation: {args.operation}")


async def run(args: argparse.Namespace) -> int:
    settings = get_settings()
    started_at = time.perf_counter()
    connection: AbstractRobustConnection | None = None
    rpc: BackendRpcClient | None = None

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
        rpc = BackendRpcClient(channel, exchange)
        await rpc.start()

        print_response(await call_selected(args, rpc))
        return 0
    except BackendRpcError as exc:
        print(
            f"Backend RPC error: operation={args.operation} "
            f"code={exc.code or 'UNKNOWN'} message={exc}",
            file=sys.stderr,
        )
        return 1
    except Exception as exc:
        print(
            f"Backend RPC diagnostic failed: {type(exc).__name__}. "
            "Check RabbitMQ and backend availability.",
            file=sys.stderr,
        )
        return 1
    finally:
        if rpc is not None:
            await rpc.close()
        if connection is not None and not connection.is_closed:
            await connection.close()
        print(f"elapsed: {time.perf_counter() - started_at:.2f} s")


if __name__ == "__main__":
    try:
        raise SystemExit(asyncio.run(run(parse_args())))
    except KeyboardInterrupt:
        print("Backend RPC diagnostic interrupted by user.", file=sys.stderr)
        raise SystemExit(130) from None
