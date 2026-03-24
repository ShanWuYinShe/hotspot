import json
import os
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

from repo_context import checkout_root_from_cwd, main_root_from_checkout, resolve_script_path


def emit_allow(message=""):
    payload = {"hookSpecificOutput": {"permissionDecision": "allow"}}
    if message:
        payload["hookSpecificOutput"]["permissionDecisionReason"] = message
        payload["systemMessage"] = message
    print(json.dumps(payload, ensure_ascii=False))
    sys.exit(0)


def main():
    checkout_root = checkout_root_from_cwd()
    main_root = main_root_from_checkout(checkout_root)
    if not checkout_root or not main_root:
        emit_allow("PreToolUse bootstrap 未能解析 git 根；已按 fail-open 放行。")

    script = resolve_script_path(checkout_root, main_root, ".claude/scripts/pre_tool_use.py")
    if not script:
        emit_allow("PreToolUse bootstrap 未找到脚本；已按 fail-open 放行。")

    os.execvp("python3", ["python3", script, "--main-root", main_root, "--checkout-root", checkout_root])


if __name__ == "__main__":
    main()
