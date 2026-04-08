#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from workflow_common import git_run, ok, prepare_integration_worktree


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
    integration_worktree = Path(str(payload["integration_worktree"]))
    replay_results = []
    for feature in args.features:
        try:
            git_run(integration_worktree, "cherry-pick", feature)
            replay_results.append({"source_branch": feature, "status": "applied"})
        except RuntimeError as exc:
            replay_results.append({"source_branch": feature, "status": "conflict", "reason": str(exc)})
            raise SystemExit(ok(status="blocked", action="refresh-integrate" if args.refresh else "start-integrate", **payload, replay_results=replay_results, next_step="当前 replay 发生冲突；请根据输出逐步解决冲突后再继续。"))
    action = "refresh-integrate" if args.refresh else "start-integrate"
    raise SystemExit(ok(action=action, **payload, replay_results=replay_results))


if __name__ == "__main__":
    main()
