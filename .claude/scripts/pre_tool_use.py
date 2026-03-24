import json
import os
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

from integration_meta import load_meta, stale as integration_meta_stale
from repo_context import checkout_root_from_cwd, main_root_from_checkout


def run(*args):
    try:
        return subprocess.run(args, capture_output=True, text=True).stdout.strip()
    except Exception:
        return ""


def emit_allow(message=""):
    payload = {"hookSpecificOutput": {"permissionDecision": "allow"}}
    if message:
        payload["hookSpecificOutput"]["permissionDecisionReason"] = message
        payload["systemMessage"] = message
    print(json.dumps(payload, ensure_ascii=False))
    sys.exit(0)


def emit_warn(message):
    payload = {"hookSpecificOutput": {"permissionDecision": "allow", "permissionDecisionReason": message}, "systemMessage": message}
    print(json.dumps(payload, ensure_ascii=False))
    sys.exit(0)


def emit_deny(message):
    print(
        json.dumps(
            {
                "hookSpecificOutput": {
                    "permissionDecision": "deny",
                    "permissionDecisionReason": message,
                },
                "systemMessage": message,
            },
            ensure_ascii=False,
        )
    )
    sys.exit(0)


def current_checkout_root():
    return checkout_root_from_cwd()


def repo_main_root(root):
    return main_root_from_checkout(root)


def is_linked_worktree(root):
    return bool(root) and os.path.isfile(os.path.join(root, ".git"))


def current_branch(root):
    return run("git", "-C", root, "branch", "--show-current") or "(detached)"


def is_dirty(root):
    return bool(run("git", "-C", root, "status", "--short"))


def state_summary(root):
    role = "worktree" if is_linked_worktree(root) else "主目录"
    state = "dirty" if is_dirty(root) else "clean"
    return f"位置: {role}；分支: {current_branch(root)}；工作区: {state}。"


def is_edit_like(tool_name):
    return tool_name in ("Edit", "Write", "NotebookEdit")


def is_bash(tool_name):
    return tool_name == "Bash"


def cmd_starts_with(cmd, prefix):
    cmd = cmd.strip()
    return cmd.startswith(prefix) or f" {prefix}" in cmd


def is_git_merge(cmd):
    return cmd_starts_with(cmd, "git merge")


def is_git_rebase(cmd):
    return cmd_starts_with(cmd, "git rebase")


def is_git_cherry_pick(cmd):
    return cmd_starts_with(cmd, "git cherry-pick")


def is_git_add(cmd):
    return cmd_starts_with(cmd, "git add")


def is_git_commit(cmd):
    return cmd_starts_with(cmd, "git commit")


def is_verify_script(cmd):
    return "deploy/scripts/verify.sh" in cmd


def is_preview_build_script(cmd):
    return "deploy/scripts/preview-build.sh" in cmd


def is_release_build_script(cmd):
    return "deploy/scripts/release-build.sh" in cmd


def integration_id(root):
    branch = current_branch(root)
    return branch if branch and branch != "(detached)" else os.path.basename(root)


def load_integration_meta(main_root, root):
    return load_meta(main_root, integration_id(root))


def is_integration_like(main_root, root):
    branch = current_branch(root).lower()
    label = os.path.basename(root).lower()
    if "integration" in branch or "integration" in label:
        return True
    return load_integration_meta(main_root, root) is not None


def current_main_head(main_root):
    return run("git", "-C", main_root, "rev-parse", "HEAD")


def integration_is_stale(main_root, root):
    meta = load_integration_meta(main_root, root)
    if not meta:
        return False
    return integration_meta_stale(main_root, meta, current_main_head(main_root))


def arg_value(flag):
    if flag in sys.argv:
        idx = sys.argv.index(flag)
        if idx + 1 < len(sys.argv):
            return sys.argv[idx + 1]
    return ""


def main():
    raw = sys.stdin.read().strip()
    payload = {}
    if raw:
        try:
            payload = json.loads(raw)
        except Exception:
            payload = {}

    tool_name = payload.get("tool_name") or payload.get("tool") or ""
    tool_input = payload.get("tool_input") or payload.get("input") or {}
    command = tool_input.get("command", "") if tool_name == "Bash" else ""

    root = arg_value("--checkout-root") or current_checkout_root()
    if not root:
        emit_allow("PreToolUse 未能解析当前 checkout；已按 fail-open 放行。")

    main_root = arg_value("--main-root") or repo_main_root(root)
    linked = is_linked_worktree(root)
    summary = state_summary(root)
    integration_like = linked and is_integration_like(main_root, root)
    stale = integration_like and integration_is_stale(main_root, root)

    if not linked and is_edit_like(tool_name):
        emit_deny(
            summary
            + " 当前是主目录，不允许直接修改仓库文件。主目录只做控制面工作；如需开发或集成，请先进入 worktree。"
        )

    if not linked and is_bash(tool_name):
        if is_verify_script(command) or is_preview_build_script(command) or is_release_build_script(command):
            emit_deny(
                summary
                + " 当前是主目录，不应直接运行 verify/preview-build/release-build。默认请在 linked worktree 中执行这些流程。"
            )

    if is_bash(tool_name) and is_git_merge(command):
        emit_deny(
            summary
            + " 当前仓库默认禁止 merge 进入本地集成流。请改用 cherry-pick / replay / rebase + squash 维护线性历史。"
        )

    if linked and is_bash(tool_name) and (is_git_rebase(command) or is_git_cherry_pick(command)):
        emit_warn(
            summary
            + " 当前操作将改动历史或 replay 提交。请确认你是在 feature/integration worktree 中执行线性整理，而不是在主目录修改历史。"
        )

    if integration_like and is_edit_like(tool_name):
        emit_warn(
            summary
            + " 当前看起来是 integration worktree。这里应主要用于汇总、冲突处理、verify、review 和提交收敛；若你在继续开发新功能，请考虑回到 feature worktree。"
        )

    if integration_like and is_bash(tool_name) and (is_git_add(command) or is_git_commit(command)):
        emit_warn(
            summary
            + " 当前看起来是 integration worktree。请确认这次 add/commit 是集成修正或线性化收敛，而不是继续承载新的功能开发。"
        )

    if integration_like and stale and is_bash(tool_name):
        if (
            is_verify_script(command)
            or is_preview_build_script(command)
            or is_release_build_script(command)
            or is_git_commit(command)
            or is_git_cherry_pick(command)
            or is_git_rebase(command)
        ):
            emit_warn(
                summary
                + " 当前 integration 基线相对主目录 main 已过期；继续前请优先确认是否应改走 /refresh_integrate。"
            )

    emit_allow()


if __name__ == "__main__":
    main()
