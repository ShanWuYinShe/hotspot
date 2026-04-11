import argparse
import json
import os
import sys
from datetime import datetime, timezone

SCHEMA_VERSION = 1

VERIFY_STATUSES = {"pending", "passed", "failed", "stale"}
REVIEW_STATUSES = {"pending", "passed", "failed", "stale"}
GATE_STATUSES = {"pending", "ready_to_sync", "blocked", "must_refresh"}
OWNERSHIP_STATES = {"owned", "foreign", "takeover_required"}
REPLAY_STATUSES = {"pending", "applied", "conflict", "stale"}
SIMPLIFY_STATUSES = {"pending", "passed", "failed", "stale"}
SYNC_STATUSES = {"pending", "synced", "blocked"}
CLEANUP_STATUSES = {"pending", "cleaned", "kept", "blocked"}

GATE_STATUS_ALIASES = {
    "pending": "pending",
    "ready to sync": "ready_to_sync",
    "ready_to_sync": "ready_to_sync",
    "blocked": "blocked",
    "must refresh": "must_refresh",
    "must_refresh": "must_refresh",
}

META_REQUIRED_FIELDS = (
    "schema_version",
    "integration_id",
    "baseline_main_sha",
    "integration_branch",
    "integration_worktree_path",
    "planned_features",
    "owner_token",
    "ownership_state",
    "replay_status",
    "replay_summary",
    "verify_status",
    "review_status",
    "simplify_status",
    "gate_status",
    "sync_status",
    "cleanup_status",
    "created_at",
    "updated_at",
    "verify_env_name",
    "verify_summary",
    "review_summary",
    "simplify_summary",
    "gate_summary",
    "sync_summary",
    "cleanup_summary",
    "synced_main_sha",
    "verify_evidence_path",
    "review_evidence_path",
    "gate_evidence_path",
    "source_integration_id",
)


def now_iso():
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def meta_dir(main_root):
    return os.path.join(main_root, "tmp", "claude", "integrations")


def meta_path(main_root, integration_id):
    return os.path.join(meta_dir(main_root), f"{integration_id}.json")


def ensure_meta_dir(main_root):
    os.makedirs(meta_dir(main_root), exist_ok=True)
    return meta_dir(main_root)


def normalize_list(value):
    if value is None:
        return []
    if isinstance(value, list):
        return value
    return [value]


def normalize_simple_status(value, allowed, field_name):
    if value not in allowed:
        raise ValueError(f"invalid {field_name}: {value}")
    return value


def normalize_gate_status(value):
    normalized = GATE_STATUS_ALIASES.get(value)
    if normalized is None:
        raise ValueError(f"invalid gate_status: {value}")
    return normalized


def default_planned_feature(source_branch="", source_head_sha="", replay_status="pending", replay_summary="", replayed_head_sha=""):
    return {
        "source_branch": source_branch,
        "source_head_sha": source_head_sha,
        "replayed_head_sha": replayed_head_sha,
        "replay_status": normalize_simple_status(replay_status, REPLAY_STATUSES, "planned_features.replay_status"),
        "replay_summary": replay_summary,
    }


def normalize_planned_features(value):
    items = normalize_list(value)
    normalized = []
    for item in items:
        if isinstance(item, dict):
            entry = dict(item)
            normalized.append(
                default_planned_feature(
                    source_branch=str(entry.get("source_branch") or ""),
                    source_head_sha=str(entry.get("source_head_sha") or ""),
                    replay_status=str(entry.get("replay_status") or "pending"),
                    replay_summary=str(entry.get("replay_summary") or ""),
                    replayed_head_sha=str(entry.get("replayed_head_sha") or ""),
                )
            )
        else:
            normalized.append(default_planned_feature(source_branch=str(item or "")))
    return normalized


def default_meta(
    integration_id,
    baseline_main_sha,
    integration_branch,
    integration_worktree_path,
    planned_features,
    *,
    owner_token="",
    source_integration_id="",
):
    stamp = now_iso()
    return {
        "schema_version": SCHEMA_VERSION,
        "integration_id": integration_id,
        "baseline_main_sha": baseline_main_sha,
        "integration_branch": integration_branch,
        "integration_worktree_path": integration_worktree_path,
        "planned_features": normalize_planned_features(planned_features),
        "owner_token": owner_token,
        "ownership_state": "owned" if owner_token else "takeover_required",
        "replay_status": "pending",
        "replay_summary": "",
        "verify_status": "pending",
        "review_status": "pending",
        "simplify_status": "pending",
        "gate_status": "pending",
        "sync_status": "pending",
        "cleanup_status": "pending",
        "created_at": stamp,
        "updated_at": stamp,
        "verify_env_name": "",
        "verify_summary": "",
        "review_summary": "",
        "simplify_summary": "",
        "gate_summary": "",
        "sync_summary": "",
        "cleanup_summary": "",
        "synced_main_sha": "",
        "verify_evidence_path": "",
        "review_evidence_path": "",
        "gate_evidence_path": "",
        "source_integration_id": source_integration_id,
    }


def upgrade_meta(data):
    if not data:
        return {}
    integration_id = str(data.get("integration_id") or "")
    baseline_main_sha = str(data.get("baseline_main_sha") or "")
    integration_branch = str(data.get("integration_branch") or "")
    integration_worktree_path = str(data.get("integration_worktree_path") or "")
    planned_features = data.get("planned_features") or []
    base = default_meta(
        integration_id,
        baseline_main_sha,
        integration_branch,
        integration_worktree_path,
        planned_features,
        owner_token=str(data.get("owner_token") or ""),
        source_integration_id=str(data.get("source_integration_id") or ""),
    )
    base.update(dict(data))
    return base


def normalize_meta(data):
    normalized = dict(upgrade_meta(data))

    if "planned_features" in normalized:
        normalized["planned_features"] = normalize_planned_features(normalized["planned_features"])

    if "verify_status" in normalized:
        normalized["verify_status"] = normalize_simple_status(
            str(normalized["verify_status"]), VERIFY_STATUSES, "verify_status"
        )

    if "review_status" in normalized:
        normalized["review_status"] = normalize_simple_status(
            str(normalized["review_status"]), REVIEW_STATUSES, "review_status"
        )

    if "simplify_status" in normalized:
        normalized["simplify_status"] = normalize_simple_status(
            str(normalized["simplify_status"]), SIMPLIFY_STATUSES, "simplify_status"
        )

    if "gate_status" in normalized:
        normalized["gate_status"] = normalize_gate_status(normalized["gate_status"])

    if "ownership_state" in normalized:
        normalized["ownership_state"] = normalize_simple_status(
            str(normalized["ownership_state"]), OWNERSHIP_STATES, "ownership_state"
        )

    if "replay_status" in normalized:
        normalized["replay_status"] = normalize_simple_status(
            str(normalized["replay_status"]), REPLAY_STATUSES, "replay_status"
        )

    if "sync_status" in normalized:
        normalized["sync_status"] = normalize_simple_status(
            str(normalized["sync_status"]), SYNC_STATUSES, "sync_status"
        )

    if "cleanup_status" in normalized:
        normalized["cleanup_status"] = normalize_simple_status(
            str(normalized["cleanup_status"]), CLEANUP_STATUSES, "cleanup_status"
        )

    normalized["schema_version"] = SCHEMA_VERSION
    return normalized


def validate_meta(data):
    normalized = normalize_meta(data)
    missing = [field for field in META_REQUIRED_FIELDS if field not in normalized]
    if missing:
        raise ValueError(f"integration metadata missing required fields: {', '.join(missing)}")
    return normalized


def load_meta(main_root, integration_id):
    path = meta_path(main_root, integration_id)
    if not os.path.exists(path):
        return None
    try:
        with open(path, "r", encoding="utf-8") as f:
            return validate_meta(json.load(f))
    except Exception:
        return None


def list_meta(main_root):
    directory = meta_dir(main_root)
    if not os.path.isdir(directory):
        return []
    items = []
    for name in sorted(os.listdir(directory)):
        if not name.endswith(".json"):
            continue
        integration_id = name[:-5]
        data = load_meta(main_root, integration_id)
        if data is not None:
            items.append(data)
    return items


def write_meta(main_root, integration_id, data):
    ensure_meta_dir(main_root)
    path = meta_path(main_root, integration_id)
    normalized = validate_meta(data)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(normalized, f, ensure_ascii=False, indent=2)
        f.write("\n")
    return path


def delete_meta(main_root, integration_id):
    path = meta_path(main_root, integration_id)
    if os.path.exists(path):
        os.remove(path)
    return path


def init_meta(
    main_root,
    integration_id,
    baseline_main_sha,
    integration_branch,
    integration_worktree_path,
    planned_features,
    *,
    owner_token="",
    source_integration_id="",
):
    data = default_meta(
        integration_id,
        baseline_main_sha,
        integration_branch,
        integration_worktree_path,
        planned_features,
        owner_token=owner_token,
        source_integration_id=source_integration_id,
    )
    return write_meta(main_root, integration_id, data)


def update_meta(main_root, integration_id, **patch):
    data = load_meta(main_root, integration_id)
    if data is None:
        raise ValueError(f"integration metadata does not exist: {integration_id}")
    data.update(normalize_meta(patch))
    data["updated_at"] = now_iso()
    return write_meta(main_root, integration_id, data)


def mark_verify_status(main_root, integration_id, verify_status, verify_env_name="", verify_summary="", verify_evidence_path=""):
    return update_meta(
        main_root,
        integration_id,
        verify_status=verify_status,
        verify_env_name=verify_env_name,
        verify_summary=verify_summary,
        verify_evidence_path=verify_evidence_path,
    )


def mark_review_status(main_root, integration_id, review_status, review_summary="", review_evidence_path=""):
    return update_meta(
        main_root,
        integration_id,
        review_status=review_status,
        review_summary=review_summary,
        review_evidence_path=review_evidence_path,
    )


def mark_simplify_status(main_root, integration_id, simplify_status, simplify_summary=""):
    return update_meta(
        main_root,
        integration_id,
        simplify_status=simplify_status,
        simplify_summary=simplify_summary,
    )


def mark_gate_status(main_root, integration_id, gate_status, gate_summary="", gate_evidence_path=""):
    return update_meta(
        main_root,
        integration_id,
        gate_status=gate_status,
        gate_summary=gate_summary,
        gate_evidence_path=gate_evidence_path,
    )


def mark_sync_status(main_root, integration_id, sync_status, sync_summary="", synced_main_sha=""):
    return update_meta(
        main_root,
        integration_id,
        sync_status=sync_status,
        sync_summary=sync_summary,
        synced_main_sha=synced_main_sha,
    )


def mark_cleanup_status(main_root, integration_id, cleanup_status, cleanup_summary=""):
    return update_meta(
        main_root,
        integration_id,
        cleanup_status=cleanup_status,
        cleanup_summary=cleanup_summary,
    )


def mark_replay_status(main_root, integration_id, replay_status, replay_summary="", planned_features=None):
    patch = {
        "replay_status": replay_status,
        "replay_summary": replay_summary,
    }
    if planned_features is not None:
        patch["planned_features"] = planned_features
    return update_meta(main_root, integration_id, **patch)


def claim(main_root, integration_id, owner_token):
    return update_meta(main_root, integration_id, owner_token=owner_token, ownership_state="owned")


def stale(main_root, data, current_main_sha):
    baseline = (data or {}).get("baseline_main_sha", "")
    return bool(baseline and current_main_sha and baseline != current_main_sha)


def _print_json(payload):
    json.dump(payload, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")


def _cmd_list(args):
    items = list_meta(args.main_root)
    _print_json({"status": "ok", "items": items})
    return 0


def _cmd_show(args):
    item = load_meta(args.main_root, args.integration_id)
    if item is None:
        _print_json({"status": "blocked", "reason": f"integration metadata does not exist: {args.integration_id}"})
        return 1
    _print_json({"status": "ok", "item": item})
    return 0


def _cmd_init(args):
    path = init_meta(
        args.main_root,
        args.integration_id,
        args.baseline_main_sha,
        args.integration_branch,
        args.integration_worktree_path,
        args.planned_features,
        owner_token=args.owner_token,
        source_integration_id=args.source_integration_id,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_verify(args):
    path = mark_verify_status(
        args.main_root,
        args.integration_id,
        args.verify_status,
        args.verify_env_name,
        args.verify_summary,
        args.verify_evidence_path,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_review(args):
    path = mark_review_status(
        args.main_root,
        args.integration_id,
        args.review_status,
        args.review_summary,
        args.review_evidence_path,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_simplify(args):
    path = mark_simplify_status(args.main_root, args.integration_id, args.simplify_status, args.simplify_summary)
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_gate(args):
    path = mark_gate_status(
        args.main_root,
        args.integration_id,
        args.gate_status,
        args.gate_summary,
        args.gate_evidence_path,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_sync(args):
    path = mark_sync_status(
        args.main_root,
        args.integration_id,
        args.sync_status,
        args.sync_summary,
        args.synced_main_sha,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_cleanup(args):
    path = mark_cleanup_status(args.main_root, args.integration_id, args.cleanup_status, args.cleanup_summary)
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_mark_replay(args):
    path = mark_replay_status(
        args.main_root,
        args.integration_id,
        args.replay_status,
        args.replay_summary,
        planned_features=args.planned_features,
    )
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_claim(args):
    path = claim(args.main_root, args.integration_id, args.owner_token)
    _print_json({"status": "ok", "path": path})
    return 0


def _cmd_is_stale(args):
    item = load_meta(args.main_root, args.integration_id)
    if item is None:
        _print_json({"status": "blocked", "reason": f"integration metadata does not exist: {args.integration_id}"})
        return 1
    _print_json(
        {
            "status": "ok",
            "integration_id": args.integration_id,
            "baseline_main_sha": item.get("baseline_main_sha", ""),
            "current_main_sha": args.current_main_sha,
            "stale": stale(args.main_root, item, args.current_main_sha),
        }
    )
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description="integration 元信息统一读写入口")
    subparsers = parser.add_subparsers(dest="command", required=True)

    parser_list = subparsers.add_parser("list")
    parser_list.add_argument("--main-root", required=True)
    parser_list.set_defaults(func=_cmd_list)

    parser_show = subparsers.add_parser("show")
    parser_show.add_argument("--main-root", required=True)
    parser_show.add_argument("--integration-id", required=True)
    parser_show.set_defaults(func=_cmd_show)

    parser_init = subparsers.add_parser("init")
    parser_init.add_argument("--main-root", required=True)
    parser_init.add_argument("--integration-id", required=True)
    parser_init.add_argument("--baseline-main-sha", required=True)
    parser_init.add_argument("--integration-branch", required=True)
    parser_init.add_argument("--integration-worktree-path", required=True)
    parser_init.add_argument("--planned-feature", dest="planned_features", action="append", default=[])
    parser_init.add_argument("--owner-token", default="")
    parser_init.add_argument("--source-integration-id", default="")
    parser_init.set_defaults(func=_cmd_init)

    parser_verify = subparsers.add_parser("mark-verify")
    parser_verify.add_argument("--main-root", required=True)
    parser_verify.add_argument("--integration-id", required=True)
    parser_verify.add_argument("--verify-status", required=True)
    parser_verify.add_argument("--verify-env-name", default="")
    parser_verify.add_argument("--verify-summary", default="")
    parser_verify.add_argument("--verify-evidence-path", default="")
    parser_verify.set_defaults(func=_cmd_mark_verify)

    parser_review = subparsers.add_parser("mark-review")
    parser_review.add_argument("--main-root", required=True)
    parser_review.add_argument("--integration-id", required=True)
    parser_review.add_argument("--review-status", required=True)
    parser_review.add_argument("--review-summary", default="")
    parser_review.add_argument("--review-evidence-path", default="")
    parser_review.set_defaults(func=_cmd_mark_review)

    parser_simplify = subparsers.add_parser("mark-simplify")
    parser_simplify.add_argument("--main-root", required=True)
    parser_simplify.add_argument("--integration-id", required=True)
    parser_simplify.add_argument("--simplify-status", required=True)
    parser_simplify.add_argument("--simplify-summary", default="")
    parser_simplify.set_defaults(func=_cmd_mark_simplify)

    parser_gate = subparsers.add_parser("mark-gate")
    parser_gate.add_argument("--main-root", required=True)
    parser_gate.add_argument("--integration-id", required=True)
    parser_gate.add_argument("--gate-status", required=True)
    parser_gate.add_argument("--gate-summary", default="")
    parser_gate.add_argument("--gate-evidence-path", default="")
    parser_gate.set_defaults(func=_cmd_mark_gate)

    parser_sync = subparsers.add_parser("mark-sync")
    parser_sync.add_argument("--main-root", required=True)
    parser_sync.add_argument("--integration-id", required=True)
    parser_sync.add_argument("--sync-status", required=True)
    parser_sync.add_argument("--sync-summary", default="")
    parser_sync.add_argument("--synced-main-sha", default="")
    parser_sync.set_defaults(func=_cmd_mark_sync)

    parser_cleanup = subparsers.add_parser("mark-cleanup")
    parser_cleanup.add_argument("--main-root", required=True)
    parser_cleanup.add_argument("--integration-id", required=True)
    parser_cleanup.add_argument("--cleanup-status", required=True)
    parser_cleanup.add_argument("--cleanup-summary", default="")
    parser_cleanup.set_defaults(func=_cmd_mark_cleanup)

    parser_replay = subparsers.add_parser("mark-replay")
    parser_replay.add_argument("--main-root", required=True)
    parser_replay.add_argument("--integration-id", required=True)
    parser_replay.add_argument("--replay-status", required=True)
    parser_replay.add_argument("--replay-summary", default="")
    parser_replay.add_argument("--planned-feature", dest="planned_features", action="append", default=[])
    parser_replay.set_defaults(func=_cmd_mark_replay)

    parser_claim = subparsers.add_parser("claim")
    parser_claim.add_argument("--main-root", required=True)
    parser_claim.add_argument("--integration-id", required=True)
    parser_claim.add_argument("--owner-token", required=True)
    parser_claim.set_defaults(func=_cmd_claim)

    parser_stale = subparsers.add_parser("is-stale")
    parser_stale.add_argument("--main-root", required=True)
    parser_stale.add_argument("--integration-id", required=True)
    parser_stale.add_argument("--current-main-sha", required=True)
    parser_stale.set_defaults(func=_cmd_is_stale)

    return parser


if __name__ == "__main__":
    parser = build_parser()
    args = parser.parse_args()
    try:
        raise SystemExit(args.func(args))
    except Exception as exc:
        _print_json({"status": "blocked", "reason": str(exc)})
        raise SystemExit(1)
