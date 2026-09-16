#!/usr/bin/env python3
"""Run all dialog UX flows sequentially through RabbitMQ RPC."""

from __future__ import annotations

import argparse
import asyncio
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from scripts.rpc_client import (  # noqa: E402
    DiagnosticRpcClient,
    DiagnosticRpcResponseError,
    DiagnosticRpcTimeoutError,
    positive_timeout,
    print_result,
)

SELECTED_TEXT = "У конкурента дешевле, я не уверен, что стоит переплачивать"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("deal_id", type=int, help="Integer deal ID")
    parser.add_argument("user_id", type=int, help="Integer manager/user ID")
    parser.add_argument(
        "--timeout",
        type=positive_timeout,
        default=30.0,
        help="Timeout for each RPC step in seconds (default: 30)",
    )
    return parser.parse_args()


async def run(args: argparse.Namespace) -> int:
    common_payload = {
        "deal_id": args.deal_id,
        "user_id": args.user_id,
    }
    steps = (
        (
            "STEP 1 — Анализ диалога",
            "agent.dialog.analyze",
            "dialog.analyze",
            common_payload,
        ),
        (
            "STEP 2 — Помочь ответить на последнее сообщение клиента",
            "agent.dialog.reply_assist",
            "dialog.reply_assist",
            common_payload,
        ),
        (
            "STEP 3 — Помочь ответить на выбранный фрагмент",
            "agent.dialog.reply_assist",
            "dialog.reply_assist",
            {**common_payload, "selected_text": SELECTED_TEXT},
        ),
    )
    started_at = time.perf_counter()

    try:
        async with DiagnosticRpcClient() as client:
            for title, routing_key, action, payload in steps:
                print(f"\n{'=' * 72}\n{title}\n{'=' * 72}")
                result = await client.call(
                    routing_key=routing_key,
                    action=action,
                    payload=payload,
                    timeout=args.timeout,
                )
                print_result(result)
                if not result.response["success"]:
                    print(f"Demo stopped: {title} returned success=false.", file=sys.stderr)
                    return 1
    except DiagnosticRpcTimeoutError as exc:
        print(f"Demo stopped by RPC timeout: {exc}", file=sys.stderr)
        return 1
    except DiagnosticRpcResponseError as exc:
        print(f"Demo stopped by invalid RPC response: {exc}", file=sys.stderr)
        return 1
    except Exception as exc:
        # Never print connection settings because the URL may contain credentials.
        print(
            f"Dialog demo failed: {type(exc).__name__}. "
            "Check RabbitMQ, mock backend, Agent Service and environment settings.",
            file=sys.stderr,
        )
        return 1
    finally:
        print(f"\ntotal elapsed: {time.perf_counter() - started_at:.2f} s")

    print("\nAll dialog demo steps completed successfully.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(asyncio.run(run(parse_args())))
    except KeyboardInterrupt:
        print("Dialog demo interrupted.", file=sys.stderr)
        raise SystemExit(130) from None
