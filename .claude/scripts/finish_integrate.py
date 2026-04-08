#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import load_meta, mark_gate_status, stale
from workflow_common import git_output, head_sha, ok, require_integration, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="执行 integration 历史线性化基础检查并回写 gate 候选状态")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--integration-id")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_integration(repo)
    branch = str(repo.get("branch") or "")
    integration_id = args.integration_id or branch.removeprefix("integration-")
    primary_root = Path(str(repo['primary_root'])).resolve()
    current_main_sha = head_sha(primary_root, "main")
    meta = load_meta(str(primary_root), integration_id)
    if meta is None:
        raise SystemExit(ok(status="blocked", reason=f"integration metadata 不存在: {integration_id}"))
    commits_ahead = int(git_output(cwd, "rev-list", "--count", f"main..{branch}") or "0")
    merge_commits = int(git_output(cwd, "rev-list", "--count", "--merges", f"main..{branch}") or "0")
    baseline_stale = stale(str(primary_root), meta, current_main_sha)
    verify_passed = meta.get("verify_status") == "passed"
    review_passed = meta.get("review_status") == "passed"
    gate_status = "ready_to_sync" if commits_ahead == 1 and merge_commits == 0 and not baseline_stale and verify_passed and review_passed else ("must_refresh" if baseline_stale else "blocked")
    gate_summary = (
        "integration 已满足基础 gate 条件，可进入下一步。"
        if gate_status == "ready_to_sync"
        else ("integration 基线已过期，必须先 refresh。" if gate_status == "must_refresh" else "integration 尚未满足 verify/review/线性化等 gate 条件。")
    )
    meta_path = mark_gate_status(str(primary_root), integration_id, gate_status, gate_summary)
    raise SystemExit(
        ok(
            action="finish-integrate",
            integration_branch=branch,
            integration_id=integration_id,
            commits_ahead_of_main=commits_ahead,
            merge_commits=merge_commits,
            baseline_stale=baseline_stale,
            verify_passed=verify_passed,
            review_passed=review_passed,
            gate_status=gate_status,
            metadata_path=meta_path,
            next_step=("当前基础 gate 已通过，可继续进入下一步。" if gate_status == "ready_to_sync" else ("当前 integration 需要先 refresh。" if gate_status == "must_refresh" else "当前 integration 仍未满足最终 gate 条件，请先补齐 verify/review 或继续整理历史。")),
        )
    )


if __name__ == "__main__":
    main()
