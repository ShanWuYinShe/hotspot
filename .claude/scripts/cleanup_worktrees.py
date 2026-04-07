#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from workflow_common import classify_worktree, clean_status, git_output, git_run, list_worktrees, ok, require_main_primary, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="盘点 worktree cleanup 候选并给出手动清理建议")
    parser.add_argument("--cwd", default=".")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_main_primary(repo)
    items = list_worktrees(cwd)
    candidates = []
    merged = git_output(cwd, "branch", "--merged", "main")
    merged_set = {line.strip().lstrip("* ").strip() for line in merged.splitlines() if line.strip()}
    for item in items:
        branch = item.get("branch", "").removeprefix("refs/heads/")
        role = classify_worktree(branch)
        if role == "main":
            continue
        worktree_path = item.get("worktree", "")
        merged_to_main = branch in merged_set if branch else False
        blocked_reason = ""
        next_step = ""
        if merged_to_main:
            if Path(worktree_path).resolve() == cwd.resolve():
                blocked_reason = "当前会话仍驻留在该 worktree 中，请先 ExitWorktree。"
            else:
                next_step = f"可在 Bash 模式下逐步执行: git worktree remove {worktree_path}，再观察结果后决定是否删除分支 {branch}。"
        candidates.append(
            {
                "path": worktree_path,
                "branch": branch,
                "role": role,
                "merged_to_main": merged_to_main,
                "cleanup_candidate": merged_to_main,
                "blocked_reason": blocked_reason,
                "next_step": next_step,
            }
        )
    raise SystemExit(ok(action="cleanup-worktrees", main_clean=clean_status(cwd), candidates=candidates))


if __name__ == "__main__":
    main()
