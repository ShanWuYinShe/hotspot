#!/usr/bin/env python3
import json
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

from git_context import GitContextError, get_repo_context, run_git

_VALID_SUBPROJECTS = frozenset({
    "backend",
    "backend-go",
    "frontend-svelte",
    "harmonyapp",
    "spec",
    "deploy",
    "claude-config",
})

_WORKTREE_NAME_RE = re.compile(
    r"^(?P<subproject>[a-z][a-z0-9-]+)\.(?P<module>[a-z][a-z0-9-]+)\.(?P<feature>[a-z][a-z0-9-]+)$"
)
_BRANCH_PREFIX = "worktree-"
_RESUME_SUFFIX_RE = re.compile(r"^(?P<root>.+)-r(?P<generation>[2-9][0-9]*)$")


@dataclass(frozen=True)
class WorktreeLineage:
    name: str
    generation: int
    branch_name: str


def validate_worktree_name(name: str) -> str | None:
    """Return an error message if the name is invalid, or None if valid."""
    m = _WORKTREE_NAME_RE.match(name)
    if not m:
        return (
            f"worktree 名称 '{name}' 不符合命名规则。"
            f"必须使用 '{{子项目名}}.{{模块名}}.{{功能描述}}' 格式，"
            f"各段仅允许小写字母、数字和连字符，以字母开头。"
            f"\n示例: backend-go.auth.add-oauth, harmonyapp.profile.fix-padding"
        )
    subproject = m.group("subproject")
    if subproject not in _VALID_SUBPROJECTS:
        valid = ", ".join(sorted(_VALID_SUBPROJECTS))
        return f"未知子项目 '{subproject}'，当前允许: {valid}"
    return None


def fail(message: str) -> int:
    sys.stderr.write(message + "\n")
    return 1


def parse_lineage_name(name: str) -> tuple[str, int]:
    m = _RESUME_SUFFIX_RE.match(name)
    if not m:
        return name, 1
    return m.group("root"), int(m.group("generation"))


def format_lineage_name(root_name: str, generation: int) -> str:
    if generation <= 1:
        return root_name
    return f"{root_name}-r{generation}"


def list_lineage_branches(primary_root: Path, root_name: str) -> list[WorktreeLineage]:
    try:
        refs = run_git(primary_root, "for-each-ref", "--format=%(refname:short)", "refs/heads")
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        raise RuntimeError(f"无法枚举本地分支: {stderr}") from exc

    lineages: list[WorktreeLineage] = []
    for branch_name in refs.splitlines():
        branch_name = branch_name.strip()
        if not branch_name.startswith(_BRANCH_PREFIX):
            continue
        lineage_name = branch_name[len(_BRANCH_PREFIX):]
        lineage_root, generation = parse_lineage_name(lineage_name)
        if lineage_root != root_name:
            continue
        lineages.append(
            WorktreeLineage(
                name=lineage_name,
                generation=generation,
                branch_name=branch_name,
            )
        )
    return sorted(lineages, key=lambda item: item.generation)


def create_worktree(primary_root: Path, worktree_dir: Path, branch_name: str, base_ref: str) -> None:
    subprocess.run(
        ["git", "-C", str(primary_root), "worktree", "add", "-b", branch_name, str(worktree_dir), base_ref],
        check=True,
        capture_output=True,
        text=True,
    )


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception as exc:
        return fail(f"WorktreeCreate hook 无法解析输入: {exc}")

    cwd_raw = payload.get("cwd")
    if not cwd_raw:
        return fail("WorktreeCreate hook 缺少 cwd，无法确认主目录。")

    cwd = Path(str(cwd_raw)).resolve()

    try:
        repo = get_repo_context(cwd)
        status = run_git(cwd, "status", "--porcelain")
        run_git(cwd, "show-ref", "--verify", "refs/heads/main")
    except GitContextError as exc:
        return fail(f"WorktreeCreate hook 无法读取 git 上下文: {exc}")
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        return fail(f"WorktreeCreate hook 无法读取 git 状态: {stderr}")

    primary_root = Path(repo["primary_root"])

    if not repo["is_primary_checkout"]:
        return fail("EnterWorktree 只能从主目录执行，当前不在主目录 checkout。")

    if repo["branch"] != "main":
        return fail("EnterWorktree 只能从主目录本地 main 分支执行。")

    if status:
        return fail("EnterWorktree 前主目录必须保持本地 main 且无未提交改动。")

    name = str(payload.get("name") or "").strip()
    if not name:
        return fail(
            "必须为 worktree 指定名称，格式为 '{子项目名}.{模块名}.{功能描述}'。"
            "\n示例: backend-go.auth.add-oauth, harmonyapp.profile.fix-padding"
        )

    name_error = validate_worktree_name(name)
    if name_error:
        return fail(name_error)

    try:
        existing_lineages = list_lineage_branches(primary_root, name)
    except RuntimeError as exc:
        return fail(str(exc))

    if existing_lineages:
        latest = existing_lineages[-1]
        effective_generation = latest.generation + 1
        effective_name = format_lineage_name(name, effective_generation)
        base_ref = latest.branch_name
        sys.stderr.write(
            f"检测到遗留 feature lineage '{name}'，将创建派生 worktree '{effective_name}' 并基于 {base_ref} 继续进度。\n"
        )
    else:
        effective_name = name
        base_ref = "main"

    worktree_dir = primary_root / ".claude" / "worktrees" / effective_name
    branch_name = f"{_BRANCH_PREFIX}{effective_name}"

    try:
        create_worktree(primary_root, worktree_dir, branch_name, base_ref)
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        return fail(f"创建 worktree 失败（branch={branch_name}, base={base_ref}）: {stderr}")

    sys.stdout.write(str(worktree_dir.resolve()) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
