// ============================================================================
// HotSpot Design Kit — Figma 插件运行时（由 build-figma-plugin.mjs 打包进 code.js）
// 依赖全局 SPEC：{ meta, variables, svgDict, imageDict, gradientDict, shadowDict, screens }
// ============================================================================
var FAMILY_LATIN = 'Inter';
var FAMILY_CJK = 'PingFang SC';

var _fontsLoaded = {};
var _varMap = null;
var _collection = null;
var _modeIds = {};

function log(msg) {
  try { figma.notify(msg); } catch (e) {}
}

function hexToRgb(hex) {
  if (!hex || hex.charAt(0) !== '#') return { r: 0, g: 0, b: 0 };
  return {
    r: parseInt(hex.slice(1, 3), 16) / 255,
    g: parseInt(hex.slice(3, 5), 16) / 255,
    b: parseInt(hex.slice(5, 7), 16) / 255
  };
}

function parseColorValue(value) {
  var parts = String(value).split('|');
  var c = hexToRgb(parts[0]);
  var a = parts.length > 1 ? parseFloat(parts[1]) : 1;
  return { r: c.r, g: c.g, b: c.b, a: isNaN(a) ? 1 : a };
}

function getOrCreateCollection() {
  var collections = figma.variables.getLocalVariableCollections();
  for (var i = 0; i < collections.length; i++) if (collections[i].name === 'HotSpot Tokens') return collections[i];
  return figma.variables.createVariableCollection('HotSpot Tokens');
}

function ensureModes() {
  _collection = getOrCreateCollection();
  var modes = SPEC.meta.modes;
  var existing = {};
  for (var i = 0; i < _collection.modes.length; i++) existing[_collection.modes[i].name] = _collection.modes[i].modeId;
  if (!existing[modes[0]]) {
    _collection.renameMode(_collection.modes[0].modeId, modes[0]);
    existing[modes[0]] = _collection.modes[0].modeId;
  }
  for (var j = 0; j < modes.length; j++) {
    if (!existing[modes[j]]) existing[modes[j]] = _collection.addMode(modes[j]);
    _modeIds[modes[j]] = existing[modes[j]];
  }
}

function ensureVariables() {
  var all = figma.variables.getLocalVariables('COLOR');
  var byName = {};
  for (var i = 0; i < all.length; i++) {
    if (all[i].variableCollectionId === _collection.id) byName[all[i].name] = all[i];
  }
  _varMap = {};
  for (var k = 0; k < SPEC.variables.length; k++) {
    var def = SPEC.variables[k];
    var variable = byName[def.name];
    if (!variable) variable = figma.variables.createVariable(def.name, _collection, 'COLOR');
    for (var m = 0; m < SPEC.meta.modes.length; m++) {
      var mode = SPEC.meta.modes[m];
      var value = def.values[mode];
      if (value === undefined) continue;
      variable.setValueForMode(_modeIds[mode], parseColorValue(value));
    }
    _varMap[def.name] = variable;
  }
}

function tokenColor(name, alpha) {
  var a = alpha === undefined ? 1 : alpha;
  if (name && name.charAt(0) === '#') return { color: hexToRgb(name), variable: null, alpha: a };
  var variable = _varMap ? _varMap[name] : null;
  if (!variable) {
    var fallback = { r: 0.6, g: 0.6, b: 0.6 };
    var hex = SPEC.fallbacks && SPEC.fallbacks[name];
    if (hex) fallback = hexToRgb(hex);
    return { color: { r: fallback.r, g: fallback.g, b: fallback.b }, variable: null, alpha: a };
  }
  var base = { r: 0.6, g: 0.6, b: 0.6 };
  return { color: base, variable: variable, alpha: a };
}

function solidPaint(name, alpha) {
  var rec = tokenColor(name, alpha);
  var paint = { type: 'SOLID', color: rec.color, opacity: rec.alpha };
  if (rec.variable) {
    try { paint = figma.variables.setBoundVariableForPaint(paint, 'color', rec.variable); } catch (e) {}
  }
  return paint;
}

function gradientTransformForAngle(angle) {
  var rad = (angle || 180) * Math.PI / 180;
  var cos = Math.cos(rad);
  var sin = Math.sin(rad);
  return [
    [cos, sin, 0.5 - 0.5 * cos - 0.5 * sin],
    [-sin, cos, 0.5 + 0.5 * sin - 0.5 * cos]
  ];
}

function gradientPaint(grad) {
  var stops = [];
  for (var i = 0; i < grad.stops.length; i++) {
    var s = grad.stops[i];
    var rec = tokenColor(s.c, s.a);
    stops.push({ position: s.p, color: { r: rec.color.r, g: rec.color.g, b: rec.color.b, a: rec.alpha } });
  }
  if (grad.type === 'radial') {
    return { type: 'GRADIENT_RADIAL', gradientTransform: [[1, 0, 0], [0, 1, 0]], gradientStops: stops };
  }
  return { type: 'GRADIENT_LINEAR', gradientTransform: gradientTransformForAngle(grad.angle), gradientStops: stops };
}

function fillsFor(n) {
  var fills = [];
  if (n.f) fills.push(solidPaint(n.f, n.fa));
  if (n.g !== undefined && SPEC.gradientDict[n.g]) fills.push(gradientPaint(SPEC.gradientDict[n.g]));
  return fills;
}

function effectsFor(shadowId) {
  var effects = [];
  if (shadowId === undefined) return effects;
  var list = SPEC.shadowDict[shadowId];
  if (!list) return effects;
  for (var i = 0; i < list.length; i++) {
    var s = list[i];
    var rec = tokenColor(s.c, s.a);
    effects.push({
      type: 'DROP_SHADOW',
      color: { r: rec.color.r, g: rec.color.g, b: rec.color.b, a: rec.alpha },
      offset: { x: s.x, y: s.y },
      radius: s.blur,
      spread: s.spread,
      visible: true,
      blendMode: 'NORMAL'
    });
  }
  return effects;
}

function applyCommon(node, n) {
  var fills = fillsFor(n);
  if (fills.length) node.fills = fills;
  if (n.bd) {
    node.strokes = [solidPaint(n.bd[1], n.bd[2])];
    node.strokeWeight = n.bd[0];
    try { node.strokeAlign = 'INSIDE'; } catch (e) {}
  }
  if (n.sh !== undefined) node.effects = effectsFor(n.sh);
  if (n.o !== undefined) node.opacity = n.o;
}

function weightStyle(weight) {
  if (weight >= 700) return 'Bold';
  if (weight >= 600) return 'SemiBold';
  if (weight >= 500) return 'Medium';
  return 'Regular';
}

function loadFont(family, style) {
  var key = family + '|' + style;
  if (_fontsLoaded[key]) return _fontsLoaded[key];
  _fontsLoaded[key] = figma.loadFontAsync({ family: family, style: style }).then(
    function () { return { family: family, style: style }; },
    function () {
      if (style !== 'Regular') return loadFont(family, 'Regular');
      if (family !== FAMILY_LATIN) return loadFont(FAMILY_LATIN, 'Regular');
      return { family: 'Roboto', style: 'Regular' };
    }
  );
  return _fontsLoaded[key];
}

function getOrCreatePage(name) {
  var pages = figma.root.children;
  for (var i = 0; i < pages.length; i++) if (pages[i].name === name) return pages[i];
  var page = figma.createPage();
  page.name = name;
  return page;
}

function clearPage(page) {
  var children = page.children.slice();
  for (var i = 0; i < children.length; i++) children[i].remove();
}

function addSwatch(parent, x, y, name, colorName, size) {
  var rect = figma.createRectangle();
  rect.x = x; rect.y = y;
  rect.resize(size, size);
  rect.cornerRadius = 8;
  rect.fills = [solidPaint(colorName, 1)];
  parent.appendChild(rect);
  return rect;
}

// ---------- 文本与节点 ----------
async function addText(parent, opts) {
  var family = opts.cjk ? FAMILY_CJK : FAMILY_LATIN;
  var font = await loadFont(family, weightStyle(opts.weight || 400));
  var text = figma.createText();
  parent.appendChild(text);
  text.fontName = font;
  text.fontSize = opts.size || 14;
  text.characters = String(opts.text);
  if (opts.lineHeight) text.lineHeight = { unit: 'PIXELS', value: opts.lineHeight };
  if (opts.letterSpacing) text.letterSpacing = { unit: 'PIXELS', value: opts.letterSpacing };
  if (opts.align === 'center') text.textAlignHorizontal = 'CENTER';
  else if (opts.align === 'right') text.textAlignHorizontal = 'RIGHT';
  text.textAutoResize = 'NONE';
  text.resize(Math.max(1, opts.w), Math.max(1, opts.h));
  text.x = opts.x; text.y = opts.y;
  if (opts.color) {
    var paint = solidPaint(opts.color, opts.alpha === undefined ? 1 : opts.alpha);
    text.fills = [paint];
  }
  if (opts.opacity !== undefined) text.opacity = opts.opacity;
  return text;
}

var B64CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
function b64ToBytes(b64) {
  var clean = String(b64).replace(/[^A-Za-z0-9+/=]/g, '');
  var len = clean.length;
  var bytes = [];
  var i = 0;
  while (i < len) {
    var c1 = B64CHARS.indexOf(clean.charAt(i++));
    var c2 = B64CHARS.indexOf(clean.charAt(i++));
    var c3 = B64CHARS.indexOf(clean.charAt(i++));
    var c4 = B64CHARS.indexOf(clean.charAt(i++));
    var n1 = (c1 << 2) | (c2 >> 4);
    var n2 = ((c2 & 15) << 4) | (c3 >> 2);
    var n3 = ((c3 & 3) << 6) | c4;
    bytes.push(n1 & 255);
    if (c3 !== -1 && c3 !== 64) bytes.push(n2 & 255);
    if (c4 !== -1 && c4 !== 64) bytes.push(n3 & 255);
  }
  return new Uint8Array(bytes);
}

function createClipFrame(parent, cl, cache) {
  var key = cl.join(',');
  if (cache[key]) return cache[key];
  var frame = figma.createFrame();
  parent.appendChild(frame);
  frame.x = cl[0]; frame.y = cl[1];
  frame.resize(cl[2], cl[3]);
  frame.fills = [];
  frame.clipsContent = true;
  if (cl[4] || cl[5] || cl[6] || cl[7]) frame.cornerRadius = [cl[4], cl[5], cl[6], cl[7]];
  cache[key] = frame;
  return frame;
}

function svgColorHex(name) {
  if (!name) return null;
  if (name.charAt(0) === '#') return name;
  var hex = SPEC.tokenColors && SPEC.tokenColors[name];
  return hex || null;
}

function bindVectorColors(node, tokenName, alpha) {
  if (!tokenName || tokenName.charAt(0) === '#') return;
  var variable = _varMap ? _varMap[tokenName] : null;
  if (!variable) return;
  var all = node.findAll(function () { return true; });
  for (var i = 0; i < all.length; i++) {
    var child = all[i];
    if (child.fills && child.fills.length) {
      var nextFills = [];
      for (var f = 0; f < child.fills.length; f++) {
        var paint = child.fills[f];
        if (paint.type === 'SOLID') {
          try { paint = figma.variables.setBoundVariableForPaint(paint, 'color', variable); } catch (e) {}
        }
        nextFills.push(paint);
      }
      child.fills = nextFills;
    }
    if (child.strokes && child.strokes.length) {
      var nextStrokes = [];
      for (var s = 0; s < child.strokes.length; s++) {
        var stroke = child.strokes[s];
        if (stroke.type === 'SOLID') {
          try { stroke = figma.variables.setBoundVariableForPaint(stroke, 'color', variable); } catch (e) {}
        }
        nextStrokes.push(stroke);
      }
      child.strokes = nextStrokes;
    }
  }
}

async function addNode(parent, n, clipCache) {
  var target = parent;
  var tx = n.x;
  var ty = n.y;
  if (n.cl) {
    var wrap = createClipFrame(parent, n.cl, clipCache);
    target = wrap;
    tx = n.x - n.cl[0];
    ty = n.y - n.cl[1];
  }
  if (n.t === 'r') {
    var rect = figma.createRectangle();
    target.appendChild(rect);
    rect.resize(n.w, n.h);
    rect.x = tx; rect.y = ty;
    if (n.r) {
      var same = true;
      for (var ri = 1; ri < n.r.length; ri++) if (n.r[ri] !== n.r[0]) same = false;
      rect.cornerRadius = same ? n.r[0] : n.r;
    }
    applyCommon(rect, n);
    return;
  }
  if (n.t === 't') {
    await addText(target, {
      x: tx, y: ty, w: n.w, h: n.h, text: n.tx, size: n.s, weight: n.fw,
      lineHeight: n.lh || undefined, letterSpacing: n.ls || undefined,
      align: n.al || 'left', color: n.c, alpha: n.ca, cjk: !!n.cj, opacity: n.o
    });
    return;
  }
  if (n.t === 'i') {
    var image = SPEC.imageDict[n.im];
    var box = figma.createRectangle();
    target.appendChild(box);
    box.resize(n.w, n.h);
    box.x = tx; box.y = ty;
    if (n.r) box.cornerRadius = n.r[0];
    if (image && image.data) {
      try {
        var raw = String(image.data).split(',');
        var bytes = b64ToBytes(raw[raw.length - 1]);
        var figmaImage = figma.createImage(bytes);
        box.fills = [{ type: 'IMAGE', imageHash: figmaImage.hash, scaleMode: n.fit === 'contain' ? 'FIT' : 'FILL' }];
      } catch (e) {
        box.fills = [solidPaint('surface-subtle', 1)];
      }
    } else {
      box.fills = [solidPaint('surface-subtle', 1)];
    }
    if (n.o !== undefined) box.opacity = n.o;
    return;
  }
  if (n.t === 'v') {
    var svg = SPEC.svgDict[n.sv];
    if (!svg) return;
    var hex = svgColorHex(n.c) || '#888888';
    var prepared = svg.split('currentColor').join(hex);
    var vector;
    try { vector = figma.createNodeFromSvg(prepared); } catch (e) { return; }
    target.appendChild(vector);
    vector.resize(Math.max(1, n.w), Math.max(1, n.h));
    vector.x = tx; vector.y = ty;
    if (n.c) bindVectorColors(vector, n.c, n.ca === undefined ? 1 : n.ca);
    return;
  }
  if (n.t === 'c') {
    var placeholder = figma.createRectangle();
    target.appendChild(placeholder);
    placeholder.resize(n.w, n.h);
    placeholder.x = tx; placeholder.y = ty;
    placeholder.fills = [solidPaint('surface-subtle', 1)];
    if (n.r) placeholder.cornerRadius = n.r[0];
    return;
  }
}

async function buildScreen(parent, screen, x, y) {
  var frame = figma.createFrame();
  parent.appendChild(frame);
  frame.name = screen.id + ' · ' + screen.title;
  frame.x = x; frame.y = y;
  frame.resize(screen.w, screen.h);
  frame.clipsContent = true;
  frame.fills = [solidPaint('surface-page', 1)];
  var clipCache = {};
  for (var i = 0; i < screen.nodes.length; i++) {
    await addNode(frame, screen.nodes[i], clipCache);
    if (i % 120 === 119) await new Promise(function (r) { setTimeout(r, 0); });
  }
  return frame;
}

async function buildScreensPage(page, device) {
  figma.currentPage = page;
  clearPage(page);
  var screens = SPEC.screens.filter(function (s) { return s.device === device; });
  var columns = device === 'desktop' ? 1 : device === 'tablet' ? 2 : 3;
  var gapX = device === 'desktop' ? 80 : 96;
  var gapY = 120;
  var startX = 0;
  var startY = 0;
  var rowH = 0;
  for (var i = 0; i < screens.length; i++) {
    var col = i % columns;
    var row = Math.floor(i / columns);
    var screen = screens[i];
    var x = startX + col * (screen.w + gapX);
    var y = startY + row * (rowH + gapY);
    if (col === 0) rowH = screen.h;
    await buildScreen(page, screen, x, y);
    var label = await loadFont(FAMILY_LATIN, 'Medium');
    var labelNode = figma.createText();
    page.appendChild(labelNode);
    labelNode.fontName = label;
    labelNode.fontSize = 28;
    labelNode.characters = screen.title + '  ·  ' + screen.route;
    labelNode.x = x;
    labelNode.y = y - 40;
    labelNode.fills = [{ type: 'SOLID', color: hexToRgb('#52647f') }];
    if (i % 2 === 1) await new Promise(function (r) { setTimeout(r, 0); });
  }
}

// ---------- 设计规范页 ----------
function simpleRect(parent, x, y, w, h, fill, radius, border) {
  var rect = figma.createRectangle();
  parent.appendChild(rect);
  rect.x = x; rect.y = y;
  rect.resize(w, h);
  if (radius) rect.cornerRadius = radius;
  rect.fills = fill ? [solidPaint(fill, 1)] : [];
  if (border) {
    rect.strokes = [solidPaint(border, 1)];
    rect.strokeWeight = 1;
    try { rect.strokeAlign = 'INSIDE'; } catch (e) {}
  }
  return rect;
}

async function sectionTitle(parent, x, y, text) {
  return addText(parent, { x: x, y: y, w: 1600, h: 40, text: text, size: 30, weight: 700, color: 'text-primary' });
}

async function buildModePanel(parent, x, y, mode) {
  var panel = figma.createFrame();
  parent.appendChild(panel);
  panel.name = 'mode/' + mode;
  panel.x = x; panel.y = y;
  panel.resize(250, 210);
  panel.cornerRadius = 16;
  panel.fills = [solidPaint('surface-primary', 1)];
  try { panel.setExplicitVariableModeForCollection(_collection, _modeIds[mode]); } catch (e) {}
  await addText(panel, { x: 14, y: 12, w: 220, h: 20, text: mode.replace('-', ' · '), size: 13, weight: 600, color: 'text-primary' });
  var keys = ['surface-page', 'surface-primary', 'text-primary', 'accent-default', 'status-online', 'status-danger'];
  for (var i = 0; i < keys.length; i++) {
    var sx = 14 + (i % 3) * 72;
    var sy = 44 + Math.floor(i / 3) * 78;
    simpleRect(panel, sx, sy, 60, 40, keys[i], 10);
    await addText(panel, { x: sx, y: sy + 44, w: 70, h: 14, text: keys[i], size: 9, weight: 400, color: 'text-secondary' });
  }
  return panel;
}

async function buildDesignSystemPage(page) {
  figma.currentPage = page;
  clearPage(page);
  var y = 0;
  await addText(page, { x: 0, y: 0, w: 1800, h: 64, text: 'HotSpot 设计规范 · Design System', size: 44, weight: 700, color: 'text-primary' });
  await addText(page, { x: 0, y: 64, w: 1800, h: 28, text: '由前端代码 token 与实时页面结构生成；主题/明暗通过 Figma Variables 的 6 个模式切换。', size: 14, weight: 400, color: 'text-secondary' });
  y = 130;
  await sectionTitle(page, 0, y, '主题 × 明暗（6 模式）');
  y += 50;
  for (var mi = 0; mi < SPEC.meta.modes.length; mi++) await buildModePanel(page, mi * 280, y, SPEC.meta.modes[mi]);
  y += 250;
  await sectionTitle(page, 0, y, '语义色 Semantic Colors');
  y += 50;
  var colorVars = [];
  for (var vi = 0; vi < SPEC.variables.length; vi++) {
    var name = SPEC.variables[vi].name;
    if (name.indexOf('palette-') !== 0) colorVars.push(name);
  }
  var perRow = 6, cellW = 250, cellH = 62;
  for (var ci = 0; ci < colorVars.length; ci++) {
    var cx = (ci % perRow) * cellW;
    var cy = y + Math.floor(ci / perRow) * cellH;
    simpleRect(page, cx, cy, 56, 44, colorVars[ci], 10);
    await addText(page, { x: cx + 66, y: cy + 4, w: 180, h: 16, text: colorVars[ci], size: 11, weight: 600, color: 'text-primary' });
    await addText(page, { x: cx + 66, y: cy + 22, w: 180, h: 16, text: (SPEC.tokenColors[colorVars[ci]] || ''), size: 10, weight: 400, color: 'text-secondary' });
  }
  y += Math.ceil(colorVars.length / perRow) * cellH + 50;
  await sectionTitle(page, 0, y, '字号 Typography');
  y += 50;
  var sizes = [10, 11, 12, 14, 18, 20, 24, 30, 36];
  for (var si = 0; si < sizes.length; si++) await addText(page, { x: 0, y: y + si * 46, w: 700, h: 40, text: 'HotSpot 城市此刻 / ' + sizes[si] + 'px', size: sizes[si], weight: si > 5 ? 700 : 500, color: 'text-primary' });
  y += sizes.length * 46 + 40;
  await sectionTitle(page, 0, y, '圆角 Radius');
  y += 50;
  var radii = [12, 16, 20, 24, 28];
  for (var r2 = 0; r2 < radii.length; r2++) {
    simpleRect(page, r2 * 120, y, 90, 72, 'accent-soft', radii[r2], 'accent-border');
    await addText(page, { x: r2 * 120, y: y + 80, w: 100, h: 16, text: 'radius ' + radii[r2], size: 11, weight: 500, color: 'text-secondary' });
  }
  y += 130;
  await sectionTitle(page, 0, y, '组件 Components');
  y += 50;
  await buildComponents(page, y);
}

async function makeComponent(parent, name, w, h, x, y) {
  var comp = figma.createComponent();
  parent.appendChild(comp);
  comp.name = name;
  comp.resize(w, h);
  comp.x = x; comp.y = y;
  comp.fills = [];
  return comp;
}

async function buildComponents(page, startY) {
  var y = startY;
  var tones = [
    ['primary', 'accent-default', 'text-on-accent', null],
    ['secondary', 'surface-primary', 'text-primary', 'border-default'],
    ['soft', 'accent-soft', 'text-accent-on-soft', null],
    ['ghost', null, 'text-secondary', null],
    ['danger', 'status-danger', 'text-on-status', null]
  ];
  var sizes = [['SM', 32, 12], ['MD', 42, 13], ['LG', 48, 14]];
  for (var ti = 0; ti < tones.length; ti++) {
    for (var sz = 0; sz < sizes.length; sz++) {
      var t = tones[ti], s = sizes[sz];
      var comp = await makeComponent(page, 'Button/' + t[0] + '/' + s[0], 130, s[1], ti * 150, y + sz * 60);
      simpleRect(comp, 0, 0, 130, s[1], t[1], 999, t[3]);
      await addText(comp, { x: 0, y: (s[1] - 18) / 2, w: 130, h: 18, text: '按钮 Button', size: s[2], weight: 600, align: 'center', color: t[2] });
    }
  }
  y += 200;
  var states = [['default', 'field-border', '输入内容'], ['focus', 'accent-default', '聚焦状态'], ['error', 'status-danger', '格式不正确'], ['disabled', 'border-subtle', '不可编辑']];
  for (var ii = 0; ii < states.length; ii++) {
    var inp = await makeComponent(page, 'Input/' + states[ii][0], 220, 43, ii * 240, y);
    simpleRect(inp, 0, 0, 220, 43, 'field-bg', 16, states[ii][1]);
    await addText(inp, { x: 14, y: 12, w: 190, h: 20, text: states[ii][2], size: 13, weight: 400, color: 'text-secondary' });
  }
  y += 90;
  var badges = [['accent', 'accent-bg', 'text-accent-on-soft'], ['online', 'status-online-bg', 'text-status-online-on-soft'], ['warning', 'status-warning-bg', 'text-status-warning-on-soft'], ['danger', 'status-danger-bg', 'text-status-danger-on-soft']];
  for (var bi = 0; bi < badges.length; bi++) {
    var badge = await makeComponent(page, 'Badge/' + badges[bi][0], 88, 24, bi * 110, y);
    simpleRect(badge, 0, 0, 88, 24, badges[bi][1], 999);
    await addText(badge, { x: 0, y: 4, w: 88, h: 16, text: badges[bi][0], size: 11, weight: 600, align: 'center', color: badges[bi][2] });
  }
  y += 60;
  var tabs = await makeComponent(page, 'Tabs/Pills', 300, 34, 0, y);
  simpleRect(tabs, 0, 0, 300, 34, 'surface-subtle', 17);
  simpleRect(tabs, 4, 3, 94, 28, 'accent-default', 14);
  await addText(tabs, { x: 4, y: 9, w: 94, h: 16, text: '全部', size: 11, weight: 600, align: 'center', color: 'text-on-accent' });
  await addText(tabs, { x: 102, y: 9, w: 94, h: 16, text: '附近', size: 11, weight: 500, align: 'center', color: 'text-secondary' });
  await addText(tabs, { x: 200, y: 9, w: 94, h: 16, text: '关注', size: 11, weight: 500, align: 'center', color: 'text-secondary' });
  var underline = await makeComponent(page, 'Tabs/Underline', 260, 36, 320, y);
  await addText(underline, { x: 0, y: 8, w: 60, h: 20, text: '简介', size: 13, weight: 600, color: 'text-primary' });
  simpleRect(underline, 0, 32, 60, 2, 'accent-default', 1);
  await addText(underline, { x: 80, y: 8, w: 60, h: 20, text: '成员', size: 13, weight: 400, color: 'text-secondary' });
  await addText(underline, { x: 160, y: 8, w: 60, h: 20, text: '公告', size: 13, weight: 400, color: 'text-secondary' });
  y += 60;
  var row = await makeComponent(page, 'List/Row', 360, 64, 0, y);
  simpleRect(row, 0, 0, 360, 64, 'surface-island', 16);
  var avatar = figma.createEllipse(); row.appendChild(avatar);
  avatar.x = 12; avatar.y = 12; avatar.resize(40, 40);
  avatar.fills = [solidPaint('accent-soft', 1)];
  await addText(row, { x: 64, y: 10, w: 240, h: 20, text: '鹿鸣', size: 14, weight: 600, color: 'text-primary' });
  await addText(row, { x: 64, y: 32, w: 240, h: 18, text: '云龙湖东岸 · 2 周前', size: 11, weight: 400, color: 'text-secondary' });
  y += 90;
  var card = await makeComponent(page, 'Card/Feed', 240, 300, 0, y);
  simpleRect(card, 0, 0, 240, 300, 'surface-primary', 20);
  simpleRect(card, 0, 0, 240, 220, 'surface-subtle', 20);
  await addText(card, { x: 14, y: 232, w: 210, h: 20, text: '云龙湖的落日把水面染成金色', size: 14, weight: 600, color: 'text-primary' });
  await addText(card, { x: 14, y: 260, w: 210, h: 18, text: '#云龙湖 · 云龙湖东岸', size: 11, weight: 400, color: 'text-secondary' });
  y += 330;
  var dock = await makeComponent(page, 'Navigation/Dock', 360, 64, 0, y);
  simpleRect(dock, 0, 0, 360, 64, 'surface-island', 28, 'border-default');
  var labels = ['首页', '附近', '发布', '消息', '我的'];
  for (var di = 0; di < labels.length; di++) {
    var dx = 24 + di * 74;
    if (di === 2) {
      var fab = figma.createEllipse(); dock.appendChild(fab);
      fab.x = dx; fab.y = 8; fab.resize(48, 48);
      fab.fills = [solidPaint('accent-default', 1)];
      await addText(dock, { x: dx, y: 20, w: 48, h: 24, text: '+', size: 20, weight: 400, align: 'center', color: 'text-on-accent' });
    } else {
      await addText(dock, { x: dx, y: 24, w: 40, h: 16, text: labels[di], size: 10, weight: 500, align: 'center', color: di === 0 ? 'accent-default' : 'text-secondary' });
    }
  }
  y += 110;
  var sheet = await makeComponent(page, 'Sheet/Bottom', 360, 220, 0, y);
  simpleRect(sheet, 0, 0, 360, 220, 'surface-panel', [28, 28, 0, 0]);
  simpleRect(sheet, 160, 10, 40, 4, 'border-default', 2);
  await addText(sheet, { x: 20, y: 30, w: 300, h: 24, text: '封禁并移出', size: 18, weight: 700, color: 'text-primary' });
  await addText(sheet, { x: 20, y: 62, w: 320, h: 42, text: '确定将该用户封禁并移出群组？封禁后对方将无法再加入。', size: 12, weight: 400, color: 'text-secondary' });
  simpleRect(sheet, 20, 160, 150, 42, 'surface-subtle', 999);
  await addText(sheet, { x: 20, y: 172, w: 150, h: 18, text: '取消', size: 13, weight: 600, align: 'center', color: 'text-primary' });
  simpleRect(sheet, 190, 160, 150, 42, 'status-danger', 999);
  await addText(sheet, { x: 190, y: 172, w: 150, h: 18, text: '封禁并移出', size: 13, weight: 600, align: 'center', color: 'text-on-status' });
  y += 260;
  var toast = await makeComponent(page, 'Toast', 320, 48, 0, y);
  simpleRect(toast, 0, 0, 320, 48, 'surface-panel', 16, 'border-default');
  await addText(toast, { x: 16, y: 14, w: 288, h: 20, text: '「用户管理」请在 Web 管理后台完成', size: 12, weight: 400, color: 'text-primary' });
  y += 80;
  var empty = await makeComponent(page, 'Empty State', 240, 140, 0, y);
  var circle = figma.createEllipse(); empty.appendChild(circle);
  circle.x = 90; circle.y = 8; circle.resize(60, 60);
  circle.fills = [solidPaint('accent-soft', 1)];
  await addText(empty, { x: 0, y: 78, w: 240, h: 20, text: '暂无内容', size: 14, weight: 600, align: 'center', color: 'text-primary' });
  await addText(empty, { x: 0, y: 102, w: 240, h: 18, text: '换个筛选条件试试', size: 11, weight: 400, align: 'center', color: 'text-secondary' });
}

// ---------- 入口 ----------
(function () {
  (async function () {
    var started = Date.now();
    try {
      ensureModes();
      ensureVariables();
      log('HotSpot：token 变量已就绪（' + SPEC.variables.length + ' 个 × ' + SPEC.meta.modes.length + ' 模式）');
      await buildDesignSystemPage(getOrCreatePage('01 · 设计规范 Design System'));
      log('设计规范页已生成');
      var labels = { mobile: '02 · 移动端 Mobile', tablet: '03 · 平板端 Tablet', desktop: '04 · 桌面端 Desktop' };
      var devices = SPEC.meta.devices;
      for (var di = 0; di < devices.length; di++) {
        var device = devices[di];
        var page = getOrCreatePage(labels[device] || device);
        await buildScreensPage(page, device);
        figma.currentPage = page;
        log('HotSpot：' + labels[device] + ' 已生成');
        await new Promise(function (r) { setTimeout(r, 0); });
      }
      log('HotSpot：全部页面生成完成，用时 ' + Math.round((Date.now() - started) / 1000) + 's');
    } catch (e) {
      figma.notify('HotSpot 设计稿生成失败：' + e.message, { error: true, timeout: 15000 });
      try { console.error('HOTSPOT-PLUGIN-ERROR', e && e.stack ? e.stack : e); } catch (err) {}
    }
    figma.closePlugin();
  })();
})();
