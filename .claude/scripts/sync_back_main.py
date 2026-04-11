#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from feature_meta import load_meta as load_feature_meta, mark_sync_status as mark_feature_sync_status
from integration_meta import load_meta as load_integration_meta, mark_sync_status as mark_integration_sync_status
from workflow_common import (
    clean_status,
    classify_ownership,
    commit_count_between,
    feature_ready_to_sync,
    head_sha,
    integration_ready_to_sync,
    is_integration_branch,
    merge_commit_count_between,
    ok,
    resolve_repo,
)
from workflow_common import git_run


def main() -> int:
    parser = argparse.ArgumentParser(description="执行 rebase + fast-forward 的 sync-back")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--source-branch")
    parser.add_argument("--owner-token", default="")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    branch = str(repo.get("branch") or "")
    source_branch = args.source_branch or branch
    primary_root = Path(str(repo["primary_root"])).resolve()
    owner_token = args.owner_token.strip() or None

    if not source_branch:
        raise SystemExit(ok(status="blocked", reason="缺少来源分支，无法执行 sync-back。"))
    if not clean_status(cwd):
        raise SystemExit(ok(status="blocked", reason="来源 worktree 必须保持干净后才能进行 sync-back。"))
    if not clean_status(primary_root):
        raise SystemExit(ok(status="blocked", reason="主目录本地 main 必须保持干净后才能进行 sync-back。"))

    if is_integration_branch(source_branch):
        integration_id = source_branch.removeprefix("integration-")
        meta = load_integration_meta(str(primary_root), integration_id)
        if meta is None:
            raise SystemExit(ok(status="blocked", reason=f"integration metadata 不存在: {integration_id}"))
        if classify_ownership(meta, owner_token) != "owned":
            raise SystemExit(ok(status="blocked", reason="integration 对象归属不明，不能自动 sync-back。"))
        if not integration_ready_to_sync(meta):
            raise SystemExit(ok(status="blocked", reason="integration gate 尚未 ready_to_sync。"))
    else:
        meta = load_feature_meta(str(primary_root), branch=source_branch)
        if meta is None:
            raise SystemExit(ok(status="blocked", reason=f"feature metadata 不存在: {source_branch}"))
        if classify_ownership(meta, owner_token) != "owned":
            raise SystemExit(ok(status="blocked", reason="feature 对象归属不明，不能自动 sync-back。"))
        if not feature_ready_to_sync(meta):
            raise SystemExit(ok(status="blocked", reason="feature 质量门尚未全部通过，不能自动 sync-back。"))

    commits_ahead = commit_count_between(cwd, "main")
    merge_commits = merge_commit_count_between(cwd, "main")
    if commits_ahead <= 0:
        raise SystemExit(ok(status="blocked", reason="来源分支相对 main 没有可同步的提交。", commits_ahead=commits_ahead))
    if merge_commits > 0:
        raise SystemExit(
            ok(
                status="blocked",
                reason=f"来源分支包含 {merge_commits} 个 merge commit；请先整理为线性历史后再 sync-back。",
                commits_ahead=commits_ahead,
                merge_commits=merge_commits,
            )
        )
    if is_integration_branch(source_branch) and commits_ahead != 1:
        raise SystemExit(
            ok(
                status="blocked",
                reason=f"integration 分支有 {commits_ahead} 个 commit（需要 1 个）；请先 squash 为单个 commit 后再 sync-back。",
                commits_ahead=commits_ahead,
                merge_commits=merge_commits,
            )
        )

    main_head_before = head_sha(primary_root, "main")
    source_head_before = head_sha(cwd)

    git_run(cwd, "rebase", "main")
    rebased_head = head_sha(cwd)
    git_run(primary_root, "rebase", source_branch)
    main_head_after = head_sha(primary_root, "main")

    if is_integration_branch(source_branch):
        mark_integration_sync_status(str(primary_root), source_branch.removeprefix("integration-"), "synced", "synced to local main", main_head_after)
    else:
        mark_feature_sync_status(str(primary_root), "synced", branch=source_branch, sync_summary="synced to local main", synced_main_sha=main_head_after)

    raise SystemExit(
        ok(
            action="sync-back-main",
            source_branch=source_branch,
            source_kind=("integration" if is_integration_branch(source_branch) else "feature"),
            source_head_before=source_head_before,
            main_head_before=main_head_before,
            commits_ahead=commits_ahead,
            merge_commits=merge_commits,
            rebased_head=rebased_head,
            main_head_after=main_head_after,
            next_step="主目录 main 已沿线性历史前进到当前结果；若当前会话仍驻留在该 worktree 中，请继续 ExitWorktree，并在主目录执行 cleanup。",
        )
    )


if __name__ == "__main__":
    main()
