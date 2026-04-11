from __future__ import annotations

import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from uuid import uuid4

from feature_meta import feature_id_from_branch, init_meta as init_feature_meta, list_meta as list_feature_meta, load_meta as load_feature_meta
from git_context import GitContextError, get_repo_context, run_git
from integration_meta import init_meta, list_meta as list_integration_meta, load_meta as load_integration_meta


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def ok(**payload: object) -> int:
    data = {"status": "ok"}
    data.update(payload)
    json.dump(data, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


def fail(message: str, *, code: int = 1, **extra: object) -> int:
    payload = {"status": "blocked", "reason": message}
    payload.update(extra)
    json.dump(payload, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return code


def resolve_repo(cwd: Path) -> dict[str, object]:
    try:
        return get_repo_context(cwd)
    except GitContextError as exc:
        raise RuntimeError(f"无法读取 git 上下文: {exc}") from exc


def require_main_primary(repo: dict[str, object]) -> None:
    if not repo["is_main_primary"]:
        raise RuntimeError("该操作只能在主目录本地 main 上执行。")


def require_integration(repo: dict[str, object]) -> None:
    branch = str(repo.get("branch") or "")
    if not branch.startswith("integration-"):
        raise RuntimeError("该操作只能在 integration worktree 中执行。")


def git_output(cwd: Path, *args: str) -> str:
    try:
        return run_git(cwd, *args)
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        raise RuntimeError(stderr or f"git {' '.join(args)} 失败") from exc


def git_run(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    completed = subprocess.run(["git", "-C", str(cwd), *args], capture_output=True, text=True)
    if completed.returncode != 0:
        stderr = (completed.stderr or completed.stdout or "").strip()
        raise RuntimeError(stderr or f"git {' '.join(args)} 失败")
    return completed


def shell_run(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    completed = subprocess.run(list(args), cwd=str(cwd), capture_output=True, text=True)
    if completed.returncode != 0:
        stderr = (completed.stderr or completed.stdout or "").strip()
        raise RuntimeError(stderr or f"命令 {' '.join(args)} 失败")
    return completed


def clean_status(cwd: Path) -> bool:
    return git_output(cwd, "status", "--short") == ""


def head_sha(cwd: Path, ref: str = "HEAD") -> str:
    return git_output(cwd, "rev-parse", ref)


def list_worktrees(cwd: Path) -> list[dict[str, str]]:
    output = git_output(cwd, "worktree", "list", "--porcelain")
    items: list[dict[str, str]] = []
    current: dict[str, str] = {}
    for raw in output.splitlines():
        line = raw.strip()
        if not line:
            if current:
                items.append(current)
                current = {}
            continue
        key, _, value = line.partition(" ")
        current[key] = value
    if current:
        items.append(current)
    return items


def classify_worktree(branch: str) -> str:
    if branch == "main":
        return "main"
    if branch.startswith("integration-"):
        return "integration"
    if branch.startswith("worktree-"):
        return "feature"
    return "other"


def parse_feature_lineage(branch: str) -> tuple[str, int]:
    if not branch.startswith("worktree-"):
        return branch, 1
    name = branch[len("worktree-") :]
    if name.endswith("-r2") or "-r" in name:
        head, _, maybe = name.rpartition("-r")
        if head and maybe.isdigit() and int(maybe) >= 2:
            return head, int(maybe)
    return name, 1


def merge_base_sha(cwd: Path, ref1: str, ref2: str = "HEAD") -> str:
    return git_output(cwd, "merge-base", ref1, ref2)


def commit_count_between(cwd: Path, ref1: str, ref2: str = "HEAD") -> int:
    return int(git_output(cwd, "rev-list", "--count", f"{ref1}..{ref2}") or "0")


def merge_commit_count_between(cwd: Path, ref1: str, ref2: str = "HEAD") -> int:
    return int(git_output(cwd, "rev-list", "--count", "--merges", f"{ref1}..{ref2}") or "0")


def is_integration_branch(branch: str) -> bool:
    return branch.startswith("integration-")


def is_feature_branch(branch: str) -> bool:
    return branch.startswith("worktree-")


def feature_id_from_repo(repo: dict[str, object]) -> str:
    branch = str(repo.get("branch") or "")
    if not is_feature_branch(branch):
        raise RuntimeError("当前不在 feature worktree。")
    return feature_id_from_branch(branch)


def ensure_owner_token(owner_token: str | None = None) -> str:
    token = (owner_token or "").strip()
    return token or f"owner-{uuid4().hex[:12]}"


def classify_ownership(meta: dict[str, object] | None, owner_token: str | None) -> str:
    if not meta:
        return "takeover_required"
    meta_token = str(meta.get("owner_token") or "")
    if not meta_token:
        return "takeover_required"
    if owner_token and meta_token == owner_token:
        return "owned"
    return "foreign"


def feature_ready_to_sync(meta: dict[str, object] | None) -> bool:
    if not meta:
        return False
    return (
        meta.get("check_status") == "passed"
        and meta.get("test_status") in {"passed", "skipped"}
        and meta.get("simplify_status") in {"passed", "skipped"}
        and meta.get("review_status") == "passed"
        and meta.get("verify_status") in {"passed", "skipped"}
        and meta.get("commit_status") == "ready"
    )


def integration_ready_to_sync(meta: dict[str, object] | None) -> bool:
    if not meta:
        return False
    return meta.get("gate_status") == "ready_to_sync"


def metadata_drift(repo: dict[str, object], meta: dict[str, object] | None, path_field: str) -> bool:
    if not meta:
        return False
    expected = Path(str(meta.get(path_field) or "")).resolve()
    actual = Path(str(repo.get("top") or "")).resolve()
    return bool(expected and expected != actual)


def cleanup_eligibility(meta: dict[str, object] | None, *, tip_equals_main: bool, clean: bool, owner_token: str | None) -> str:
    if not meta:
        return "needs confirmation"
    if classify_ownership(meta, owner_token) != "owned":
        return "needs confirmation"
    if meta.get("sync_status") != "synced":
        return "keep"
    if not tip_equals_main or not clean:
        return "keep"
    return "safe to cleanup"


def collect_feature_runtime(primary_root: Path) -> dict[str, dict[str, object]]:
    items = list_feature_meta(str(primary_root))
    return {str(item.get("feature_branch") or ""): item for item in items}


def collect_integration_runtime(primary_root: Path) -> dict[str, dict[str, object]]:
    items = list_integration_meta(str(primary_root))
    return {str(item.get("integration_branch") or ""): item for item in items}


def init_feature_runtime(
    *,
    primary_root: Path,
    branch: str,
    worktree_path: str,
    baseline_main_sha: str,
    task_summary: str,
    owner_token: str,
) -> str:
    return init_feature_meta(
        str(primary_root),
        feature_branch=branch,
        feature_worktree_path=worktree_path,
        baseline_main_sha=baseline_main_sha,
        task_summary=task_summary,
        owner_token=owner_token,
    )


def load_feature_runtime(primary_root: Path, branch: str) -> dict[str, object] | None:
    return load_feature_meta(str(primary_root), branch=branch)


def load_integration_runtime(primary_root: Path, integration_id: str) -> dict[str, object] | None:
    return load_integration_meta(str(primary_root), integration_id)


def ensure_within_worktree(cwd: Path, raw_paths: list[str]) -> tuple[Path, list[Path]]:
    repo = resolve_repo(cwd)
    worktree_root = Path(str(repo["top"])).resolve()
    resolved_paths: list[Path] = []
    for raw in raw_paths:
        target = (worktree_root / raw).resolve(strict=False)
        try:
            target.relative_to(worktree_root)
        except ValueError:
            raise RuntimeError(f"路径超出当前 worktree: {raw}")
        resolved_paths.append(target)
    return worktree_root, resolved_paths


def prepare_integration_worktree(
    *,
    cwd: Path,
    integration_id: str,
    branch: str | None,
    worktree_path: str | None,
    features: list[str],
    create_if_missing: bool,
    owner_token: str | None = None,
    source_integration_id: str = "",
) -> dict[str, object]:
    repo = resolve_repo(cwd)
    require_main_primary(repo)
    if not clean_status(cwd):
        raise RuntimeError("主目录本地 main 必须保持干净后才能操作 integration。")
    if not features:
        raise RuntimeError("缺少至少一个 --feature。")
    integration_branch = branch or f"integration-{integration_id}"
    integration_worktree = Path(worktree_path) if worktree_path else cwd / ".claude" / "worktrees" / integration_branch
    baseline_main_sha = head_sha(cwd, "main")
    if create_if_missing and not integration_worktree.exists():
        git_run(cwd, "worktree", "add", "-b", integration_branch, str(integration_worktree), "main")
    elif not create_if_missing:
        git_run(cwd, "worktree", "add", "-b", integration_branch, str(integration_worktree), "main")

    normalized_features = []
    replay_plan = []
    for feature in features:
        source_branch = str(feature)
        source_head_sha = head_sha(cwd, source_branch)
        normalized_features.append(
            {
                "source_branch": source_branch,
                "source_head_sha": source_head_sha,
                "replayed_head_sha": "",
                "replay_status": "pending",
                "replay_summary": "",
            }
        )
        replay_plan.append(
            {
                "source_branch": source_branch,
                "source_head_sha": source_head_sha,
                "command": ["git", "-C", str(integration_worktree), "cherry-pick", source_branch],
            }
        )

    token = ensure_owner_token(owner_token)
    meta_path = init_meta(
        str(cwd),
        integration_id,
        baseline_main_sha,
        integration_branch,
        str(integration_worktree.resolve()),
        normalized_features,
        owner_token=token,
        source_integration_id=source_integration_id,
    )
    return {
        "integration_id": integration_id,
        "integration_branch": integration_branch,
        "integration_worktree": str(integration_worktree.resolve()),
        "baseline_main_sha": baseline_main_sha,
        "owner_token": token,
        "planned_features": normalized_features,
        "metadata_path": meta_path,
        "replay_plan": replay_plan,
    }
