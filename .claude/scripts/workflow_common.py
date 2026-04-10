from __future__ import annotations

import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

from dev_common import finish_blocked, finish_report
from git_context import GitContextError, get_repo_context, run_git
from integration_meta import init_meta


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
    meta_path = init_meta(str(cwd), integration_id, baseline_main_sha, integration_branch, str(integration_worktree.resolve()), list(features))
    replay_plan = [
        {
            "source_branch": feature,
            "command": ["git", "-C", str(integration_worktree), "cherry-pick", feature],
        }
        for feature in features
    ]
    return {
        "integration_id": integration_id,
        "integration_branch": integration_branch,
        "integration_worktree": str(integration_worktree.resolve()),
        "baseline_main_sha": baseline_main_sha,
        "planned_features": list(features),
        "metadata_path": meta_path,
        "replay_plan": replay_plan,
    }
