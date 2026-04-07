#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import mark_verify_status
from workflow_common import ok, require_integration, resolve_repo


def main() -> int:
    parser = argparse.ArgumentParser(description="登记 integration verify 计划并回写 verify 状态（不直接执行标准验证）")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--integration-id")
    parser.add_argument("--env-name")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_integration(repo)
    branch = str(repo.get("branch") or "")
    integration_id = args.integration_id or branch.removeprefix("integration-")
    env_name = args.env_name or branch
    verify_summary = f"integration {integration_id} 已登记 verify 入口，env-name={env_name}"
    meta_path = mark_verify_status(str(Path(str(repo['primary_root'])).resolve()), integration_id, "pending", env_name, verify_summary)
    raise SystemExit(
        ok(
            action="verify-integrate-plan",
            integration_branch=branch,
            integration_id=integration_id,
            env_name=env_name,
            metadata_path=meta_path,
            preflight_plan=[
                "Rust / Go / Frontend / Harmony 局部自检",
            ],
            verify_entry="需要在受控入口或 Bash 模式下实际执行 deploy verify",
        )
    )


if __name__ == "__main__":
    main()
