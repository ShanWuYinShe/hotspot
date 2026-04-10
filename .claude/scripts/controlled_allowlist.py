from __future__ import annotations

# 这里登记的是“仓库提供给 Claude 主动发现和调用的严格入口”。
# 它们主要用于隔离运行、集成控制面和少量高确定性步骤，
# 不是用来声明“只有这些脚本才能做事”。

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
)

# hook 自身的内部守卫脚本；它们不是给 Claude 当成功能入口来发现的。
INTERNAL_GUARD_SCRIPTS = (
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
