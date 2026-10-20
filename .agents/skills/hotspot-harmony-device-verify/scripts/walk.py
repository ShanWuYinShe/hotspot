#!/usr/bin/env python3
"""鸿蒙端 UI 走查最小工具集（2026-10-20 实测可用）。

用法：作为库 import，或 `python3 walk.py <文本>` 快速点某个可见文本。

设计约束（都是踩过的坑）：
- **每步 dump 校验**：点完立刻 dump，页面不对就停，别继续盲点；
- **坐标从 dump 里取**，不要用记忆/截图估算的坐标（底栏实测 y=2599，写 2620 会落进系统手势区 → 切桌面）；
- **`keyEvent Back` 只在 IME/安全键盘打开时才安全**（只关键盘）；键盘未开时它等于导航后退，
  连按两次会退出应用；
- 登录密码框会弹**安全键盘**，会盖住输入框 → 输入完必须 Back 一次再点提交；
- 提交按钮常在折叠线以下，dump 只含可见节点 → 先 `fling` 滚动再按特征找
  （`clickable=true` + `w>600` + `h>120`）。
"""
import json
import re
import subprocess
import sys
import time

HDC = "/Applications/DevEco-Studio.app/Contents/sdk/default/openharmony/toolchains/hdc"
BOUNDS_RE = re.compile(r"\[(\d+),(\d+)\]\[(\d+),(\d+)\]")


def hdc(*args, timeout=40):
    return subprocess.run([HDC, *args], capture_output=True, text=True, timeout=timeout).stdout


def dump(retries=3):
    """取当前布局树（dict）；失败返回 {}。"""
    for _ in range(retries):
        hdc("shell", "uitest dumpLayout -p /data/local/tmp/walk.json")
        out = hdc("shell", "cat /data/local/tmp/walk.json")
        if out.strip().startswith("{"):
            try:
                return json.loads(out)
            except Exception:
                pass
        time.sleep(1)
    return {}


def nodes(tree):
    """扁平化布局树：每个节点给出 text/description/type/中心点/宽高/clickable。"""
    res = []

    def go(n):
        if isinstance(n, dict):
            a = n.get("attributes", {}) or {}
            m = BOUNDS_RE.match(a.get("bounds") or "")
            if m:
                x1, y1, x2, y2 = map(int, m.groups())
                res.append({
                    "text": (a.get("text") or "").strip(),
                    "desc": (a.get("description") or "").strip(),
                    "type": a.get("type") or "",
                    "cx": (x1 + x2) // 2,
                    "cy": (y1 + y2) // 2,
                    "w": x2 - x1,
                    "h": y2 - y1,
                    "clickable": str(a.get("clickable")),
                })
            for c in n.get("children", []) or []:
                go(c)
        elif isinstance(n, list):
            for c in n:
                go(c)

    go(tree)
    return res


def texts(ns):
    return [n["text"] for n in ns if n["text"]]


def find(ns, *labels, exact=True):
    """按 text/description 找节点（默认精确匹配，避免 includes 命中说明文字）。"""
    for n in ns:
        for lab in labels:
            if exact and (n["text"] == lab or n["desc"] == lab):
                return n
            if not exact and (lab in n["text"] or lab in n["desc"]):
                return n
    return None


def inputs(ns):
    """找输入框（dump 里 type 含 Input/TextArea）。"""
    return [n for n in ns if "Input" in n["type"] or "TextArea" in n["type"]]


def buttons(ns, min_w=600, min_h=100, min_cy=1200):
    """找大按钮（提交类）。按钮在折叠线下时先 fling 再调用。"""
    return [n for n in ns if n["clickable"] == "true" and n["w"] >= min_w and n["h"] >= min_h and n["cy"] >= min_cy]


def tap(x, y, wait=2.5):
    hdc("shell", f"uitest uiInput click {x} {y}")
    time.sleep(wait)


def tap_node(n, wait=2.5):
    tap(n["cx"], n["cy"], wait)


def input_text(x, y, value, wait=2.0):
    hdc("shell", f"uitest uiInput inputText {x} {y} '{value}'")
    time.sleep(wait)


def fling(x, y1, y2, speed=2500, wait=2.5):
    hdc("shell", f"uitest uiInput fling {x} {y1} {x} {y2} {speed}")
    time.sleep(wait)


def back(wait=2.5):
    hdc("shell", "uitest uiInput keyEvent Back")
    time.sleep(wait)


def snapshot(path):
    """保存当前屏幕到本地 jpg，返回字节数。"""
    hdc("shell", "snapshot_display -f /data/local/tmp/walk.jpeg")
    data = subprocess.run([HDC, "shell", "cat /data/local/tmp/walk.jpeg"], capture_output=True).stdout
    with open(path, "wb") as f:
        f.write(data)
    return len(data)


def main():
    if len(sys.argv) < 2:
        print("usage: walk.py <可见文本>   # 点击当前页面上该文本所在的可见元素")
        return 1
    ns = nodes(dump())
    target = find(ns, sys.argv[1])
    print("当前页文本:", texts(ns)[:10])
    if not target:
        print(f"未找到「{sys.argv[1]}」")
        return 1
    print(f"点击「{sys.argv[1]}」于 ({target['cx']},{target['cy']})")
    tap_node(target)
    print("点击后文本:", texts(nodes(dump()))[:10])
    return 0


if __name__ == "__main__":
    sys.exit(main())
