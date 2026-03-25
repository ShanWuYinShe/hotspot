import json
import os
from datetime import datetime, timezone


VERIFY_STATUSES = {"pending", "passed", "failed", "stale"}
REVIEW_STATUSES = {"pending", "passed", "failed", "stale"}
GATE_STATUSES = {"pending", "ready_to_sync", "blocked", "must_refresh"}

GATE_STATUS_ALIASES = {
    "ready to sync": "ready_to_sync",
    "ready_to_sync": "ready_to_sync",
    "blocked": "blocked",
    "must refresh": "must_refresh",
    "must_refresh": "must_refresh",
}

META_REQUIRED_FIELDS = (
    "integration_id",
    "baseline_main_sha",
    "integration_branch",
    "integration_worktree_path",
    "planned_features",
    "verify_status",
    "review_status",
    "gate_status",
    "created_at",
    "updated_at",
    "verify_env_name",
    "verify_summary",
    "review_summary",
    "gate_summary",
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


def normalize_meta(data):
    normalized = dict(data or {})

    if "planned_features" in normalized:
        normalized["planned_features"] = normalize_list(normalized["planned_features"])

    if "verify_status" in normalized:
        normalized["verify_status"] = normalize_simple_status(
            normalized["verify_status"], VERIFY_STATUSES, "verify_status"
        )

    if "review_status" in normalized:
        normalized["review_status"] = normalize_simple_status(
            normalized["review_status"], REVIEW_STATUSES, "review_status"
        )

    if "gate_status" in normalized:
        normalized["gate_status"] = normalize_gate_status(normalized["gate_status"])

    return normalized


def validate_meta(data):
    missing = [field for field in META_REQUIRED_FIELDS if field not in data]
    if missing:
        raise ValueError(f"integration metadata missing required fields: {', '.join(missing)}")
    normalize_meta(data)
    return data


def default_meta(integration_id, baseline_main_sha, integration_branch, integration_worktree_path, planned_features):
    stamp = now_iso()
    return {
        "integration_id": integration_id,
        "baseline_main_sha": baseline_main_sha,
        "integration_branch": integration_branch,
        "integration_worktree_path": integration_worktree_path,
        "planned_features": normalize_list(planned_features),
        "verify_status": "pending",
        "review_status": "pending",
        "gate_status": "pending",
        "created_at": stamp,
        "updated_at": stamp,
        "verify_env_name": "",
        "verify_summary": "",
        "review_summary": "",
        "gate_summary": "",
        "source_integration_id": "",
    }


def load_meta(main_root, integration_id):
    path = meta_path(main_root, integration_id)
    if not os.path.exists(path):
        return None
    try:
        with open(path, "r", encoding="utf-8") as f:
            return normalize_meta(json.load(f))
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
    normalized = normalize_meta(data)
    validate_meta(normalized)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(normalized, f, ensure_ascii=False, indent=2)
        f.write("\n")
    return path


def init_meta(main_root, integration_id, baseline_main_sha, integration_branch, integration_worktree_path, planned_features):
    data = default_meta(
        integration_id,
        baseline_main_sha,
        integration_branch,
        integration_worktree_path,
        planned_features,
    )
    return write_meta(main_root, integration_id, data)


def update_meta(main_root, integration_id, **patch):
    data = load_meta(main_root, integration_id)
    if data is None:
        raise ValueError(f"integration metadata does not exist: {integration_id}")
    data.update(normalize_meta(patch))
    data["updated_at"] = now_iso()
    return write_meta(main_root, integration_id, data)


def mark_verify_status(main_root, integration_id, verify_status, verify_env_name="", verify_summary=""):
    return update_meta(
        main_root,
        integration_id,
        verify_status=verify_status,
        verify_env_name=verify_env_name,
        verify_summary=verify_summary,
    )


def mark_review_status(main_root, integration_id, review_status, review_summary=""):
    return update_meta(
        main_root,
        integration_id,
        review_status=review_status,
        review_summary=review_summary,
    )


def mark_gate_status(main_root, integration_id, gate_status, gate_summary=""):
    return update_meta(
        main_root,
        integration_id,
        gate_status=gate_status,
        gate_summary=gate_summary,
    )


def stale(main_root, data, current_main_sha):
    baseline = (data or {}).get("baseline_main_sha", "")
    return bool(baseline and current_main_sha and baseline != current_main_sha)
