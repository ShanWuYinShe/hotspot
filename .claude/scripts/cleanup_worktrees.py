#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from workflow_common import classify_worktree, clean_status, git_output, git_run, head_sha, list_worktrees, ok, require_main_primary, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="盘点并执行 worktree cleanup")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--force", action="store_true", help="允许对未完全合并的分支使用 -D 强制删除")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_main_primary(repo)
    items = list_worktrees(cwd)
    main_head = head_sha(cwd, "main")
    candidates = []
    merged = git_output(cwd, "branch", "--merged", "main")
    merged_set = {line.strip().lstrip("*+ ").strip() for line in merged.splitlines() if line.strip()}
    for item in items:
        branch = item.get("branch", "").removeprefix("refs/heads/")
        role = classify_worktree(branch)
        if role == "main":
            continue
        worktree_path = item.get("worktree", "")
        merged_to_main = branch in merged_set if branch else False
        branch_tip = ""
        if branch:
            try:
                branch_tip = head_sha(cwd, branch)
            except RuntimeError:
                pass
        tip_equals_main = branch_tip == main_head if branch_tip else False
        blocked_reason = ""
        removed = False
        branch_deleted = False
        deletion_method = ""
        if merged_to_main and args.apply:
            if Path(worktree_path).resolve() == cwd.resolve():
                blocked_reason = "当前会话仍驻留在该 worktree 中，请先 ExitWorktree。"
            else:
                try:
                    git_run(cwd, "worktree", "remove", worktree_path)
                except RuntimeError as exc:
                    blocked_reason = str(exc)
                if not blocked_reason:
                    removed = True
                    if tip_equals_main:
                        # 标准 sync-back 后来源分支 tip == main tip，-d 必然成功
                        git_run(cwd, "branch", "-d", branch)
                        deletion_method = "-d (synced)"
                        branch_deleted = True
                    else:
                        try:
                            git_run(cwd, "branch", "-d", branch)
                            deletion_method = "-d"
                            branch_deleted = True
                        except RuntimeError:
                            if args.force:
                                git_run(cwd, "branch", "-D", branch)
                                deletion_method = "-D (forced)"
                                branch_deleted = True
                            else:
                                deletion_method = "blocked: 需要 --force 才能强制删除"
        candidates.append(
            {
                "path": worktree_path,
                "branch": branch,
                "role": role,
                "merged_to_main": merged_to_main,
                "tip_equals_main": tip_equals_main,
                "cleanup_candidate": merged_to_main,
                "blocked_reason": blocked_reason,
                "removed": removed,
                "branch_deleted": branch_deleted,
                "deletion_method": deletion_method,
            }
        )
    raise SystemExit(ok(action="cleanup-worktrees", main_clean=clean_status(cwd), candidates=candidates))


if __name__ == "__main__":
    main()
