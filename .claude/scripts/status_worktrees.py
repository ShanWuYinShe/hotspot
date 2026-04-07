#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import list_meta, stale
from workflow_common import classify_worktree, clean_status, fail, head_sha, list_worktrees, now_iso, ok, parse_feature_lineage, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="汇总当前主目录、feature lineage 与 integration 状态")
    parser.add_argument("--cwd", default=".")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    primary_root = Path(str(repo["primary_root"])).resolve()
    main_clean = clean_status(primary_root)
    main_sha = head_sha(primary_root, "main")
    items = list_worktrees(primary_root)

    lineage_latest: dict[str, int] = {}
    parsed: list[dict[str, object]] = []
    for item in items:
        branch = item.get("branch", "")
        short_branch = branch.removeprefix("refs/heads/")
        role = classify_worktree(short_branch)
        entry: dict[str, object] = {
            "path": item.get("worktree", ""),
            "branch": short_branch,
            "role": role,
        }
        if role == "feature":
            root, generation = parse_feature_lineage(short_branch)
            lineage_latest[root] = max(lineage_latest.get(root, 0), generation)
            entry["lineage_root"] = root
            entry["lineage_generation"] = generation
        parsed.append(entry)

    for entry in parsed:
        if entry.get("role") == "feature":
            root = str(entry["lineage_root"])
            generation = int(entry["lineage_generation"])
            latest = lineage_latest[root]
            entry["lineage_status"] = "active-latest" if generation == latest else "superseded"

    integration_items = list_meta(str(primary_root))
    integration_state = []
    for item in integration_items:
        integration_state.append(
            {
                "integration_id": item["integration_id"],
                "integration_branch": item["integration_branch"],
                "baseline_main_sha": item["baseline_main_sha"],
                "current_main_sha": main_sha,
                "derived_state": "stale" if stale(str(primary_root), item, main_sha) else "active",
                "gate_status": item.get("gate_status", "pending"),
            }
        )

    active_feature_candidates = [
        {
            "branch": str(entry["branch"]),
            "lineage_status": entry.get("lineage_status", "n/a"),
        }
        for entry in parsed
        if entry.get("role") == "feature" and entry.get("lineage_status") == "active-latest"
    ]

    recommended_next_steps = ["/start_feature"]
    notes: list[str] = []
    if any(item["role"] == "integration" for item in parsed):
        recommended_next_steps = ["/verify_integrate", "/finish_integrate"]
    elif active_feature_candidates:
        recommended_next_steps = ["/start_feature", "/start_integrate", "/cleanup_worktrees"]

    stale_integrations = [item for item in integration_state if item["derived_state"] == "stale"]
    if stale_integrations:
        notes.append("检测到历史 integration metadata 已过期；这只是状态提示，不自动覆盖当前正常开发主路径。仅当你明确要继续该 integration 时，才考虑 /refresh_integrate。")

    return ok(
        generated_at=now_iso(),
        main={
            "path": str(primary_root),
            "branch": "main",
            "clean": main_clean,
            "head": main_sha,
        },
        worktrees=parsed,
        integrations=integration_state,
        active_feature_candidates=active_feature_candidates,
        recommended_next_steps=recommended_next_steps,
        notes=notes,
    )


if __name__ == "__main__":
    raise SystemExit(main())
