#!/usr/bin/env node
// a11y 报告汇总：读取 raw/a11y.jsonl + raw/reduced.jsonl，输出 summary.txt / report.json。
// 退出码：serious/critical violation > 0 → 1；axe 未注入/执行错误 → 1；reduced 模式有残留 → 1。
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';

const outDir = process.argv[2];
const reducedMode = process.argv[3] === '1';
if (!outDir) {
  console.error('用法: a11y-summary.mjs <outDir> [reduced=0|1]');
  process.exit(2);
}

function readJsonl(p) {
  if (!existsSync(p)) return [];
  return readFileSync(p, 'utf8')
    .split('\n')
    .filter((l) => l.trim())
    .map((l) => {
      try { return JSON.parse(l); } catch { return { parseError: l.slice(0, 200) }; }
    });
}

function coerce(v) {
  if (typeof v !== 'string') return v;
  try { return JSON.parse(v); } catch { return v; }
}

const a11yRows = readJsonl(join(outDir, 'raw', 'a11y.jsonl'));
const reducedRows = readJsonl(join(outDir, 'raw', 'reduced.jsonl'));

const impactCount = { minor: 0, moderate: 0, serious: 0, critical: 0 };
const issueMap = new Map();
const failures = [];
const errors = [];
const pages = [];

for (const row of a11yRows) {
  const axe = coerce(row.axe);
  if (!axe || axe.error) {
    errors.push({ route: row.route, error: axe && axe.error ? axe.error : 'axe 未返回结果' });
    continue;
  }
  const violations = Array.isArray(axe.violations) ? axe.violations : [];
  for (const v of violations) {
    impactCount[v.impact] = (impactCount[v.impact] || 0) + 1;
    const key = v.id + '|' + v.impact;
    issueMap.set(key, (issueMap.get(key) || 0) + 1);
  }
  const serious = violations.filter((v) => v.impact === 'serious' || v.impact === 'critical');
  if (serious.length > 0) failures.push({ route: row.route, violations: serious });
  pages.push({ slug: row.slug, route: row.route, vw: row.vw, theme: row.theme, mode: row.mode, violations: violations.length });
}

const reduced = [];
for (const row of reducedRows) {
  const r = coerce(row.reduced);
  if (!r || r.error) {
    reduced.push({ slug: row.slug, route: row.route, error: r && r.error ? r.error : '未返回结果' });
    continue;
  }
  const hasResidual = Array.isArray(r.residual) && r.residual.length > 0;
  const hasLong = (r.longCount || 0) > 0;
  reduced.push({ slug: row.slug, route: row.route, residual: r.residual || [], longCount: r.longCount || 0, animations: r.animations || 0, long: (r.long || []).slice(0, 5), ok: !hasResidual && !hasLong });
}

const seriousOrCritical = (impactCount.serious || 0) + (impactCount.critical || 0);
const reducedBad = reduced.filter((r) => r.ok === false);
const reducedErrors = reduced.filter((r) => r.error);
const ok = seriousOrCritical === 0 && errors.length === 0 && (!reducedMode || (reducedBad.length === 0 && reducedErrors.length === 0));

const report = { generatedAt: new Date().toISOString(), outDir, reducedMode, ok, pages, impactCount, issueMap: Object.fromEntries(issueMap), failures, errors, reduced };
writeFileSync(join(outDir, 'report.json'), JSON.stringify(report, null, 2));

const lines = [];
lines.push('HotSpot 可访问性审计（axe-core）');
lines.push('页面数: ' + pages.length + ' / 错误: ' + errors.length);
lines.push('impact: minor=' + (impactCount.minor || 0) + ' moderate=' + (impactCount.moderate || 0) + ' serious=' + (impactCount.serious || 0) + ' critical=' + (impactCount.critical || 0));
if (seriousOrCritical > 0) {
  lines.push('');
  lines.push('serious/critical 明细:');
  for (const f of failures) {
    for (const v of f.violations) lines.push('  - ' + f.route + ' [' + v.impact + '] ' + v.id + ' x' + v.nodes + ' : ' + (v.targets || []).join(' | '));
  }
}
if (errors.length > 0) {
  lines.push('');
  lines.push('错误:');
  for (const e of errors) lines.push('  - ' + e.route + ': ' + e.error);
}
if (reducedMode) {
  lines.push('');
  lines.push('reduced-motion 检查: 页面数=' + reduced.length + ' 异常=' + reducedBad.length + ' 错误=' + reducedErrors.length);
  for (const r of reduced) {
    if (r.error) lines.push('  - ' + r.route + ': ERROR ' + r.error);
    else if (!r.ok) lines.push('  - ' + r.route + ': residual=' + JSON.stringify(r.residual) + ' longCount=' + r.longCount + ' long=' + JSON.stringify(r.long));
  }
}
lines.push('');
lines.push(ok ? '结果: PASS' : '结果: FAIL');
const summary = lines.join('\n');
writeFileSync(join(outDir, 'summary.txt'), summary + '\n');
console.log(summary);
process.exit(ok ? 0 : 1);
