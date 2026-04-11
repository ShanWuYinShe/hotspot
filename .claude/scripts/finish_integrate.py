#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import load_meta, mark_gate_status, stale
from workflow_common import commit_count_between, git_output, head_sha, integration_ready_to_sync, ok, require_integration, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="执行 integration 历史线性化基础检查并回写 gate 候选状态")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--integration-id")
    parser.add_argument("--gate-evidence-path", default="")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_integration(repo)
    branch = str(repo.get("branch") or "")
    integration_id = args.integration_id or branch.removeprefix("integration-")
    primary_root = Path(str(repo["primary_root"])).resolve()
    current_main_sha = head_sha(primary_root, "main")
    meta = load_meta(str(primary_root), integration_id)
    if meta is None:
        raise SystemExit(ok(status="blocked", reason=f"integration metadata 不存在: {integration_id}"))

    commits_ahead = commit_count_between(cwd, "main")
    merge_commits = int(git_output(cwd, "rev-list", "--count", "--merges", f"main..{branch}") or "0")
    baseline_stale = stale(str(primary_root), meta, current_main_sha)
    verify_passed = meta.get("verify_status") == "passed"
    review_passed = meta.get("review_status") == "passed"
    simplify_passed = meta.get("simplify_status") == "passed"
    replay_applied = meta.get("replay_status") == "applied" and all(item.get("replay_status") == "applied" for item in meta.get("planned_features", []))

    if baseline_stale:
        gate_status = "must_refresh"
        gate_summary = "integration 基线已过期，必须先 refresh。"
    elif not replay_applied:
        gate_status = "blocked"
        gate_summary = "planned features 尚未全部 replay 完整。"
    elif not verify_passed or not review_passed or not simplify_passed:
        gate_status = "blocked"
        gate_summary = "integration 尚未满足 verify/simplify/review 条件。"
    elif merge_commits > 0:
        gate_status = "blocked"
        gate_summary = f"integration 存在 {merge_commits} 个 merge commit，必须先整理为纯线性历史。"
    elif commits_ahead != 1:
        gate_status = "blocked"
        gate_summary = f"integration 当前有 {commits_ahead} 个 commit（需要 1 个），必须先 squash 为单个 commit 再进入 sync-back。"
    else:
        gate_status = "ready_to_sync"
        gate_summary = "integration 已满足 verify/simplify/review/replay 与线性历史要求，可进入 sync-back。"

    meta_path = mark_gate_status(str(primary_root), integration_id, gate_status, gate_summary, args.gate_evidence_path)
    raise SystemExit(
        ok(
            action="finish-integrate",
            integration_branch=branch,
            integration_id=integration_id,
            commits_ahead_of_main=commits_ahead,
            merge_commits=merge_commits,
            baseline_stale=baseline_stale,
            replay_applied=replay_applied,
            verify_passed=verify_passed,
            simplify_passed=simplify_passed,
            review_passed=review_passed,
            gate_status=gate_status,
            metadata_path=meta_path,
            ready_to_sync=integration_ready_to_sync({**meta, "gate_status": gate_status}),
            next_step=("当前基础 gate 已通过，可继续进入 /sync_back_main。" if gate_status == "ready_to_sync" else ("当前 integration 需要先 refresh。" if gate_status == "must_refresh" else f"当前 integration 仍未满足最终 gate 条件：{gate_summary}")),
        )
    )


if __name__ == "__main__":
    main()
