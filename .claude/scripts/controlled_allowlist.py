from __future__ import annotations

CONTROLLED_PYTHON_SCRIPTS = (
    "status_worktrees",
    "integrate_prepare",
    "verify_integrate",
    "finish_integrate",
    "sync_back_main",
    "cleanup_worktrees",
    "rust_dev",
    "go_dev",
    "frontend_dev",
    "harmony_worktree",
    "integration_meta",
    "inspect_verify",
    "worktree_guard",
    "tool_guard",
)

CONTROLLED_DEPLOY_SCRIPTS = (
    "verify.sh",
    "verify-inspect.sh",
    "verify-inspect-down.sh",
    "preview-build.sh",
    "preview-up.sh",
    "preview-inspect.sh",
    "preview-down.sh",
    "release-build.sh",
)
