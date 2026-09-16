#!/usr/bin/env python3
"""Run reply assist through the real RabbitMQ RPC path."""

from __future__ import annotations

import argparse
import asyncio
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from scripts.rpc_client import positive_timeout, run_single_call  # noqa: E402

ROUTING_KEY = "agent.dialog.reply_assist"
ACTION = "dialog.reply_assist"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("deal_id", type=int, help="Integer deal ID")
    parser.add_argument("user_id", type=int, help="Integer manager/user ID")
    parser.add_argument(
        "--selected-text",
        help="Selected client message fragment; omit to use the latest client message",
    )
    parser.add_argument(
        "--timeout",
        type=positive_timeout,
        default=30.0,
        help="RPC response timeout in seconds (default: 30)",
    )
    return parser.parse_args()


async def run(args: argparse.Namespace) -> int:
    payload = {
        "deal_id": args.deal_id,
        "user_id": args.user_id,
    }
    if args.selected_text is not None:
        payload["selected_text"] = args.selected_text

    return await run_single_call(
        routing_key=ROUTING_KEY,
        action=ACTION,
        payload=payload,
        timeout=args.timeout,
    )


if __name__ == "__main__":
    try:
        raise SystemExit(asyncio.run(run(parse_args())))
    except KeyboardInterrupt:
        print("Reply assist diagnostic interrupted.", file=sys.stderr)
        raise SystemExit(130) from None
