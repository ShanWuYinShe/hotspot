import os
import subprocess


def run(*args):
    try:
        return subprocess.run(args, capture_output=True, text=True).stdout.strip()
    except Exception:
        return ""


def checkout_root_from_cwd():
    try:
        cwd = os.getcwd()
    except OSError:
        return ""
    if os.path.exists(os.path.join(cwd, ".git")):
        return cwd
    return run("git", "rev-parse", "--show-toplevel") or ""


def main_root_from_checkout(checkout_root):
    if not checkout_root:
        return ""
    common_dir = run("git", "-C", checkout_root, "rev-parse", "--git-common-dir")
    if not common_dir:
        return checkout_root
    common_abs = os.path.abspath(os.path.join(checkout_root, common_dir))
    return os.path.abspath(os.path.join(common_abs, os.pardir))


def resolve_script_path(checkout_root, main_root, relative_path):
    candidates = []
    if checkout_root:
        candidates.append(os.path.join(checkout_root, relative_path))
    if main_root:
        candidates.append(os.path.join(main_root, relative_path))
    for path in candidates:
        if path and os.path.isfile(path):
            return path
    return ""
