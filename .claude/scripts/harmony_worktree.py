#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import shutil
import sys
from pathlib import Path

from dev_common import (
    CommandResult,
    ensure_allowed_worktree,
    finish_blocked,
    now_stamp,
    run,
    summarize_failure,
    write_command_log,
)

_FAILURE_KEYWORDS = ("error", "failed", "exception", "arkts", "ets", "test")


def detect_harmony_root(repo_top: Path, explicit: str | None) -> Path:
    target = Path(explicit).expanduser().resolve() if explicit else (repo_top / "harmonyapp").resolve()
    if not target.is_dir():
        raise RuntimeError(f"Harmony 工程目录不存在: {target}")
    return target


def detect_hvigor_command(harmony_root: Path) -> list[str]:
    local_wrapper = harmony_root / "hvigorw"
    if local_wrapper.is_file():
        return [str(local_wrapper)]
    for name in ("hvigorw", "hvigor"):
        found = shutil.which(name)
        if found:
            return [found]
    known_candidates = [
        Path("/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw"),
        Path("/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigor"),
    ]
    for candidate in known_candidates:
        if candidate.is_file():
            return [str(candidate)]
    raise RuntimeError("未找到 hvigorw/hvigor 可执行文件。请先在本机安装或配置 HarmonyOS/DevEco Studio 命令行构建环境。")


def detect_ohpm_command() -> str:
    found = shutil.which("ohpm")
    if found:
        return found
    known_candidates = [
        Path("/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm"),
        Path("/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm-cli"),
    ]
    for candidate in known_candidates:
        if candidate.is_file():
            return str(candidate)
    raise RuntimeError("未找到 ohpm。请先安装或配置 Harmony 依赖管理环境。")


def ensure_dependencies_present(harmony_root: Path) -> None:
    oh_modules = harmony_root / "oh_modules"
    if oh_modules.exists():
        return
    raise RuntimeError(f"缺少依赖目录: {oh_modules}。请在当前 worktree 中先运行 harmony_worktree.py --task install 或 --task all。")


def install_dependencies(harmony_root: Path, logs_dir: Path) -> CommandResult:
    ohpm = detect_ohpm_command()
    result = run([ohpm, "install"], harmony_root)
    write_command_log(logs_dir / "install.log", result)
    return result


def candidate_build_commands(base: list[str]) -> list[list[str]]:
    return [
        [*base, "--mode", "module", "-p", "module=entry@default", "-p", "product=default", "-p", "buildMode=debug", "assembleHap"],
        [*base, "--mode", "project", "-p", "product=default", "-p", "buildMode=debug", "assembleApp"],
        [*base, "assembleHap"],
        [*base, "assembleApp"],
    ]


def run_build_like(harmony_root: Path, logs_dir: Path, stage_name: str) -> tuple[int, dict]:
    ensure_dependencies_present(harmony_root)
    hvigor_base = detect_hvigor_command(harmony_root)
    tried: list[dict[str, object]] = []
    for index, argv in enumerate(candidate_build_commands(hvigor_base), start=1):
        result = run(argv, harmony_root)
        log_path = logs_dir / f"{stage_name}-{index}.log"
        write_command_log(log_path, result)
        tried.append({"command": argv, "exit_code": result.returncode, "log": str(log_path)})
        if result.returncode == 0:
            summary = [f"Harmony 受控{stage_name}通过。"]
            if stage_name == "test":
                summary.append("当前仓库未配置独立 Harmony 测试任务，test 阶段复用 Hvigor 编译质量门。")
            return 0, {
                "status": "passed",
                "stage": stage_name,
                "harmony_root": str(harmony_root),
                "logs_dir": str(logs_dir),
                "command": argv,
                "tried": tried,
                "summary": summary,
            }
    last_log = Path(str(tried[-1]["log"]))
    log_text = last_log.read_text(encoding="utf-8") if last_log.exists() else ""
    last_result = CommandResult(list(tried[-1]["command"]), int(tried[-1]["exit_code"]), log_text, log_text)
    return 2, {
        "status": "failed",
        "stage": stage_name,
        "harmony_root": str(harmony_root),
        "logs_dir": str(logs_dir),
        "tried": tried,
        "summary": summarize_failure(last_result, keywords=_FAILURE_KEYWORDS),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="在当前 worktree 中受控运行 Harmony 安装 / 测试 / 编译闭环")
    parser.add_argument("--harmony-root", help="Harmony 工程目录，默认 <repo>/harmonyapp")
    parser.add_argument("--task", choices=["install", "build", "check", "test", "all"], default="check")
    parser.add_argument("--install", action="store_true", help="兼容旧入口：在 build/check/test/all 前先执行依赖安装")
    args = parser.parse_args()

    cwd = Path.cwd().resolve()
    logs_dir = cwd / "tmp" / "harmony-worktree" / now_stamp()
    logs_dir.mkdir(parents=True, exist_ok=True)

    try:
        repo = ensure_allowed_worktree(cwd)
        repo_top = Path(repo["top"]).resolve()
        harmony_root = detect_harmony_root(repo_top, args.harmony_root)

        report: dict[str, object] = {
            "status": "passed",
            "task": args.task,
            "worktree_root": str(repo_top),
            "harmony_root": str(harmony_root),
            "logs_dir": str(logs_dir),
            "steps": [],
        }

        need_install = args.install or args.task in {"install", "all"}
        if need_install:
            install_result = install_dependencies(harmony_root, logs_dir)
            install_step = {
                "stage": "install",
                "command": install_result.argv,
                "log": str(logs_dir / "install.log"),
                "exit_code": install_result.returncode,
            }
            report["steps"].append(install_step)
            if install_result.returncode != 0:
                report["status"] = "blocked"
                report["reason"] = "ohpm install 失败"
                report["summary"] = summarize_failure(install_result, keywords=_FAILURE_KEYWORDS)
                json.dump(report, sys.stdout, ensure_ascii=False, indent=2)
                sys.stdout.write("\n")
                return 1
            if args.task == "install":
                report["summary"] = ["Harmony 依赖安装通过。"]
                json.dump(report, sys.stdout, ensure_ascii=False, indent=2)
                sys.stdout.write("\n")
                return 0

        stage_task = "build" if args.task in {"build", "check", "all"} else "test"
        code, payload = run_build_like(harmony_root, logs_dir, stage_task)
        report["steps"].append(payload)
        report["status"] = payload["status"]
        report["summary"] = payload["summary"]
        json.dump(report, sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
        return code
    except RuntimeError as exc:
        return finish_blocked(args.task, str(exc), logs_dir)


if __name__ == "__main__":
    raise SystemExit(main())
