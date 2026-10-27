#!/usr/bin/env python3
"""跨端「on-soft 文字色」一致性校验（R12）。

对齐口径：web --cst-text-{accent,status-online,status-warning,status-danger}-on-soft
= 语义色 80% + text-primary 20%，仅用于 tint 软底上的文字。

断言：鸿蒙 *OnSoft == web 同主题同明暗 80/20 混色（逐通道 ±1）；3 主题×明暗×4 类
on-soft vs 最差软底对比度 ≥4.5（含 danger /50）；语义色/palette/textPrimary 两端一致。
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
WEB_THEMES = ROOT / "frontend" / "src" / "styles" / "theme" / "themes"
HARMONY = ROOT / "harmonyapp" / "entry" / "src" / "main" / "ets" / "core" / "theme" / "ThemeConstants.ets"

CONFIG_MAP = {
    "VIBRANT_LIGHT": ("vibrant", "light"),
    "VIBRANT_DARK": ("vibrant", "dark"),
    "DEFAULT_CLEAN_LIGHT": ("forest", "light"),
    "DEFAULT_CLEAN_DARK": ("forest", "dark"),
    "INKWASH_LIGHT": ("inkwash", "light"),
    "INKWASH_DARK": ("inkwash", "dark"),
}
ALPHA = {
    "accent": {"light": 0.10, "dark": 0.12},
    "online": {"light": 0.13, "dark": 0.14},
    "warning": {"light": 0.13, "dark": 0.14},
    "danger": {"light": 0.13, "dark": 0.14},
}
HEX_KEYS = {
    "accent": "accent-primary",
    "online": "online",
    "warning": "warning",
    "danger": "hot",
}
def rgb(hex_str):
    h = hex_str.lstrip("#")
    if len(h) == 8:
        h = h[2:]
    return tuple(int(h[i : i + 2], 16) for i in (0, 2, 4))


def tohex(t):
    return "#%02X%02X%02X" % t


def mix(a, b, m=0.8):
    return tuple(round(x * m + y * (1 - m)) for x, y in zip(a, b))


def blend(top, alpha, base):
    return tuple(round(c * alpha + b * (1 - alpha)) for c, b in zip(top, base))


def lum(c):
    def f(x):
        x /= 255
        return x / 12.92 if x <= 0.03928 else ((x + 0.055) / 1.055) ** 2.4

    r, g, b = c
    return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)


def ratio(a, b):
    la, lb = lum(a), lum(b)
    hi, lo = max(la, lb), min(la, lb)
    return (hi + 0.05) / (lo + 0.05)


def parse_web(theme, mode):
    text = (WEB_THEMES / (theme + ".css")).read_text(encoding="utf-8")
    if mode == "dark":
        marker = "." + theme + ".dark {"
        idx = text.find(marker)
        if idx >= 0:
            text = text[idx:]
    out = {}
    for key in ("accent-primary", "online", "warning", "hot", "text"):
        m = re.search(r"--cst-palette-" + key + r":\s*(#[0-9A-Fa-f]{6})", text)
        if m:
            out[key] = m.group(1)
    m = re.search(r"--cst-palette-page-bg:\s*(#[0-9A-Fa-f]{6})", text)
    if m:
        out["page"] = m.group(1)
    return out


def parse_harmony():
    text = HARMONY.read_text(encoding="utf-8")
    blocks = re.findall(r"static readonly (\w+)_COLORS: ColorConfig = \{(.*?)\n  \};", text, re.S)
    result = {}
    for name, body in blocks:
        fields = {}
        for m in re.finditer(r"^\s*(\w+):\s*'(#?[0-9A-Fa-f]{6,8})'", body, re.M):
            fields[m.group(1)] = m.group(2)
        result[name] = fields
    return result
def main():
    harmony = parse_harmony()
    failures = []
    rows = []
    worst = {}
    for cfg_name, (theme, mode) in CONFIG_MAP.items():
        cfg = harmony.get(cfg_name)
        if not cfg:
            failures.append("鸿蒙缺少 " + cfg_name)
            continue
        web = parse_web(theme, mode)
        for key, w, h in (
            ("online", web.get("online"), cfg.get("online")),
            ("warning", web.get("warning"), cfg.get("warning")),
            ("error", web.get("hot"), cfg.get("error")),
            ("accent", web.get("accent-primary"), cfg.get("accentBackground")),
            ("textPrimary", web.get("text"), cfg.get("textPrimary")),
            ("pageBackground", web.get("page"), cfg.get("pageBackground")),
        ):
            if not w or not h:
                failures.append(cfg_name + "." + key + " 缺失")
            elif rgb(w) != rgb(h):
                failures.append(cfg_name + "." + key + " 与 web 不一致：web " + w + " / harmony " + h)
        for cat, field in (
            ("accent", "accentOnSoft"),
            ("online", "onlineOnSoft"),
            ("warning", "warningOnSoft"),
            ("danger", "dangerOnSoft"),
        ):
            w = web.get(HEX_KEYS[cat])
            tp = web.get("text")
            h = cfg.get(field)
            if not w or not tp or not h:
                failures.append(cfg_name + "." + field + " 缺失")
                continue
            expected = mix(rgb(w), rgb(tp))
            if any(abs(a - b) > 1 for a, b in zip(expected, rgb(h))):
                failures.append(cfg_name + "." + field + " 与 80/20 混色不一致：期望 " + tohex(expected) + " / harmony " + h)
            page = web.get("page")
            if not page:
                continue
            for alpha in (ALPHA[cat][mode], ALPHA[cat][mode] * 0.5):
                bg = blend(rgb(w), alpha, rgb(page))
                r = ratio(rgb(h), bg)
                k = (cfg_name, cat, "%.3f" % alpha)
                worst[k] = min(worst.get(k, 99.0), r)
                if r < 4.5:
                    failures.append(cfg_name + " " + cat + " on-soft vs soft-bg(" + ("%.0f%%" % (alpha * 100)) + ") 对比度 %.2f < 4.5" % r)
    for k, r in sorted(worst.items()):
        rows.append("%-20s %-8s alpha=%s worst=%.2f" % (k[0], k[1], k[2], r))
    print("跨端 on-soft 状态色校验（web palette/text-primary <-> harmony ThemeConstants）")
    print("最差软底对比度（含 danger /50 行内按钮）：")
    for row in rows:
        print("  " + row)
    if failures:
        print("")
        print("FAIL:")
        for f in failures:
            print("  - " + f)
        return 1
    print("")
    print("结果: PASS（6 配置 x 4 语义色 = 24 项 on-soft 与对比度断言全部通过）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
