#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from workflow_common import clean_status, git_run, head_sha, ok, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="执行 sync-back 前的线性化与前置检查")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--source-branch")
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    branch = str(repo.get("branch") or "")
    source_branch = args.source_branch or branch
    primary_root = Path(str(repo["primary_root"])).resolve()
    if not clean_status(primary_root):
        raise SystemExit(ok(status="blocked", reason="主目录本地 main 必须保持干净后才能进行 sync-back。"))
    if not source_branch:
        raise SystemExit(ok(status="blocked", reason="缺少来源分支，无法执行 sync-back。"))
    main_head_before = head_sha(primary_root, "main")
    source_head_before = head_sha(cwd)
    git_run(cwd, "rebase", "main")
    rebased_head = head_sha(cwd)
    main_head_after = main_head_before
    ff_applied = False
    if args.apply:
        raise SystemExit(ok(status="blocked", reason="当前受控入口尚未提供主目录 main 前进的 apply 路径；在 rebase-only 规则下，禁止通过 git merge --ff-only 推进主目录。"))
    raise SystemExit(
        ok(
            action="sync-back-preflight",
            source_branch=source_branch,
            source_head_before=source_head_before,
            main_head_before=main_head_before,
            rebased_head=rebased_head,
            main_head_after=main_head_after,
            ff_applied=ff_applied,
            next_step="线性化已完成；若后续要真正前进主目录 main、退出 worktree 并清理对象，应继续走仓库定义的后续步骤。",
        )
    )


if __name__ == "__main__":
    main()
