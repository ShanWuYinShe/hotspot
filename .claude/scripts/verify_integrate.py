#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from integration_meta import load_meta, mark_verify_status, stale
from workflow_common import classify_ownership, head_sha, ok, require_integration, resolve_repo, shell_run


def main() -> int:
    parser = argparse.ArgumentParser(description="执行 integration 标准验证并回写 verify 状态")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--integration-id")
    parser.add_argument("--env-name")
    parser.add_argument("--owner-token", default="")
    parser.add_argument("--verify-evidence-path", default="")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    repo = resolve_repo(cwd)
    require_integration(repo)
    branch = str(repo.get("branch") or "")
    integration_id = args.integration_id or branch.removeprefix("integration-")
    env_name = args.env_name or branch
    primary_root = Path(str(repo["primary_root"])).resolve()
    meta = load_meta(str(primary_root), integration_id)
    if meta is None:
        raise SystemExit(ok(status="blocked", reason=f"integration metadata 不存在: {integration_id}"))
    if stale(str(primary_root), meta, head_sha(primary_root, "main")):
        raise SystemExit(ok(status="blocked", reason="integration 基线已过期，请先 refresh。", next_step="/refresh_integrate"))
    if classify_ownership(meta, args.owner_token.strip() or None) != "owned":
        raise SystemExit(ok(status="blocked", reason="当前 integration 不属于本次自动流程拥有的对象。"))
    if meta.get("replay_status") != "applied":
        raise SystemExit(ok(status="blocked", reason="planned features 尚未全部 replay 完成。"))

    verify_cmd = ["./deploy/scripts/verify.sh", "--env-name", env_name]
    try:
        shell_run(primary_root, *verify_cmd)
        verify_status = "passed"
        verify_summary = f"integration {integration_id} verify 通过，env-name={env_name}"
    except RuntimeError as exc:
        verify_status = "failed"
        verify_summary = str(exc)

    meta_path = mark_verify_status(str(primary_root), integration_id, verify_status, env_name, verify_summary, args.verify_evidence_path)
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
