#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from workflow_common import clean_status, git_run, head_sha, ok, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="执行纯 rebase 路径的 sync-back")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--source-branch")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    branch = str(repo.get("branch") or "")
    source_branch = args.source_branch or branch
    primary_root = Path(str(repo["primary_root"])).resolve()

    if not source_branch:
        raise SystemExit(ok(status="blocked", reason="缺少来源分支，无法执行 sync-back。"))
    if not clean_status(cwd):
        raise SystemExit(ok(status="blocked", reason="来源 worktree 必须保持干净后才能进行 sync-back。"))
    if not clean_status(primary_root):
        raise SystemExit(ok(status="blocked", reason="主目录本地 main 必须保持干净后才能进行 sync-back。"))

    main_head_before = head_sha(primary_root, "main")
    source_head_before = head_sha(cwd)

    git_run(cwd, "rebase", "main")
    rebased_head = head_sha(cwd)
    git_run(primary_root, "rebase", source_branch)
    main_head_after = head_sha(primary_root, "main")

    raise SystemExit(
        ok(
            action="sync-back-main",
            source_branch=source_branch,
            source_head_before=source_head_before,
            main_head_before=main_head_before,
            rebased_head=rebased_head,
            main_head_after=main_head_after,
            next_step="主目录 main 已通过纯 rebase 路径前进到当前结果；若当前会话仍驻留在该 worktree 中，请继续 ExitWorktree，并在主目录执行 cleanup。",
        )
    )


if __name__ == "__main__":
    main()
