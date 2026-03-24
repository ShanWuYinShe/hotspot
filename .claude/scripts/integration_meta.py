import json
import os
from datetime import datetime, timezone


def now_iso():
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def meta_dir(main_root):
    return os.path.join(main_root, "tmp", "claude", "integrations")


def meta_path(main_root, integration_id):
    return os.path.join(meta_dir(main_root), f"{integration_id}.json")


def ensure_meta_dir(main_root):
    os.makedirs(meta_dir(main_root), exist_ok=True)
    return meta_dir(main_root)


def default_meta(integration_id, baseline_main_sha, integration_branch, integration_worktree_path, planned_features):
    stamp = now_iso()
    return {
        "integration_id": integration_id,
        "baseline_main_sha": baseline_main_sha,
        "integration_branch": integration_branch,
        "integration_worktree_path": integration_worktree_path,
        "planned_features": planned_features,
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
            return json.load(f)
    except Exception:
        return None


def write_meta(main_root, integration_id, data):
    ensure_meta_dir(main_root)
    path = meta_path(main_root, integration_id)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
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
    data = load_meta(main_root, integration_id) or {}
    data.update(patch)
    data["updated_at"] = now_iso()
    return write_meta(main_root, integration_id, data)


def stale(main_root, data, current_main_sha):
    baseline = (data or {}).get("baseline_main_sha", "")
    return bool(baseline and current_main_sha and baseline != current_main_sha)
