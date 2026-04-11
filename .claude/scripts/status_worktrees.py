#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import stale
from workflow_common import (
    classify_ownership,
    classify_worktree,
    cleanup_eligibility,
    clean_status,
    collect_feature_runtime,
    collect_integration_runtime,
    head_sha,
    list_worktrees,
    now_iso,
    ok,
    parse_feature_lineage,
    resolve_repo,
)


def main() -> int:
    parser = argparse.ArgumentParser(description="汇总当前主目录、feature lineage 与 integration 状态")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--owner-token", default="")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    primary_root = Path(str(repo["primary_root"])).resolve()
    owner_token = args.owner_token.strip() or None
    main_clean = clean_status(primary_root)
    main_sha = head_sha(primary_root, "main")
    feature_runtime = collect_feature_runtime(primary_root)
    integration_runtime = collect_integration_runtime(primary_root)
    items = list_worktrees(primary_root)

    lineage_latest: dict[str, int] = {}
    parsed: list[dict[str, object]] = []
    for item in items:
        branch = item.get("branch", "")
        short_branch = branch.removeprefix("refs/heads/")
        role = classify_worktree(short_branch)
        worktree_path = str(item.get("worktree", ""))
        clean = clean_status(Path(worktree_path)) if worktree_path else False
        entry: dict[str, object] = {
            "path": worktree_path,
            "branch": short_branch,
            "role": role,
            "clean": clean,
        }
        if role == "feature":
            root, generation = parse_feature_lineage(short_branch)
            lineage_latest[root] = max(lineage_latest.get(root, 0), generation)
            meta = feature_runtime.get(short_branch)
            entry["lineage_root"] = root
            entry["lineage_generation"] = generation
            entry["runtime"] = meta or {}
            entry["ownership_state"] = classify_ownership(meta, owner_token)
        elif role == "integration":
            meta = integration_runtime.get(short_branch)
            entry["runtime"] = meta or {}
            entry["ownership_state"] = classify_ownership(meta, owner_token)
        parsed.append(entry)

    for entry in parsed:
        if entry.get("role") == "feature":
            root = str(entry["lineage_root"])
            generation = int(entry["lineage_generation"])
            latest = lineage_latest[root]
            entry["lineage_status"] = "active-latest" if generation == latest else "superseded"
            meta = dict(entry.get("runtime") or {})
            tip_equals_main = str(meta.get("synced_main_sha") or "") == main_sha and meta.get("sync_status") == "synced"
            entry["cleanup_state"] = cleanup_eligibility(meta, tip_equals_main=tip_equals_main, clean=bool(entry["clean"]), owner_token=owner_token)
        elif entry.get("role") == "integration":
            meta = dict(entry.get("runtime") or {})
            baseline_stale = stale(str(primary_root), meta, main_sha) if meta else False
            entry["derived_state"] = "stale" if baseline_stale else "active"
            tip_equals_main = str(meta.get("synced_main_sha") or "") == main_sha and meta.get("sync_status") == "synced"
            entry["cleanup_state"] = cleanup_eligibility(meta, tip_equals_main=tip_equals_main, clean=bool(entry["clean"]), owner_token=owner_token)

    integration_state = []
    for branch, item in integration_runtime.items():
        integration_state.append(
            {
                "integration_id": item["integration_id"],
                "integration_branch": branch,
                "baseline_main_sha": item["baseline_main_sha"],
                "current_main_sha": main_sha,
                "derived_state": "stale" if stale(str(primary_root), item, main_sha) else "active",
                "gate_status": item.get("gate_status", "pending"),
                "ownership_state": classify_ownership(item, owner_token),
                "cleanup_state": cleanup_eligibility(
                    item,
                    tip_equals_main=str(item.get("synced_main_sha") or "") == main_sha and item.get("sync_status") == "synced",
                    clean=True,
                    owner_token=owner_token,
                ),
            }
        )

    active_feature_candidates = [
        {
            "branch": str(entry["branch"]),
            "lineage_status": entry.get("lineage_status", "n/a"),
            "ownership_state": entry.get("ownership_state", "takeover_required"),
            "sync_status": dict(entry.get("runtime") or {}).get("sync_status", "pending"),
        }
        for entry in parsed
        if entry.get("role") == "feature" and entry.get("lineage_status") == "active-latest"
    ]

    recommended_next_steps = ["/start_feature"]
    notes: list[str] = []
    if any(item["derived_state"] == "stale" for item in integration_state):
        notes.append("检测到 integration metadata 已过期；仅当你明确要继续该 integration 时，才考虑 /refresh_integrate。")
    if any(item.get("role") == "integration" for item in parsed):
        recommended_next_steps = ["/verify_integrate", "/finish_integrate", "/cleanup_worktrees"]
    elif active_feature_candidates:
        recommended_next_steps = ["/start_feature", "/start_integrate", "/cleanup_worktrees"]

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
