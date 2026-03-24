import json
import os
import subprocess
import sys


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
    cwd = os.getcwd()
    return cwd if os.path.exists(os.path.join(cwd, ".git")) else (run("git", "rev-parse", "--show-toplevel") or "")


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


def is_git_merge(cmd):
    cmd = cmd.strip()
    return cmd.startswith("git merge") or " git merge " in cmd


def is_verify_script(cmd):
    return "deploy/scripts/verify.sh" in cmd


def is_preview_build_script(cmd):
    return "deploy/scripts/preview-build.sh" in cmd


def is_release_build_script(cmd):
    return "deploy/scripts/release-build.sh" in cmd


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

    root = current_checkout_root()
    if not root:
        emit_allow()

    linked = is_linked_worktree(root)
    summary = state_summary(root)

    if not linked and is_edit_like(tool_name):
        emit_deny(
            summary
            + " 当前是主目录，不允许直接修改仓库文件。主目录只做控制面工作；如需开发或集成，请先进入 worktree。"
        )

    if not linked and is_bash(tool_name):
        if (
            is_verify_script(command)
            or is_preview_build_script(command)
            or is_release_build_script(command)
        ):
            emit_deny(
                summary
                + " 当前是主目录，不应直接运行 verify/preview-build/release-build。默认请在 linked worktree 中执行这些流程。"
            )

    if is_bash(tool_name) and is_git_merge(command):
        emit_deny(
            summary
            + " 当前仓库默认禁止 merge 进入本地集成流。请改用 cherry-pick / replay / rebase + squash 维护线性历史。"
        )

    emit_allow()


if __name__ == "__main__":
    main()
