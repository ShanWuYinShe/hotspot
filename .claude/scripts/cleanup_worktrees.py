#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from feature_meta import delete_meta as delete_feature_meta, load_meta as load_feature_meta, mark_cleanup_status as mark_feature_cleanup_status
from integration_meta import delete_meta as delete_integration_meta, load_meta as load_integration_meta, mark_cleanup_status as mark_integration_cleanup_status
from workflow_common import classify_ownership, classify_worktree, clean_status, cleanup_eligibility, git_output, git_run, head_sha, list_worktrees, ok, require_main_primary, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="盘点并执行 worktree cleanup")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--owner-token", default="")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_main_primary(repo)
    owner_token = args.owner_token.strip() or None
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
        target_clean = clean_status(Path(worktree_path)) if worktree_path else False
        blocked_reason = ""
        removed = False
        branch_deleted = False
        deletion_method = ""
        ownership_state = "takeover_required"
        cleanup_state = "keep"

        if role == "feature":
            meta = load_feature_meta(str(cwd), branch=branch)
            ownership_state = classify_ownership(meta, owner_token)
            cleanup_state = cleanup_eligibility(meta, tip_equals_main=tip_equals_main, clean=target_clean, owner_token=owner_token)
        elif role == "integration":
            meta = load_integration_meta(str(cwd), branch.removeprefix("integration-"))
            ownership_state = classify_ownership(meta, owner_token)
            cleanup_state = cleanup_eligibility(meta, tip_equals_main=tip_equals_main, clean=target_clean, owner_token=owner_token)
        else:
            meta = None
            cleanup_state = "needs confirmation"

        if cleanup_state == "safe to cleanup" and args.apply:
            if Path(worktree_path).resolve() == cwd.resolve():
                blocked_reason = "当前会话仍驻留在该 worktree 中，请先 ExitWorktree。"
            else:
                try:
                    git_run(cwd, "worktree", "remove", worktree_path)
                except RuntimeError as exc:
                    blocked_reason = str(exc)
                if not blocked_reason:
                    removed = True
                    try:
                        git_run(cwd, "branch", "-d", branch)
                        deletion_method = "-d"
                        branch_deleted = True
                    except RuntimeError as exc:
                        blocked_reason = str(exc)
                    if branch_deleted:
                        if role == "feature" and meta is not None:
                            mark_feature_cleanup_status(str(cwd), "cleaned", branch=branch, cleanup_summary="worktree and branch cleaned")
                            delete_feature_meta(str(cwd), branch.removeprefix("worktree-"))
                        elif role == "integration" and meta is not None:
                            integration_id = branch.removeprefix("integration-")
                            mark_integration_cleanup_status(str(cwd), integration_id, "cleaned", "worktree and branch cleaned")
                            delete_integration_meta(str(cwd), integration_id)
        elif meta is not None:
            summary = blocked_reason or cleanup_state
            recorded_status = "kept" if cleanup_state != "safe to cleanup" or not args.apply else "cleaned"
            if role == "feature":
                mark_feature_cleanup_status(str(cwd), recorded_status, branch=branch, cleanup_summary=summary)
            elif role == "integration":
                mark_integration_cleanup_status(str(cwd), branch.removeprefix("integration-"), recorded_status, summary)

        candidates.append(
            {
                "path": worktree_path,
                "branch": branch,
                "role": role,
                "merged_to_main": merged_to_main,
                "tip_equals_main": tip_equals_main,
                "clean": target_clean,
                "ownership_state": ownership_state,
                "cleanup_state": cleanup_state,
                "blocked_reason": blocked_reason,
                "removed": removed,
                "branch_deleted": branch_deleted,
                "deletion_method": deletion_method,
            }
        )
    raise SystemExit(ok(action="cleanup-worktrees", main_clean=clean_status(cwd), candidates=candidates))


if __name__ == "__main__":
    main()
