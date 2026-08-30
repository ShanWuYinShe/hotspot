#!/usr/bin/env python3
"""批量把 SymbolSpan 包装块转换为 LucideIcon 调用。

规则（仅处理可识别的统一模式）：
  Text() {
    SymbolSpan($r(EXPR))
      .fontSize(SIZE)
      .fontColor([COLOR])
  }
→
  LucideIcon({
    icon: EXPR,
    size: SIZE,
    color: COLOR
  })

- EXPR 可以是 AppIcon.XXX / this.icon / 条件表达式（icon 参数直接透传）
- COLOR 来自 fontColor([...]) 去方括号；无 fontColor 时省略 color（LucideIcon 默认 textSecondary）
- 每个文件自动补 import LucideIcon（已 import 或已用则跳过）
- 保留块外层 Text 的缩进层级
"""
import re
import glob
import sys

BASE = 'entry/src/main/ets'

# Text(){ SymbolSpan($r(EXPR)) .fontSize(...) .fontColor([...]) } 块
# 匹配：Text() { 换行 缩进 SymbolSpan($r(EXPR)) 换行 缩进 .fontSize(SIZE) 换行 缩进 .fontColor([COLOR]) 换行 缩进 }
BLOCK_RE = re.compile(
    r'Text\(\)\s*\{\s*\n'
    r'(?P<inner_indent>[ \t]*)SymbolSpan\(\$r\((?P<expr>[^)]+)\)\)\s*\n'
    r'(?P<fz_indent>[ \t]*)\.fontSize\((?P<size>[^)]+)\)\s*\n'
    r'(?P<fc_indent>[ \t]*)\.fontColor\(\[(?P<color>[^\]]+)\]\)\s*\n'
    r'(?P<end_indent>[ \t]*)\}',
    re.MULTILINE
)

# 无 fontColor 的块（只 fontSize）
BLOCK_NO_COLOR_RE = re.compile(
    r'Text\(\)\s*\{\s*\n'
    r'(?P<inner_indent>[ \t]*)SymbolSpan\(\$r\((?P<expr>[^)]+)\)\)\s*\n'
    r'(?P<fz_indent>[ \t]*)\.fontSize\((?P<size>[^)]+)\)\s*\n'
    r'(?P<end_indent>[ \t]*)\}',
    re.MULTILINE
)

def rel_import_path(path: str) -> str:
    """计算从文件到 components/LucideIcon 的相对路径前缀。"""
    depth = path.replace(BASE + '/', '').count('/')
    if depth == 0:
        return './LucideIcon'
    return '../' * depth + 'components/LucideIcon'

def ensure_import(src: str, path: str) -> str:
    if 'import { LucideIcon }' in src:
        return src
    imp = f"import {{ LucideIcon }} from '{rel_import_path(path)}';"
    # 插到最后一个 import 行之后
    lines = src.split('\n')
    last_import = -1
    for i, line in enumerate(lines):
        if line.strip().startswith('import '):
            last_import = i
    if last_import >= 0:
        lines.insert(last_import + 1, imp)
    else:
        lines.insert(0, imp)
    return '\n'.join(lines)

def transform(path: str) -> int:
    src = open(path).read()
    orig = src
    count = 0

    def repl(m):
        nonlocal count
        count += 1
        inner = m.group('inner_indent')
        expr = m.group('expr').strip()
        size = m.group('size').strip()
        color = m.group('color').strip()
        # 多层调用（this.getPalette().xxx）可能换行，去掉换行美化
        color = re.sub(r'\s+', ' ', color)
        fields = [f"icon: {expr}", f"size: {size}", f"color: {color}"]
        body = ',\n'.join(f"{inner}  {f}" for f in fields)
        return f"LucideIcon({{\n{body}\n{inner}}})"

    src, n1 = BLOCK_RE.subn(repl, src)
    count += n1

    def repl_nc(m):
        nonlocal count
        count += 1
        inner = m.group('inner_indent')
        expr = m.group('expr').strip()
        size = m.group('size').strip()
        body = ',\n'.join(f"{inner}  {f}" for f in [f"icon: {expr}", f"size: {size}"])
        return f"LucideIcon({{\n{body}\n{inner}}})"

    src, n2 = BLOCK_NO_COLOR_RE.subn(repl_nc, src)
    count += n2

    if src != orig:
        src = ensure_import(src, path)
        open(path, 'w').write(src)
    return count

def main():
    files = sorted(glob.glob(f'{BASE}/**/*.ets', recursive=True))
    files = [f for f in files if 'LucideIcon.ets' not in f and 'SystemIcon.ets' not in f]
    total = 0
    for f in files:
        n = transform(f)
        if n:
            total += n
            print(f'{n:3d}  {f}')
    print(f'TOTAL: {total}')

if __name__ == '__main__':
    main()
