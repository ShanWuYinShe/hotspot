#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import mark_verify_status
from workflow_common import ok, require_integration, resolve_repo, shell_run


def main() -> int:
    parser = argparse.ArgumentParser(description="执行 integration 标准验证并回写 verify 状态")
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
    primary_root = Path(str(repo["primary_root"])).resolve()
    verify_cmd = ["./deploy/scripts/verify.sh", "--env-name", env_name]

    try:
        shell_run(primary_root, *verify_cmd)
        verify_status = "passed"
        verify_summary = f"integration {integration_id} verify 通过，env-name={env_name}"
    except RuntimeError as exc:
        verify_status = "failed"
        verify_summary = str(exc)

    meta_path = mark_verify_status(str(primary_root), integration_id, verify_status, env_name, verify_summary)
    raise SystemExit(
        ok(
            action="verify-integrate",
            integration_branch=branch,
            integration_id=integration_id,
            env_name=env_name,
            metadata_path=meta_path,
            verify_status=verify_status,
            verify_summary=verify_summary,
        )
    )


if __name__ == "__main__":
    main()
