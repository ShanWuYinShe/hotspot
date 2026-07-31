// 源码审计：扫描 React/Next 残留 import，确保迁移彻底。
// 用法：bun run verify（宿主机，无需 Docker）
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";

const sourceRoot = "src";
const sourceExtensions = new Set([".ts", ".svelte", ".tsx", ".js", ".mjs"]);
const forbiddenPatterns = [
  { pattern: /from\s*["'](?:next|next\/[^"']*|react|react\/[^"']*|motion|motion\/[^"']*|lucide-react|@\/contexts|@\/hooks)["']/g, label: "React/Next/motion 残留 import" },
  { pattern: /NEXT_PUBLIC_/g, label: "NEXT_PUBLIC_ 环境变量残留" },
  { pattern: /\busePathname\b|\buseRouter\b|\buseSearchParams\b/g, label: "next/navigation hook 残留" },
  { pattern: /generateStaticParams|force-static/g, label: "Next 静态导出约束残留" },
];

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    if (entry.name === "node_modules" || entry.name.startsWith(".")) continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) files.push(...await walk(path));
    else if (sourceExtensions.has(`.${entry.name.split(".").pop()}`)) files.push(path);
  }
  return files;
}

const files = await walk(sourceRoot);
const errors = [];

for (const file of files) {
  const source = await readFile(file, "utf8");
  const display = relative(process.cwd(), file);
  for (const { pattern, label } of forbiddenPatterns) {
    for (const match of source.matchAll(pattern)) {
      errors.push(`${display}: ${label} — "${match[0].trim()}"`);
    }
  }
}

if (errors.length) {
  console.error("\n源码审计失败：存在 React/Next 残留：\n");
  for (const error of [...new Set(errors)].slice(0, 50)) console.error(`- ${error}`);
  process.exit(1);
}

console.log(`源码审计通过：${files.length} 个源文件无 React/Next 残留。`);
