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
//   card-chip-edge / card-chip-inset-asym / card-text-edge  卡片右上角时间胶囊/文字不得贴边
//   overlay-missing-glass 图片叠字卡片（图片 >80% 铺底）有白字或黑渐变却没有玻璃条
//   overlay-glass-theme   叠字卡片的玻璃条颜色必须与卡片所在主题匹配（浅色主题白条 / 深色主题深条）
//   overlay-text-theme    叠字卡片正文/计数颜色必须与主题匹配（浅色主题深字 / 深色主题浅字）
//   overlay-chip-text     叠字卡片黑色胶囊内文字必须反白
//   nav-item-indent       侧边栏导航项 x 必须同一基准（激活项不得缩进，<=2px）
//
// 返回值：{ summary, violations, warnings }（violations 为空即通过）
// ============================================================================
var ROW_LIMIT = 2.0;      // 同行字形中心允许偏差（px）
var HEADER_LIMIT = 2.0;   // 页头中心允许偏差（px）
var MAX_ITEMS = 160;      // 每类最多输出条数，避免超出 MCP 返回值上限
var NAV_LIMIT = 2.0;      // 侧边栏导航项 x 允许偏差（px）
var OVERLAY_COVER = 0.8;  // 叠字卡片内图片覆盖率阈值
var NAV_LABELS = ['首页', '附近的人', '附近的群组', '附近瞬间', '消息', '我的瞬间', '我的群组', '关系', '管理后台'];

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

function relLum(c) { return c.r * 0.299 + c.g * 0.587 + c.b * 0.114; }

function isWhiteFill(n) {
  var f = (n.fills || [])[0];
  if (!f || f.type !== 'SOLID' || !f.color) return false;
  return f.color.r > 0.93 && f.color.g > 0.93 && f.color.b > 0.93;
}

function isGlassBar(n) {
  if (n.type !== 'RECTANGLE') return false;
  var f = (n.fills || [])[0];
  if (!f || f.type !== 'SOLID' || !f.color) return false;
  var e = (n.effects || [])[0];
  return !!(e && e.type === 'BACKGROUND_BLUR');
}

function isBlackPillRect(n) {
  if (n.type !== 'RECTANGLE') return false;
  var f = (n.fills || [])[0];
  if (!f || f.type !== 'SOLID' || !f.color) return false;
  if (f.opacity === undefined || f.opacity < 0.25 || f.opacity > 0.85) return false;
  return f.color.r < 0.25 && f.color.g < 0.25 && f.color.b < 0.25 && n.width < 220 && n.height < 40;
}

// 节点是否被某个黑胶囊背景覆盖（胶囊与文字在生成稿里是兄弟节点）
function insideAnyPill(node, pills) {
  for (var i = 0; i < pills.length; i++) {
    var p = pills[i];
    if (node.x >= p.x - 6 && node.x + node.width <= p.x + p.width + 6 &&
        node.y >= p.y - 4 && node.y + node.height <= p.y + p.height + 4) return true;
  }
  return false;
}

// 主题判定：取变量的「浅色/深色」模式集合，向上找最近一层已解析的变量模式
var tokenModes = {};
try {
  var varColls = await figma.variables.getLocalVariableCollectionsAsync();
  for (var vci = 0; vci < varColls.length; vci++) {
    var vc2 = varColls[vci];
    var hasLight = false, hasDark = false;
    for (var vmi = 0; vmi < vc2.modes.length; vmi++) {
      if (/light/i.test(vc2.modes[vmi].name)) hasLight = true;
      if (/dark/i.test(vc2.modes[vmi].name)) hasDark = true;
    }
    if (!hasLight || !hasDark) continue;
    tokenModes[vc2.id] = {};
    for (var vmi2 = 0; vmi2 < vc2.modes.length; vmi2++) tokenModes[vc2.id][vc2.modes[vmi2].modeId] = vc2.modes[vmi2].name;
  }
} catch (e) { tokenModes = {}; }

function themeOf(node) {
  var c = node;
  while (c && c.type !== 'PAGE') {
    var modes = c.resolvedVariableModes || {};
    for (var cid in tokenModes) {
      if (!tokenModes.hasOwnProperty(cid)) continue;
      var mid = modes[cid];
      if (mid && tokenModes[cid][mid]) return tokenModes[cid][mid];
    }
    c = c.parent;
  }
  return '';
}

function modeOfX(items) {
  var counts = {};
  for (var i = 0; i < items.length; i++) {
    var k = Math.round(items[i].x);
    counts[k] = (counts[k] || 0) + 1;
  }
  var base = null, best = 0;
  for (var k2 in counts) {
    if (counts.hasOwnProperty(k2) && counts[k2] > best) { best = counts[k2]; base = Number(k2); }
  }
  return base;
}

function isOverlayImageCard(n) {
  if (n.type !== 'FRAME' || n.visible === false) return false;
  if (!(n.width >= 120 && n.width <= 820)) return false;
  if (!(n.height >= 120 && n.height <= 560)) return false;
  var kids = n.children || [];
  for (var i = 0; i < kids.length; i++) {
    var k = kids[i];
    var f = (k.fills || [])[0];
    if (k.type === 'RECTANGLE' && f && f.type === 'IMAGE' && k.width >= n.width * OVERLAY_COVER && k.height >= n.height * OVERLAY_COVER) return true;
  }
  return false;
}

var violations = [];
var warnings = [];
var stats = { pages: 0, screens: 0, rows: 0, headers: 0, chips: 0, overlays: 0, navGroups: 0 };

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

    // ---------- 规则 4：卡片右上角时间胶囊不得贴边/溢出 ----------
    var cardFrames = fr.findAll(function (n) {
      return n.type === 'FRAME' && n.visible !== false && n.width >= 150 && n.width <= 820 && n.height >= 120 && n.height <= 480;
    });
    for (var cfi = 0; cfi < cardFrames.length; cfi++) {
      var card = cardFrames[cfi];
      var ckids = (card.children || []).filter(function (k) { return k.visible !== false; });
      var chipRects = [];
      for (var cki = 0; cki < ckids.length; cki++) {
        var chip = ckids[cki];
        if (chip.type !== 'RECTANGLE') continue;
        if (typeof chip.cornerRadius !== 'number' || chip.cornerRadius < 6) continue;
        if (!(chip.height >= 17 && chip.height <= 30)) continue;
        if (!(chip.width >= 36 && chip.width <= 140)) continue;
        if (!(chip.x > card.width * 0.55)) continue;
        if (!(chip.y >= 4 && chip.y <= 20)) continue;
        var chipText = null;
        for (var cti = 0; cti < ckids.length; cti++) {
          var cand2 = ckids[cti];
          if (cand2.type !== 'TEXT' || !String(cand2.characters || '').trim()) continue;
          if (cand2.x >= chip.x - 8 && cand2.x <= chip.x + chip.width && cand2.y >= chip.y - 6 && cand2.y <= chip.y + chip.height + 6) { chipText = cand2; break; }
        }
        if (!chipText) continue;
        chipRects.push(chip);
        stats.chips++;
        var rightGap = round1(card.width - (chip.x + chip.width));
        var topGap = round1(chip.y);
        if (rightGap < 6 && violations.length < MAX_ITEMS) {
          violations.push({
            rule: 'card-chip-edge',
            page: page.name,
            screen: fr.name,
            card: card.id,
            chip: chip.id + '|' + String(chipText.characters).slice(0, 10),
            cardW: round1(card.width),
            rightGap: rightGap,
            topGap: topGap,
            hint: '卡片右上角时间胶囊右缘贴边/溢出（应内缩到与上内边距对称，建议 10px）：chip.x = card.width - inset - chip.width，文字同步左移'
          });
        } else if (Math.abs(rightGap - topGap) > 3 && warnings.length < MAX_ITEMS) {
          warnings.push({
            rule: 'card-chip-inset-asym',
            page: page.name,
            screen: fr.name,
            card: card.id,
            chip: chip.id,
            rightGap: rightGap,
            topGap: topGap,
            hint: '时间胶囊内边距不对称（右 ' + rightGap + ' vs 上 ' + topGap + '），建议统一为相同 inset'
          });
        }
      }
      // 卡片右上角纯文字同样不得贴边
      for (var pti = 0; pti < ckids.length; pti++) {
        var pt = ckids[pti];
        if (pt.type !== 'TEXT' || !String(pt.characters || '').trim()) continue;
        if (pt.y > 40) continue;
        var ptBox = renderBox(pt);
        if (!ptBox) continue;
        var cardPos = absPos(card);
        var textGap = round1((cardPos.x + card.width) - (ptBox.x + ptBox.w));
        if (textGap >= 6) continue;
        var insideChip = false;
        for (var cii = 0; cii < chipRects.length; cii++) {
          var cr = chipRects[cii];
          if (pt.x >= cr.x - 8 && pt.x <= cr.x + cr.width + 8 && pt.y >= cr.y - 6 && pt.y <= cr.y + cr.height + 6) { insideChip = true; break; }
        }
        if (insideChip) continue;
        if (violations.length < MAX_ITEMS) {
          violations.push({
            rule: 'card-text-edge',
            page: page.name,
            screen: fr.name,
            card: card.id,
            text: pt.id + '|' + String(pt.characters).slice(0, 10),
            rightGap: textGap,
            hint: '卡片右上角文字贴边（右内边距 < 6px），应内缩到与卡片内边距一致'
          });
        }
      }
    }
    // ---------- 规则 5：图片叠字卡片的玻璃条与主题一致性（设计规范页除外） ----------
    if (page.name.indexOf('01') < 0) {
      var overlayCards = fr.findAll(isOverlayImageCard);
      for (var ofi = 0; ofi < overlayCards.length; ofi++) {
        var ocard = overlayCards[ofi];
        var okids = (ocard.children || []).filter(function (k) { return k.visible !== false; });
        var bars = [];
        var grads = 0;
        var pills2 = [];
        for (var oki = 0; oki < okids.length; oki++) {
          var ok = okids[oki];
          if (isGlassBar(ok)) bars.push(ok);
          var okf = (ok.fills || [])[0];
          if (ok.type === 'RECTANGLE' && okf && String(okf.type).indexOf('GRADIENT') === 0) grads++;
          if (isBlackPillRect(ok)) pills2.push(ok);
        }
        stats.overlays++;
        var otheme = themeOf(ocard);
        var oDark = /dark/i.test(otheme);
        var wTexts = [];
        var otexts = ocard.findAll(function (n) { return n.type === 'TEXT' && n.visible !== false; });
        for (var oti = 0; oti < otexts.length; oti++) {
          if (isWhiteFill(otexts[oti]) && !insideAnyPill(otexts[oti], pills2)) wTexts.push(otexts[oti]);
        }
        if (!bars.length) {
          // 仅有装饰性渐变（无叠字）不算问题，只拦「白字压图却没有承托」的情况
          if (wTexts.length > 0 && violations.length < MAX_ITEMS) {
            violations.push({
              rule: 'overlay-missing-glass',
              page: page.name,
              screen: fr.name,
              card: ocard.id + '|' + round1(ocard.width) + 'x' + round1(ocard.height),
              theme: otheme,
              whiteTexts: wTexts.length,
              gradients: grads,
              hint: '图片叠字卡片缺玻璃条（图片 >80% 铺底且有白字/黑渐变）：浅色主题=白 62% + 深字，深色主题=深色 66% + 浅字，模糊 18'
            });
          }
          continue;
        }
        for (var bari = 0; bari < bars.length; bari++) {
          var bf = bars[bari].fills[0];
          var bLum = relLum(bf.color);
          if ((bLum < 0.5) !== oDark && violations.length < MAX_ITEMS) {
            violations.push({
              rule: 'overlay-glass-theme',
              page: page.name,
              screen: fr.name,
              card: ocard.id,
              bar: bars[bari].id,
              theme: otheme,
              luminance: round1(bLum),
              hint: oDark ? '深色主题下玻璃条应为深色（建议 #0B111C 66% + 模糊 18）' : '浅色主题下玻璃条应为浅色（建议白 62% + 模糊 18）'
            });
          }
        }
        for (var otj = 0; otj < otexts.length; otj++) {
          var ot = otexts[otj];
          var otf = (ot.fills || [])[0];
          if (!otf || otf.type !== 'SOLID' || !otf.color) continue;
          var otLum = relLum(otf.color);
          if (insideAnyPill(ot, pills2)) {
            if (otLum < 0.7 && violations.length < MAX_ITEMS) {
              violations.push({
                rule: 'overlay-chip-text',
                page: page.name,
                screen: fr.name,
                card: ocard.id,
                text: ot.id + '|' + String(ot.characters).slice(0, 8),
                luminance: round1(otLum),
                hint: '黑色胶囊内文字必须反白（content-inverse），当前偏暗'
              });
            }
            continue;
          }
          if ((otLum > 0.6) !== oDark && violations.length < MAX_ITEMS) {
            violations.push({
              rule: 'overlay-text-theme',
              page: page.name,
              screen: fr.name,
              card: ocard.id,
              text: ot.id + '|' + String(ot.characters).slice(0, 8),
              theme: otheme,
              luminance: round1(otLum),
              hint: oDark ? '深色主题叠字应为浅色（正文 #E8F1E9 / 次级 #A8BDAE）' : '浅色主题叠字应为深色（正文 #16233A / 次级 #52647F）'
            });
          }
        }
      }

      // ---------- 规则 6：桌面侧边栏导航项对齐（激活项不得缩进） ----------
      if (fr.width >= 1100) {
        var sPos = absPos(fr);
        var navTexts = [];
        var navIcons = [];
        var navAll = fr.findAll(function (n) { return n.visible !== false; });
        for (var nti = 0; nti < navAll.length; nti++) {
          var nt = navAll[nti];
          if (nt.type === 'TEXT') {
            if (NAV_LABELS.indexOf(String(nt.characters || '').trim()) < 0) continue;
            var ntb = renderBox(nt);
            if (!ntb) continue;
            var ntx = ntb.x - sPos.x, nty = ntb.y - sPos.y;
            if (ntx > 240 || nty > 700) continue;
            navTexts.push({ node: nt, x: ntx, y: nty, text: String(nt.characters) });
          } else if (nt.type === 'FRAME' && nt.width >= 15 && nt.width <= 21 && Math.abs(nt.width - nt.height) <= 3) {
            var nib = renderBox(nt);
            if (!nib) continue;
            var nix = nib.x - sPos.x, niy = nib.y - sPos.y;
            if (nix > 200 || niy > 700) continue;
            navIcons.push({ node: nt, x: nix });
          }
        }
        if (navTexts.length >= 4) {
          stats.navGroups++;
          var navBase = modeOfX(navTexts);
          for (var ntj = 0; ntj < navTexts.length; ntj++) {
            var ndx = round1(navTexts[ntj].x - navBase);
            if (Math.abs(ndx) > NAV_LIMIT && violations.length < MAX_ITEMS) {
              violations.push({
                rule: 'nav-item-indent',
                page: page.name,
                screen: fr.name,
                item: navTexts[ntj].node.id + '|' + navTexts[ntj].text,
                offset: ndx,
                hint: '侧边栏导航项缩进：激活项文字/图标被右移，应与普通项同一 x 基准（文字 x=基准，图标 x=基准-35）'
              });
            }
          }
        }
        if (navIcons.length >= 5) {
          var iconBase = modeOfX(navIcons);
          for (var nij = 0; nij < navIcons.length; nij++) {
            var idx = round1(navIcons[nij].x - iconBase);
            if (Math.abs(idx) > NAV_LIMIT && violations.length < MAX_ITEMS) {
              violations.push({
                rule: 'nav-item-indent',
                page: page.name,
                screen: fr.name,
                item: navIcons[nij].node.id + '|icon',
                offset: idx,
                hint: '侧边栏导航图标缩进：同一列图标 x 必须一致（与文字共用同一基准）'
              });
            }
          }
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
    checkedChips: stats.chips,
    checkedOverlayCards: stats.overlays,
    checkedNavGroups: stats.navGroups,
    violations: violations.length,
    warnings: warnings.length,
    byRule: { violations: countBy(violations, 'rule'), warnings: countBy(warnings, 'rule') },
    thresholds: { rowSpread: ROW_LIMIT, headerCenter: HEADER_LIMIT, cardChipInset: 6, navIndent: NAV_LIMIT, overlayCover: OVERLAY_COVER },
    pass: violations.length === 0
  },
  violations: violations,
  warnings: warnings.slice(0, 40)
};