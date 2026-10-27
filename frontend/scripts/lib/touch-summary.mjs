#!/usr/bin/env node
// 触控报告汇总：读取 raw/touch.jsonl + allowlist，输出 summary.txt / report.json。
// 口径（与 r5 一致）：有效热区 <44 为离群；表单控件 / skip link 排除出分母；
// allowlist 命中的离群为登记例外（仍在分母内，不计失败）；未命中 allowlist 的离群 → 退出 1。
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';

const outDir = process.argv[2];
const allowlistPath = process.argv[3];
if (!outDir || !allowlistPath) {
  console.error('用法: touch-summary.mjs <outDir> <allowlist.json>');
  process.exit(2);
}

const allowObj = JSON.parse(readFileSync(allowlistPath, 'utf8'));
const rules = Array.isArray(allowObj.rules) ? allowObj.rules : [];
const ruleMap = new Map(rules.map((r) => [r.id, r]));

function readJsonl(p) {
  if (!existsSync(p)) return [];
  return readFileSync(p, 'utf8')
    .split('\n')
    .filter((l) => l.trim())
    .map((l) => { try { return JSON.parse(l); } catch { return { parseError: l.slice(0, 200) }; } });
}

function coerce(v) {
  if (typeof v !== 'string') return v;
  try { return JSON.parse(v); } catch { return v; }
}

const rows = readJsonl(join(outDir, 'raw', 'touch.jsonl'));
const pages = [];
const unmatched = [];
const exceptions = [];
const excluded = [];
let total = 0;
let pass = 0;
let denom = 0;

for (const row of rows) {
  const els = coerce(row.elements);
  if (!Array.isArray(els)) {
    pages.push({ slug: row.slug, route: row.route, error: '扫描未返回数组' });
    continue;
  }
  let pageUnmatched = 0;
  let pageExc = 0;
  let pagePass = 0;
  for (const el of els) {
    total += 1;
    const rule = el.allow ? ruleMap.get(el.allow) : null;
    const isExcluded = el.form || el.skip || (rule && rule.exclude === true);
    if (isExcluded) { excluded.push({ route: row.route, ...el }); continue; }
    denom += 1;
    if (el.eff >= 44) { pass += 1; pagePass += 1; continue; }
    if (rule) { exceptions.push({ route: row.route, rule: rule.id, reason: rule.reason, ...el }); pageExc += 1; continue; }
    unmatched.push({ route: row.route, ...el });
    pageUnmatched += 1;
  }
  pages.push({ slug: row.slug, route: row.route, total: els.length, pass: pagePass, exceptions: pageExc, unmatched: pageUnmatched });
}

const ok = rows.length > 0 && unmatched.length === 0 && pages.every((p) => !p.error);
const coverage = denom > 0 ? Math.round(((denom - unmatched.length - exceptions.length) / denom) * 1000) / 10 : 0;
const report = { generatedAt: new Date().toISOString(), outDir, ok, pages, total, denominator: denom, pass, exceptions, unmatched, excluded: excluded.length, coveragePct: coverage };
writeFileSync(join(outDir, 'report.json'), JSON.stringify(report, null, 2));

const lines = [];
lines.push('HotSpot 触控热区审计（390x844）');
lines.push('页面数: ' + pages.length + ' 扫描元素: ' + total + ' 计入分母: ' + denom);
lines.push('≥44 通过: ' + pass + ' 登记例外: ' + exceptions.length + ' 排除(表单/skip link): ' + excluded.length);
lines.push('覆盖率(含例外): ' + coverage + '%');
if (unmatched.length > 0) {
  lines.push('');
  lines.push('未登记离群（需要修复或加 allowlist）:');
  for (const u of unmatched) lines.push('  - ' + u.route + ' ' + u.tag + ' "' + (u.text || '') + '" ' + u.w + 'x' + u.h + ' eff=' + u.eff + ' slop=' + (u.slop || '-') + ' cls=' + (u.cls || ''));
}
lines.push('');
lines.push(ok ? '结果: PASS' : '结果: FAIL');
const summary = lines.join('\n');
writeFileSync(join(outDir, 'summary.txt'), summary + '\n');
console.log(summary);
process.exit(ok ? 0 : 1);
