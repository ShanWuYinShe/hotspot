#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import mark_gate_status
from workflow_common import git_output, ok, require_integration, resolve_repo


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
    commits_ahead = int(git_output(cwd, "rev-list", "--count", f"main..{branch}") or "0")
    merge_commits = int(git_output(cwd, "rev-list", "--count", "--merges", f"main..{branch}") or "0")
    gate_status = "ready_to_sync" if commits_ahead == 1 and merge_commits == 0 else "blocked"
    gate_summary = "integration 已收敛为单个非 merge commit，可进入下一步。" if gate_status == "ready_to_sync" else "integration 尚未满足单 commit 且无 merge commit 的 gate 条件。"
    meta_path = mark_gate_status(str(Path(str(repo['primary_root'])).resolve()), integration_id, gate_status, gate_summary)
    raise SystemExit(
        ok(
            action="finish-integrate",
            integration_branch=branch,
            integration_id=integration_id,
            commits_ahead_of_main=commits_ahead,
            merge_commits=merge_commits,
            gate_status=gate_status,
            metadata_path=meta_path,
            next_step=("当前历史线性化检查通过；如需继续完成最终 gate，请结合 verify/review 结果人工确认或在 Bash 模式下继续后续步骤。" if gate_status == "ready_to_sync" else "当前仅完成基础线性化检查；仍需继续整理状态后再评估是否 ready to sync。"),
        )
    )


if __name__ == "__main__":
    main()
