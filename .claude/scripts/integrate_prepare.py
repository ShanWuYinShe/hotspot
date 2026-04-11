#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import mark_replay_status
from workflow_common import ensure_owner_token, git_run, head_sha, ok, prepare_integration_worktree


def main() -> int:
    parser = argparse.ArgumentParser(description="准备 integration worktree 并初始化 metadata")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--integration-id", required=True)
    parser.add_argument("--branch")
    parser.add_argument("--worktree-path")
    parser.add_argument("--feature", dest="features", action="append", default=[])
    parser.add_argument("--refresh", action="store_true")
    parser.add_argument("--owner-token", default="")
    parser.add_argument("--source-integration-id", default="")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    owner_token = ensure_owner_token(args.owner_token)
    payload = prepare_integration_worktree(
        cwd=cwd,
        integration_id=args.integration_id,
        branch=args.branch,
        worktree_path=args.worktree_path,
        features=list(args.features),
        create_if_missing=args.refresh,
        owner_token=owner_token,
        source_integration_id=args.source_integration_id,
    )
    integration_worktree = Path(str(payload["integration_worktree"]))
    replay_results = []
    planned_features = list(payload["planned_features"])
    for index, feature in enumerate(args.features):
        try:
            git_run(integration_worktree, "cherry-pick", feature)
            replayed_head_sha = head_sha(integration_worktree)
            replay_results.append({"source_branch": feature, "status": "applied", "replayed_head_sha": replayed_head_sha})
            planned_features[index]["replay_status"] = "applied"
            planned_features[index]["replay_summary"] = "replay applied"
            planned_features[index]["replayed_head_sha"] = replayed_head_sha
        except RuntimeError as exc:
            replay_results.append({"source_branch": feature, "status": "conflict", "reason": str(exc)})
            planned_features[index]["replay_status"] = "conflict"
            planned_features[index]["replay_summary"] = str(exc)
            mark_replay_status(str(cwd), args.integration_id, "conflict", str(exc), planned_features=planned_features)
            raise SystemExit(ok(status="blocked", action="refresh-integrate" if args.refresh else "start-integrate", **payload, replay_results=replay_results, next_step="当前 replay 发生冲突；请根据输出逐步解决冲突后再继续。"))

    mark_replay_status(str(cwd), args.integration_id, "applied", "all planned features replayed", planned_features=planned_features)
    action = "refresh-integrate" if args.refresh else "start-integrate"
    raise SystemExit(ok(action=action, **payload, replay_results=replay_results))


if __name__ == "__main__":
    main()
