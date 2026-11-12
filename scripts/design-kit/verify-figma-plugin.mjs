#!/usr/bin/env node
/**
 * 无 Figma 环境下的插件逻辑冒烟测试：用最小 mock 的 figma API 运行生成产物，
 * 验证插件不会抛错、页面/组件/屏幕节点数量符合预期。
 * 用法：node scripts/design-kit/verify-figma-plugin.mjs
 */
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import vm from 'node:vm';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const code = readFileSync(join(ROOT, 'docs', 'design-kit', 'figma-plugin', 'code.js'), 'utf8');

const stats = { nodes: 0, texts: 0, images: 0, components: 0, svgs: 0, variables: 0, notify: [], errors: [] };

function makeNode(type, extra) {
  const node = Object.assign({
    type: type,
    name: type,
    x: 0, y: 0, width: 100, height: 100,
    fills: [], strokes: [], effects: [], children: [],
    opacity: 1, cornerRadius: 0, clipsContent: false,
    resize(w, h) { node.width = w; node.height = h; },
    appendChild(child) { node.children.push(child); child.parent = node; stats.nodes++; },
    remove() { if (node.parent) { const i = node.parent.children.indexOf(node); if (i >= 0) node.parent.children.splice(i, 1); } },
    findAll(fn) {
      const out = [];
      const walk = (n) => { for (const c of n.children) { if (fn(c)) out.push(c); walk(c); } };
      walk(node);
      return out;
    },
    setExplicitVariableModeForCollection() {},
    setPluginData() {}
  }, extra || {});
  return node;
}

let variableSeq = 0;
const collections = [];
function makeCollection(name) {
  return {
    id: 'collection-' + (++variableSeq),
    name,
    modes: [{ modeId: 'mode-' + (++variableSeq), name: 'Mode 1' }],
    renameMode(modeId, newName) { const m = this.modes.find((x) => x.modeId === modeId); if (m) m.name = newName; },
    addMode(modeName) { const id = 'mode-' + (++variableSeq); this.modes.push({ modeId: id, name: modeName }); return id; }
  };
}
const variables = [];
const figma = {
  root: { children: [] },
  currentPage: null,
  createPage() { const page = makeNode('PAGE'); page.name = 'Page'; figma.root.children.push(page); return page; },
  createFrame() { return makeNode('FRAME'); },
  createComponent() { stats.components++; return makeNode('COMPONENT'); },
  createRectangle() { return makeNode('RECTANGLE'); },
  createEllipse() { return makeNode('ELLIPSE'); },
  createText() { stats.texts++; const t = makeNode('TEXT'); t.characters = ''; t.fontName = null; t.fontSize = 12; t.lineHeight = null; t.letterSpacing = null; t.textAlignHorizontal = 'LEFT'; t.textAutoResize = 'NONE'; return t; },
  createNodeFromSvg() { stats.svgs++; const f = makeNode('FRAME'); f.children = [makeNode('VECTOR')]; return f; },
  createImage() { stats.images++; return { hash: 'image-' + stats.images }; },
  loadFontAsync() { return Promise.resolve(); },
  notify(msg) { stats.notify.push(String(msg)); },
  closePlugin() { globalThis.__closed = true; },
  variables: {
    getLocalVariableCollections() { return collections; },
    createVariableCollection(name) { const c = makeCollection(name); collections.push(c); return c; },
    getLocalVariables() { return variables; },
    createVariable(name, collection) {
      const v = { id: 'var-' + (++variableSeq), name, variableCollectionId: collection.id, values: {}, setValueForMode(modeId, value) { this.values[modeId] = value; } };
      variables.push(v);
      stats.variables++;
      return v;
    },
    setBoundVariableForPaint(paint) { return paint; }
  }
};

globalThis.figma = figma;
globalThis.setTimeout = setTimeout;

const context = vm.createContext({ figma, setTimeout, console, Promise, Uint8Array, parseInt, parseFloat, isNaN, String, Math, Date, JSON, Array, Object });
try {
  vm.runInContext(code, context, { filename: 'code.js', timeout: 120000 });
} catch (e) {
  stats.errors.push('run error: ' + e.message + '\n' + (e.stack || ''));
}

setTimeout(() => {
  const pages = figma.root.children.map((p) => p.name);
  const screenNames = [];
  for (const page of figma.root.children) for (const child of page.children) if (child.type === 'FRAME' && child.name.indexOf(' · ') > 0) screenNames.push(child.name);
  console.log(JSON.stringify({
    closed: !!globalThis.__closed,
    pages,
    screens: screenNames.length,
    components: stats.components,
    texts: stats.texts,
    images: stats.images,
    svgs: stats.svgs,
    variables: stats.variables,
    notify: stats.notify.slice(0, 6),
    errors: stats.errors
  }, null, 1));
  if (stats.errors.length || !globalThis.__closed) process.exit(1);
}, 3000);
