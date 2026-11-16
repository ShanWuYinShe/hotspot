// ============================================================================
// HotSpot Design Kit — 设计稿巡检脚本（只读，不修改任何节点）
//
// 用途：Figma 同步（插件生成）完成后立刻跑一遍，输出「同行基准线 / 页头居中」偏差列表，
//       作为同步后的质量门禁；有 violations 时按 hint 修复后重跑，直到 0。
//
// 运行（MCP 通道，脚本在 Figma 服务端沙盒执行）：
//   FILE_KEY="<fileKey>" SESSION_ID="<sessionId>" \
//   bash .workwork/figma-capture/use-figma.sh \
//     scripts/design-kit/audit-figma-design.js "design audit" "resource:figma-use"
//
// 规则：
//   tab-row-baseline 同一行 Tab/Chip/筛选标签（含选中胶囊与数字）字形中心必须一致（<=2px）
//   tab-row-va-mixed 同一行标签 textAlignVertical 混用（CENTER 与 TOP 并存）——即使当前看似对齐，
//                    字体/尺寸一变就会漂移，必须统一
//   header-center    页头标题与返回按钮同轴：单行标题=字形中心对齐按钮中心；
//                    标题+副标题两行块=整块中心对齐按钮中心（<=2px）
//
// 返回值：{ summary, violations, warnings }（violations 为空即通过）
// ============================================================================
var ROW_LIMIT = 2.0;      // 同行字形中心允许偏差（px）
var HEADER_LIMIT = 2.0;   // 页头中心允许偏差（px）
var MAX_ITEMS = 160;      // 每类最多输出条数，避免超出 MCP 返回值上限

function absPos(n) {
  var x = 0, y = 0, c = n;
  while (c && c.type !== 'PAGE') { x += c.x || 0; y += c.y || 0; c = c.parent; }
  return { x: x, y: y };
}

function renderBox(n) {
  try {
    var r = n.absoluteRenderBounds;
    if (!r) return null;
    return { x: r.x, y: r.y, w: r.width, h: r.height, cy: r.y + r.height / 2, bottom: r.y + r.height };
  } catch (e) { return null; }
}

function isCircleButton(n) {
  if (n.type !== 'RECTANGLE' && n.type !== 'ELLIPSE') return false;
  if (!(n.width >= 30 && n.width <= 54)) return false;
  if (Math.abs(n.width - n.height) > 6) return false;
  return true;
}

function isPillRect(n) {
  if (n.type !== 'RECTANGLE') return false;
  if (typeof n.cornerRadius !== 'number') return false;
  if (!(n.cornerRadius >= 6)) return false;
  if (!(n.height >= 20 && n.height <= 34)) return false;
  if (!(n.width >= 36 && n.width <= 200)) return false;
  return true;
}

function round1(v) { return Math.round(v * 10) / 10; }

var violations = [];
var warnings = [];
var stats = { pages: 0, screens: 0, rows: 0, headers: 0 };

var allPages = [];
for (var pi = 0; pi < figma.root.children.length; pi++) {
  if (figma.root.children[pi].type === 'PAGE') allPages.push(figma.root.children[pi]);
}

for (var pi2 = 0; pi2 < allPages.length; pi2++) {
  var page = allPages[pi2];
  await figma.setCurrentPageAsync(page);
  stats.pages++;
  var screens = page.children || [];
  for (var si = 0; si < screens.length; si++) {
    var fr = screens[si];
    if (fr.type !== 'FRAME') continue;
    stats.screens++;
    var kids = fr.children || [];
    var texts = [];
    var pills = [];
    var buttons = [];
    for (var ki = 0; ki < kids.length; ki++) {
      var k = kids[ki];
      if (k.visible === false) continue;
      if (k.type === 'TEXT' && String(k.characters || '').trim()) {
        var box = renderBox(k);
        if (box) texts.push({ node: k, box: box, text: String(k.characters), fs: k.fontSize || 0, va: k.textAlignVertical });
      } else if (isPillRect(k)) {
        var pbox = renderBox(k);
        if (pbox) pills.push({ node: k, box: pbox });
      }
      if (isCircleButton(k)) {
        var bbox = renderBox(k);
        if (bbox) buttons.push({ node: k, box: bbox });
      }
    }

    // ---------- 规则 1/2：胶囊标签行 ----------
    for (var pj = 0; pj < pills.length; pj++) {
      var pill = pills[pj];
      var inRow = [];
      for (var tj = 0; tj < texts.length; tj++) {
        var t = texts[tj];
        if (t.text.length > 8) continue;
        if (Math.abs(t.box.cy - pill.box.cy) > 14) continue;
        if (t.box.x < pill.box.x - 40) continue;
        if (t.box.x > pill.box.x + pill.box.w + 240) continue;
        inRow.push(t);
      }
      // 该胶囊行至少要有 2 个标签才判定为 Tab/Chip 行（排除单标签徽章）
      if (inRow.length < 2) continue;
      var fsBase = null;
      var sameSize = [];
      for (var rj = 0; rj < inRow.length; rj++) {
        if (fsBase === null) fsBase = inRow[rj].fs;
        if (Math.abs(inRow[rj].fs - fsBase) <= 1) sameSize.push(inRow[rj]);
      }
      if (sameSize.length < 2) continue;
      stats.rows++;
      var cys = [];
      for (var cj = 0; cj < sameSize.length; cj++) cys.push(sameSize[cj].box.cy);
      var maxCy = Math.max.apply(null, cys), minCy = Math.min.apply(null, cys);
      var spread = round1(maxCy - minCy);
      var vaSet = {};
      for (var vj = 0; vj < sameSize.length; vj++) vaSet[String(sameSize[vj].va)] = 1;
      var vaKeys = Object.keys(vaSet);
      if (spread > ROW_LIMIT && violations.length < MAX_ITEMS) {
        violations.push({
          rule: 'tab-row-baseline',
          page: page.name,
          screen: fr.name,
          pill: pill.node.id,
          spread: spread,
          items: sameSize.map(function (it) { return it.node.id + '|' + it.text.slice(0, 6) + '|cy' + round1(it.box.cy) + '|' + it.va; }).slice(0, 8),
          hint: '整行统一 textAlignVertical（建议全部 CENTER 且文本框高度=行高），或统一 Y 坐标使字形中心一致'
        });
      } else if (vaKeys.length > 1 && warnings.length < MAX_ITEMS) {
        warnings.push({
          rule: 'tab-row-va-mixed',
          page: page.name,
          screen: fr.name,
          pill: pill.node.id,
          spread: spread,
          items: sameSize.map(function (it) { return it.node.id + '|' + it.text.slice(0, 6) + '|' + it.va; }).slice(0, 8),
          hint: '同一行标签混用了 ' + vaKeys.join('/') + '，字体或尺寸变化后会再次漂移，应统一'
        });
      }
    }

    // ---------- 规则 3：页头标题居中（仅限页头区域：返回按钮在画板左上角） ----------
    var screenPos = absPos(fr);
    var headerBtn = null;
    for (var bj = 0; bj < buttons.length; bj++) {
      var cand = buttons[bj];
      if (cand.box.y < screenPos.y - 6) continue;
      if (cand.box.y > screenPos.y + 60) continue;
      if (cand.box.x < screenPos.x + 2) continue;
      if (cand.box.x > screenPos.x + 90) continue;
      headerBtn = cand;
      break;
    }
    var btn = headerBtn;
    if (btn) {
      var title = null;
      for (var hj = 0; hj < texts.length; hj++) {
        var candTitle = texts[hj];
        if (candTitle.fs < 15) continue;
        if (candTitle.box.y < screenPos.y - 6) continue;
        if (candTitle.box.y > screenPos.y + 60) continue;
        if (candTitle.box.x <= btn.box.x + btn.box.w - 6) continue;
        if (title === null || candTitle.box.y < title.box.y) title = candTitle;
      }
      if (title) {
        var sub = null;
        for (var sj = 0; sj < texts.length; sj++) {
          var c2 = texts[sj];
          if (c2 === title) continue;
          if (c2.fs >= title.fs - 3) continue;
          if (Math.abs(c2.box.x - title.box.x) > 36) continue;
          if (c2.box.y <= title.box.y || c2.box.y > title.box.y + 44) continue;
          sub = c2; break;
        }
        stats.headers++;
        var target = btn.box.cy;
        var delta;
        var mode;
        if (sub) {
          var blockCenter = (title.box.y + sub.box.bottom) / 2;
          delta = round1(blockCenter - target);
          mode = 'block';
        } else {
          delta = round1(title.box.cy - target);
          mode = 'single';
        }
        if (Math.abs(delta) > HEADER_LIMIT && violations.length < MAX_ITEMS) {
          violations.push({
            rule: 'header-center',
            page: page.name,
            screen: fr.name,
            title: title.node.id + '|' + title.text.slice(0, 10),
            sub: sub ? sub.node.id + '|' + sub.text.slice(0, 10) : null,
            mode: mode,
            delta: delta,
            hint: mode === 'single' ? '单行标题字形中心应与返回按钮中心对齐（va=CENTER + y=按钮中心-半行高）' : '标题+副标题整块中心应与返回按钮中心对齐'
          });
        }
      }
    }
  }
}

function countBy(list, key) {
  var m = {};
  for (var i = 0; i < list.length; i++) m[list[i][key]] = (m[list[i][key]] || 0) + 1;
  return m;
}

return {
  summary: {
    pages: stats.pages,
    screens: stats.screens,
    checkedRows: stats.rows,
    checkedHeaders: stats.headers,
    violations: violations.length,
    warnings: warnings.length,
    byRule: { violations: countBy(violations, 'rule'), warnings: countBy(warnings, 'rule') },
    thresholds: { rowSpread: ROW_LIMIT, headerCenter: HEADER_LIMIT },
    pass: violations.length === 0
  },
  violations: violations,
  warnings: warnings.slice(0, 40)
};