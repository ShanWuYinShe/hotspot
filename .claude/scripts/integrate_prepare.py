#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from workflow_common import ok, prepare_integration_worktree


def main() -> int:
    parser = argparse.ArgumentParser(description="准备 integration worktree 并初始化 metadata")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--integration-id", required=True)
    parser.add_argument("--branch")
    parser.add_argument("--worktree-path")
    parser.add_argument("--feature", dest="features", action="append", default=[])
    parser.add_argument("--refresh", action="store_true")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    payload = prepare_integration_worktree(
        cwd=cwd,
        integration_id=args.integration_id,
        branch=args.branch,
        worktree_path=args.worktree_path,
        features=list(args.features),
        create_if_missing=args.refresh,
    )
    if args.refresh:
        payload["next_step"] = "集成范围和 worktree 已准备好；如需实际 replay/cherry-pick 与冲突处理，请在 Bash 模式下逐步执行。"
        action = "refresh-integrate"
    else:
        action = "start-integrate"
    raise SystemExit(ok(action=action, **payload))


if __name__ == "__main__":
    main()
