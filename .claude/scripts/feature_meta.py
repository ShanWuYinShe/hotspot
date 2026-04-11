#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import sys
from datetime import datetime, timezone

SCHEMA_VERSION = 1

OWNERSHIP_STATES = {"owned", "foreign", "takeover_required"}
STEP_STATUSES = {"pending", "passed", "failed", "stale", "skipped"}
COMMIT_STATUSES = {"pending", "ready", "blocked"}
SYNC_STATUSES = {"pending", "synced", "blocked"}
CLEANUP_STATUSES = {"pending", "cleaned", "kept", "blocked"}

META_REQUIRED_FIELDS = (
    "schema_version",
    "feature_id",
    "lineage_root",
    "generation",
    "feature_branch",
    "feature_worktree_path",
    "baseline_main_sha",
    "task_summary",
    "owner_token",
    "ownership_state",
    "check_status",
    "test_status",
    "simplify_status",
    "review_status",
    "verify_status",
    "commit_status",
    "sync_status",
    "cleanup_status",
    "verify_summary",
    "review_summary",
    "sync_summary",
    "cleanup_summary",
    "last_commit_sha",
    "synced_main_sha",
    "verify_evidence_path",
    "review_evidence_path",
    "created_at",
    "updated_at",
)


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def meta_dir(main_root: str) -> str:
    return os.path.join(main_root, "tmp", "claude", "features")


def meta_path(main_root: str, feature_id: str) -> str:
    return os.path.join(meta_dir(main_root), f"{feature_id}.json")


def ensure_meta_dir(main_root: str) -> str:
    os.makedirs(meta_dir(main_root), exist_ok=True)
    return meta_dir(main_root)


def parse_feature_id(feature_id: str) -> tuple[str, int]:
    if feature_id.endswith("-r2") or "-r" in feature_id:
        head, _, maybe = feature_id.rpartition("-r")
        if head and maybe.isdigit() and int(maybe) >= 2:
            return head, int(maybe)
    return feature_id, 1


def feature_id_from_branch(branch: str) -> str:
    return branch.removeprefix("worktree-")


def resolve_feature_id(feature_id: str | None = None, branch: str | None = None) -> str:
    if feature_id:
        return feature_id
    if branch:
        return feature_id_from_branch(branch)
    raise ValueError("missing feature_id or branch")


def normalize_simple_status(value: str, allowed: set[str], field_name: str) -> str:
    if value not in allowed:
        raise ValueError(f"invalid {field_name}: {value}")
    return value


def normalize_meta(data: dict[str, object] | None) -> dict[str, object]:
    normalized = dict(data or {})
    feature_id = str(normalized.get("feature_id") or "")
    branch = str(normalized.get("feature_branch") or "")
    if not feature_id and branch:
        feature_id = feature_id_from_branch(branch)
        normalized["feature_id"] = feature_id
    if feature_id:
        lineage_root, generation = parse_feature_id(feature_id)
        normalized.setdefault("lineage_root", lineage_root)
        normalized.setdefault("generation", generation)

    if "ownership_state" in normalized:
        normalized["ownership_state"] = normalize_simple_status(
            str(normalized["ownership_state"]), OWNERSHIP_STATES, "ownership_state"
        )

    for key in ("check_status", "test_status", "simplify_status", "review_status", "verify_status"):
        if key in normalized:
            normalized[key] = normalize_simple_status(str(normalized[key]), STEP_STATUSES, key)

    if "commit_status" in normalized:
        normalized["commit_status"] = normalize_simple_status(
            str(normalized["commit_status"]), COMMIT_STATUSES, "commit_status"
        )

    if "sync_status" in normalized:
        normalized["sync_status"] = normalize_simple_status(str(normalized["sync_status"]), SYNC_STATUSES, "sync_status")

    if "cleanup_status" in normalized:
        normalized["cleanup_status"] = normalize_simple_status(
            str(normalized["cleanup_status"]), CLEANUP_STATUSES, "cleanup_status"
        )

    return normalized


def validate_meta(data: dict[str, object]) -> dict[str, object]:
    missing = [field for field in META_REQUIRED_FIELDS if field not in data]
    if missing:
        raise ValueError(f"feature metadata missing required fields: {', '.join(missing)}")
    return normalize_meta(data)


def default_meta(
    *,
    feature_id: str,
    feature_branch: str,
    feature_worktree_path: str,
    baseline_main_sha: str,
    task_summary: str,
    owner_token: str = "",
) -> dict[str, object]:
    lineage_root, generation = parse_feature_id(feature_id)
    stamp = now_iso()
    return {
        "schema_version": SCHEMA_VERSION,
        "feature_id": feature_id,
        "lineage_root": lineage_root,
        "generation": generation,
        "feature_branch": feature_branch,
        "feature_worktree_path": feature_worktree_path,
        "baseline_main_sha": baseline_main_sha,
        "task_summary": task_summary,
        "owner_token": owner_token,
        "ownership_state": "owned" if owner_token else "takeover_required",
        "check_status": "pending",
        "test_status": "pending",
        "simplify_status": "pending",
        "review_status": "pending",
        "verify_status": "pending",
        "commit_status": "pending",
        "sync_status": "pending",
        "cleanup_status": "pending",
        "verify_summary": "",
        "review_summary": "",
        "sync_summary": "",
        "cleanup_summary": "",
        "last_commit_sha": "",
        "synced_main_sha": "",
        "verify_evidence_path": "",
        "review_evidence_path": "",
        "created_at": stamp,
        "updated_at": stamp,
    }


def load_meta(main_root: str, feature_id: str | None = None, branch: str | None = None) -> dict[str, object] | None:
    resolved_feature_id = resolve_feature_id(feature_id, branch)
    path = meta_path(main_root, resolved_feature_id)
    if not os.path.exists(path):
        return None
    try:
        with open(path, "r", encoding="utf-8") as f:
            return normalize_meta(json.load(f))
    except Exception:
        return None


def list_meta(main_root: str) -> list[dict[str, object]]:
    directory = meta_dir(main_root)
    if not os.path.isdir(directory):
        return []
    items: list[dict[str, object]] = []
    for name in sorted(os.listdir(directory)):
        if not name.endswith(".json"):
            continue
        item = load_meta(main_root, feature_id=name[:-5])
        if item is not None:
            items.append(item)
    return items


def write_meta(main_root: str, feature_id: str, data: dict[str, object]) -> str:
    ensure_meta_dir(main_root)
    path = meta_path(main_root, feature_id)
    normalized = validate_meta(normalize_meta(data))
    with open(path, "w", encoding="utf-8") as f:
        json.dump(normalized, f, ensure_ascii=False, indent=2)
        f.write("\n")
    return path


def delete_meta(main_root: str, feature_id: str) -> str:
    path = meta_path(main_root, feature_id)
    if os.path.exists(path):
        os.remove(path)
    return path


def init_meta(
    main_root: str,
    *,
    feature_branch: str,
    feature_worktree_path: str,
    baseline_main_sha: str,
    task_summary: str,
    owner_token: str = "",
) -> str:
    feature_id = feature_id_from_branch(feature_branch)
    data = default_meta(
        feature_id=feature_id,
        feature_branch=feature_branch,
        feature_worktree_path=feature_worktree_path,
        baseline_main_sha=baseline_main_sha,
        task_summary=task_summary,
        owner_token=owner_token,
    )
    return write_meta(main_root, feature_id, data)


def update_meta(main_root: str, feature_id: str | None = None, branch: str | None = None, **patch: object) -> str:
    resolved_feature_id = resolve_feature_id(feature_id, branch)
    data = load_meta(main_root, feature_id=resolved_feature_id)
    if data is None:
        raise ValueError(f"feature metadata does not exist: {resolved_feature_id}")
    data.update(normalize_meta(patch))
    data["updated_at"] = now_iso()
    return write_meta(main_root, resolved_feature_id, data)


def mark_step_status(
    main_root: str,
    *,
    field_name: str,
    status: str,
    feature_id: str | None = None,
    branch: str | None = None,
    summary_field: str | None = None,
    summary: str = "",
    evidence_field: str | None = None,
    evidence_path: str = "",
    extra: dict[str, object] | None = None,
) -> str:
    patch: dict[str, object] = {field_name: status}
    if summary_field is not None:
        patch[summary_field] = summary
    if evidence_field is not None:
        patch[evidence_field] = evidence_path
    if extra:
        patch.update(extra)
    return update_meta(main_root, feature_id=feature_id, branch=branch, **patch)


def mark_check_status(main_root: str, status: str, feature_id: str | None = None, branch: str | None = None) -> str:
    return mark_step_status(main_root, field_name="check_status", status=status, feature_id=feature_id, branch=branch)


def mark_test_status(main_root: str, status: str, feature_id: str | None = None, branch: str | None = None) -> str:
    return mark_step_status(main_root, field_name="test_status", status=status, feature_id=feature_id, branch=branch)


def mark_simplify_status(main_root: str, status: str, feature_id: str | None = None, branch: str | None = None) -> str:
    return mark_step_status(main_root, field_name="simplify_status", status=status, feature_id=feature_id, branch=branch)


def mark_review_status(
    main_root: str,
    status: str,
    feature_id: str | None = None,
    branch: str | None = None,
    review_summary: str = "",
    review_evidence_path: str = "",
) -> str:
    return mark_step_status(
        main_root,
        field_name="review_status",
        status=status,
        feature_id=feature_id,
        branch=branch,
        summary_field="review_summary",
        summary=review_summary,
        evidence_field="review_evidence_path",
        evidence_path=review_evidence_path,
    )


def mark_verify_status(
    main_root: str,
    status: str,
    feature_id: str | None = None,
    branch: str | None = None,
    verify_summary: str = "",
    verify_evidence_path: str = "",
) -> str:
    return mark_step_status(
        main_root,
        field_name="verify_status",
        status=status,
        feature_id=feature_id,
        branch=branch,
        summary_field="verify_summary",
        summary=verify_summary,
        evidence_field="verify_evidence_path",
        evidence_path=verify_evidence_path,
    )


def mark_commit_status(
    main_root: str,
    status: str,
    feature_id: str | None = None,
    branch: str | None = None,
    last_commit_sha: str = "",
) -> str:
    return mark_step_status(
        main_root,
        field_name="commit_status",
        status=status,
        feature_id=feature_id,
        branch=branch,
        extra={"last_commit_sha": last_commit_sha},
    )


def mark_sync_status(
    main_root: str,
    status: str,
    feature_id: str | None = None,
    branch: str | None = None,
    sync_summary: str = "",
    synced_main_sha: str = "",
) -> str:
    return mark_step_status(
        main_root,
        field_name="sync_status",
        status=status,
        feature_id=feature_id,
        branch=branch,
        summary_field="sync_summary",
        summary=sync_summary,
        extra={"synced_main_sha": synced_main_sha},
    )


def mark_cleanup_status(
    main_root: str,
    status: str,
    feature_id: str | None = None,
    branch: str | None = None,
    cleanup_summary: str = "",
) -> str:
    return mark_step_status(
        main_root,
        field_name="cleanup_status",
        status=status,
        feature_id=feature_id,
        branch=branch,
        summary_field="cleanup_summary",
        summary=cleanup_summary,
    )


def claim(main_root: str, owner_token: str, feature_id: str | None = None, branch: str | None = None) -> str:
    return update_meta(main_root, feature_id=feature_id, branch=branch, owner_token=owner_token, ownership_state="owned")


def _print_json(payload: dict[str, object]) -> None:
    json.dump(payload, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")


def _cmd_list(args: argparse.Namespace) -> int:
    _print_json({"status": "ok", "items": list_meta(args.main_root)})
    return 0


def _cmd_show(args: argparse.Namespace) -> int:
    item = load_meta(args.main_root, feature_id=args.feature_id, branch=args.branch)
    if item is None:
        _print_json({"status": "blocked", "reason": "feature metadata does not exist"})
        return 1
    _print_json({"status": "ok", "item": item})
    return 0


def _cmd_init(args: argparse.Namespace) -> int:
    path = init_meta(
        args.main_root,
        feature_branch=args.feature_branch,
        feature_worktree_path=args.feature_worktree_path,
        baseline_main_sha=args.baseline_main_sha,
        task_summary=args.task_summary,
        owner_token=args.owner_token,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_check(args: argparse.Namespace) -> int:
    path = mark_check_status(args.main_root, args.status, feature_id=args.feature_id, branch=args.branch)
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_test(args: argparse.Namespace) -> int:
    path = mark_test_status(args.main_root, args.status, feature_id=args.feature_id, branch=args.branch)
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_simplify(args: argparse.Namespace) -> int:
    path = mark_simplify_status(args.main_root, args.status, feature_id=args.feature_id, branch=args.branch)
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_review(args: argparse.Namespace) -> int:
    path = mark_review_status(
        args.main_root,
        args.status,
        feature_id=args.feature_id,
        branch=args.branch,
        review_summary=args.review_summary,
        review_evidence_path=args.review_evidence_path,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_verify(args: argparse.Namespace) -> int:
    path = mark_verify_status(
        args.main_root,
        args.status,
        feature_id=args.feature_id,
        branch=args.branch,
        verify_summary=args.verify_summary,
        verify_evidence_path=args.verify_evidence_path,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_commit(args: argparse.Namespace) -> int:
    path = mark_commit_status(
        args.main_root,
        args.status,
        feature_id=args.feature_id,
        branch=args.branch,
        last_commit_sha=args.last_commit_sha,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_sync(args: argparse.Namespace) -> int:
    path = mark_sync_status(
        args.main_root,
        args.status,
        feature_id=args.feature_id,
        branch=args.branch,
        sync_summary=args.sync_summary,
        synced_main_sha=args.synced_main_sha,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_cleanup(args: argparse.Namespace) -> int:
    path = mark_cleanup_status(
        args.main_root,
        args.status,
        feature_id=args.feature_id,
        branch=args.branch,
        cleanup_summary=args.cleanup_summary,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_claim(args: argparse.Namespace) -> int:
    path = claim(args.main_root, args.owner_token, feature_id=args.feature_id, branch=args.branch)
    _print_json({"status": "ok", "path": path})
    return 0


def _add_target_args(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--feature-id")
    parser.add_argument("--branch")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="feature 元信息统一读写入口")
    subparsers = parser.add_subparsers(dest="command", required=True)

    parser_list = subparsers.add_parser("list")
    parser_list.add_argument("--main-root", required=True)
    parser_list.set_defaults(func=_cmd_list)

    parser_show = subparsers.add_parser("show")
    parser_show.add_argument("--main-root", required=True)
    _add_target_args(parser_show)
    parser_show.set_defaults(func=_cmd_show)

    parser_init = subparsers.add_parser("init")
    parser_init.add_argument("--main-root", required=True)
    parser_init.add_argument("--feature-branch", required=True)
    parser_init.add_argument("--feature-worktree-path", required=True)
    parser_init.add_argument("--baseline-main-sha", required=True)
    parser_init.add_argument("--task-summary", default="")
    parser_init.add_argument("--owner-token", default="")
    parser_init.set_defaults(func=_cmd_init)

    parser_check = subparsers.add_parser("mark-check")
    parser_check.add_argument("--main-root", required=True)
    parser_check.add_argument("--status", required=True)
    _add_target_args(parser_check)
    parser_check.set_defaults(func=_cmd_mark_check)

    parser_test = subparsers.add_parser("mark-test")
    parser_test.add_argument("--main-root", required=True)
    parser_test.add_argument("--status", required=True)
    _add_target_args(parser_test)
    parser_test.set_defaults(func=_cmd_mark_test)

    parser_simplify = subparsers.add_parser("mark-simplify")
    parser_simplify.add_argument("--main-root", required=True)
    parser_simplify.add_argument("--status", required=True)
    _add_target_args(parser_simplify)
    parser_simplify.set_defaults(func=_cmd_mark_simplify)

    parser_review = subparsers.add_parser("mark-review")
    parser_review.add_argument("--main-root", required=True)
    parser_review.add_argument("--status", required=True)
    parser_review.add_argument("--review-summary", default="")
    parser_review.add_argument("--review-evidence-path", default="")
    _add_target_args(parser_review)
    parser_review.set_defaults(func=_cmd_mark_review)

    parser_verify = subparsers.add_parser("mark-verify")
    parser_verify.add_argument("--main-root", required=True)
    parser_verify.add_argument("--status", required=True)
    parser_verify.add_argument("--verify-summary", default="")
    parser_verify.add_argument("--verify-evidence-path", default="")
    _add_target_args(parser_verify)
    parser_verify.set_defaults(func=_cmd_mark_verify)

    parser_commit = subparsers.add_parser("mark-commit")
    parser_commit.add_argument("--main-root", required=True)
    parser_commit.add_argument("--status", required=True)
    parser_commit.add_argument("--last-commit-sha", default="")
    _add_target_args(parser_commit)
    parser_commit.set_defaults(func=_cmd_mark_commit)

    parser_sync = subparsers.add_parser("mark-sync")
    parser_sync.add_argument("--main-root", required=True)
    parser_sync.add_argument("--status", required=True)
    parser_sync.add_argument("--sync-summary", default="")
    parser_sync.add_argument("--synced-main-sha", default="")
    _add_target_args(parser_sync)
    parser_sync.set_defaults(func=_cmd_mark_sync)

    parser_cleanup = subparsers.add_parser("mark-cleanup")
    parser_cleanup.add_argument("--main-root", required=True)
    parser_cleanup.add_argument("--status", required=True)
    parser_cleanup.add_argument("--cleanup-summary", default="")
    _add_target_args(parser_cleanup)
    parser_cleanup.set_defaults(func=_cmd_mark_cleanup)

    parser_claim = subparsers.add_parser("claim")
    parser_claim.add_argument("--main-root", required=True)
    parser_claim.add_argument("--owner-token", required=True)
    _add_target_args(parser_claim)
    parser_claim.set_defaults(func=_cmd_claim)

    return parser


if __name__ == "__main__":
    parser = build_parser()
    args = parser.parse_args()
    try:
        raise SystemExit(args.func(args))
    except Exception as exc:
        _print_json({"status": "blocked", "reason": str(exc)})
        raise SystemExit(1)
