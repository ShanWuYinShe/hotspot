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
//   card-text-overflow    圆角卡片内文案渲染宽度不得超出卡片右内边距，也不得压住右侧行内图标
//   card-inner-band       圆角卡片内不应出现方形（圆角 0）行带矩形（提示）
//
// 返回值：{ summary, violations, warnings }（violations 为空即通过）
// ============================================================================
var ROW_LIMIT = 2.0;      // 同行字形中心允许偏差（px）
var HEADER_LIMIT = 2.0;   // 页头中心允许偏差（px）
var MAX_ITEMS = 160;      // 每类最多输出条数，避免超出 MCP 返回值上限
var NAV_LIMIT = 2.0;      // 侧边栏导航项 x 允许偏差（px）
var OVERLAY_COVER = 0.8;  // 叠字卡片内图片覆盖率阈值
var CARD_PAD = 16;        // 圆角卡片右内边距（文案可用宽度基准）
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

function colorHex(c) {
  if (!c) return '#000000';
  function h(v) { var s = Math.round(Math.max(0, Math.min(1, v)) * 255).toString(16); return s.length < 2 ? '0' + s : s; }
  return '#' + h(c.r) + h(c.g) + h(c.b);
}

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
var stats = { pages: 0, screens: 0, rows: 0, headers: 0, chips: 0, overlays: 0, navGroups: 0, cardTexts: 0 };

var allPages = [];
for (var pi = 0; pi < figma.root.children.length; pi++) {
  if (figma.root.children[pi].type === 'PAGE') allPages.push(figma.root.children[pi]);
}

// 设计规范页（01）是规范区块不是应用截图页：其顶层 FRAME 是「概览/主题/语义色…」等区块，
// 不参与屏幕级巡检（胶囊行/页头/卡片/叠字/导航规则都按截图页语义判定，规范页会误报）。
for (var pd = allPages.length - 1; pd >= 0; pd--) {
  if (String(allPages[pd].name).indexOf('01') >= 0) allPages.splice(pd, 1);
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

    // ---------- 规则 9：禁止硬编码纯色（颜色必须绑定变量，才能随 6 个颜色模式/主题自适应） ----------
    var colorPaints = 0;
    var hardcoded = [];
    var colorNodes = fr.findAll ? fr.findAll(function (n) { return (n.fills && n.fills.length) || (n.strokes && n.strokes.length); }) : [fr];
    for (var cni = 0; cni < colorNodes.length; cni++) {
      var cnode = colorNodes[cni];
      var groups = [{ list: cnode.fills || [], role: 'fill' }, { list: cnode.strokes || [], role: 'stroke' }];
      for (var cgi = 0; cgi < groups.length; cgi++) {
        for (var cpi = 0; cpi < groups[cgi].list.length; cpi++) {
          var cpaint = groups[cgi].list[cpi];
          if (cpaint.type !== 'SOLID') continue;
          colorPaints++;
          var cBound = false;
          try { cBound = !!(cpaint.boundVariables && cpaint.boundVariables.color && cpaint.boundVariables.color.id); } catch (e) { cBound = false; }
          if (cBound) continue;
          if (hardcoded.length < 8) hardcoded.push(cnode.id + '|' + groups[cgi].role + '|' + colorHex(cpaint.color) + '|' + String(cnode.name || cnode.type).slice(0, 18));
        }
      }
    }
    stats.colorPaints = (stats.colorPaints || 0) + colorPaints;
    if (hardcoded.length && violations.length < MAX_ITEMS) {
      violations.push({
        rule: 'hardcoded-color',
        page: page.name,
        screen: fr.name,
        count: hardcoded.length,
        samples: hardcoded,
        hint: '颜色必须绑定 HotSpot Tokens 变量（paint.boundVariables.color），写死 hex 在其它颜色模式/主题下不会自适应；新增色请先加变量再绑定'
      });
    }

    // ---------- 规则 10：CJK 文案字体必须是 Noto Sans SC ----------
    var checkTexts = fr.findAll ? fr.findAll(function (n) { return n.type === 'TEXT' && n.visible !== false; }) : [];
    stats.checkTexts = (stats.checkTexts || 0) + checkTexts.length;
    var cjkBad = [];
    for (var cti = 0; cti < checkTexts.length; cti++) {
      if (!/[\u4e00-\u9fff]/.test(String(checkTexts[cti].characters || ''))) continue;
      var cfam = (checkTexts[cti].fontName || {}).family;
      if (cfam !== 'Noto Sans SC' && cfam !== 'PingFang SC') cjkBad.push(checkTexts[cti].id + '|' + cfam + '|' + String(checkTexts[cti].characters).slice(0, 12));
    }
    if (cjkBad.length && violations.length < MAX_ITEMS) {
      violations.push({ rule: 'cjk-font-missing', page: page.name, screen: fr.name, count: cjkBad.length, samples: cjkBad.slice(0, 6), hint: '中文文案统一用 Noto Sans SC：其它字族在插件沙盒缺中文字形，会出现缺字或字距错乱' });
    }

    // ---------- 规则 11：文字溢出自身文本框（可能被裁切） ----------
    var overflowSamples = [];
    for (var oti = 0; oti < checkTexts.length; oti++) {
      var ot = checkTexts[oti];
      if (ot.textAutoResize !== 'NONE' && ot.textAutoResize !== 'HEIGHT') continue;
      var orb = renderBox(ot);
      var obb = null;
      try { obb = ot.absoluteBoundingBox; } catch (e) { obb = null; }
      if (!orb || !obb) continue;
      var oox = (orb.x + orb.w) - (obb.x + obb.width);
      var ooy = (orb.y + orb.h) - (obb.y + obb.height);
      if (oox > 2 || ooy > 2) overflowSamples.push(ot.id + '|overX=' + Math.round(oox) + ',overY=' + Math.round(ooy) + '|' + String(ot.characters).slice(0, 14));
    }
    if (overflowSamples.length && violations.length < MAX_ITEMS) {
      violations.push({ rule: 'text-overflow-box', page: page.name, screen: fr.name, count: overflowSamples.length, samples: overflowSamples.slice(0, 6), hint: '文案渲染尺寸超出自身文本框（NONE/HEIGHT 自适应），会压到相邻元素；请截断或调整文本框宽度' });
    }

    // ---------- 规则 12：同父级「标题 + 副标题」纵向重叠 ----------
    // 底部固定浮层（全宽条）会盖住滚动内容，两者叠压属正常层级，不计入
    var bottomBarsR = [];
    var frH = fr.height, frW = fr.width;
    var frKids = fr.children || [];
    for (var bb0 = 0; bb0 < frKids.length; bb0++) {
      var bk0 = frKids[bb0];
      if (bk0.visible === false) continue;
      if (bk0.width < frW * 0.9) continue;
      if (bk0.height < 40) continue;
      if (Math.abs((bk0.y + bk0.height) - frH) > 2) continue;
      bottomBarsR.push({ x: bk0.x, y: bk0.y, w: bk0.width, h: bk0.height });
    }
    function underBottomBar(n) {
      var bb = null;
      try { bb = n.absoluteBoundingBox; } catch (e) { return false; }
      if (!bb) return false;
      var cx = bb.x + bb.width / 2, cy = bb.y + bb.height / 2;
      for (var bi2 = 0; bi2 < bottomBarsR.length; bi2++) {
        var r = bottomBarsR[bi2];
        var ax = absPos(fr).x, ay = absPos(fr).y;
        if (cx >= ax + r.x && cx <= ax + r.x + r.w && cy >= ay + r.y && cy <= ay + r.y + r.h) return true;
      }
      return false;
    }
    var byParent = {};
    for (var pti = 0; pti < checkTexts.length; pti++) {
      var pt = checkTexts[pti];
      if ((pt.fontSize || 0) < 8) continue;
      var ppid = pt.parent ? pt.parent.id : 'none';
      if (!byParent[ppid]) byParent[ppid] = [];
      byParent[ppid].push(pt);
    }
    var overlapPairs = [];
    for (var ppid2 in byParent) {
      var arr2 = byParent[ppid2];
      for (var pi3 = 0; pi3 < arr2.length; pi3++) {
        for (var pj3 = pi3 + 1; pj3 < arr2.length; pj3++) {
          var A2 = arr2[pi3], B2 = arr2[pj3];
          var first2 = A2.y <= B2.y ? A2 : B2;
          var second2 = A2.y <= B2.y ? B2 : A2;
          var dy2 = second2.y - first2.y;
          if (dy2 < 0 || dy2 >= (first2.height || 0)) continue;
          if (Math.abs(first2.x - second2.x) > 2) continue;
          var oy2 = Math.min(first2.y + first2.height, second2.y + second2.height) - Math.max(first2.y, second2.y);
          if (oy2 <= 2) continue;
          if (underBottomBar(first2) && underBottomBar(second2)) continue;
          overlapPairs.push(first2.id + ' y=' + Math.round(first2.y) + 'h=' + Math.round(first2.height) + ' fs=' + first2.fontSize + ' || ' + second2.id + ' y=' + Math.round(second2.y) + 'h=' + Math.round(second2.height) + ' fs=' + second2.fontSize + ' | overlapY=' + Math.round(oy2));
        }
      }
    }
    if (overlapPairs.length && violations.length < MAX_ITEMS) {
      violations.push({ rule: 'text-row-overlap', page: page.name, screen: fr.name, count: overlapPairs.length, samples: overlapPairs.slice(0, 6), hint: '同一父级下标题/副标题字盒重叠（行距小于字高）：把副标题下移或缩小行高，避免文字相撞' });
    }

    // ---------- 规则 13：底部固定容器被屏幕裁切（滚动内容贴底裁切属正常滚动状态，不检查） ----------
    var kids0 = fr.children || [];
    var cutSamples = [];
    var frAbs0 = absPos(fr);
    for (var cki = 0; cki < kids0.length; cki++) {
      var ck0 = kids0[cki];
      if (ck0.visible === false || ck0.type !== 'FRAME') continue;
      if (ck0.width < fr.width * 0.5) continue;
      if (ck0.height > 200) continue; // 高容器多为滚动内容列，贴底裁切属正常滚动状态
      var ckb = null;
      try { ckb = ck0.absoluteBoundingBox; } catch (e) { ckb = null; }
      if (!ckb) continue;
      var overB = (ckb.y + ckb.height) - (frAbs0.y + fr.height);
      if (overB > 1) cutSamples.push(ck0.id + '|' + String(ck0.name).slice(0, 16) + '|bottom 超出屏幕 ' + Math.round(overB) + 'px');
    }
    if (cutSamples.length && violations.length < MAX_ITEMS) {
      violations.push({ rule: 'screen-cut', page: page.name, screen: fr.name, count: cutSamples.length, samples: cutSamples.slice(0, 6), hint: '底部固定容器（导航/输入条/操作条/面板）超出屏幕底边：容器应完整落在屏幕内，底边贴齐屏幕底或按安全区留白' });
    }

    // ---------- 规则 14：底部安全区（小白条 / home indicator） ----------
    var pgKey = page.name.slice(0, 2);
    var safeInset = pgKey === '02' ? 34 : (pgKey === '03' ? 20 : 0);
    if (safeInset > 0) {
      var safeBad = [];
      // ① 底部导航条必须在安全区之上（留出 home indicator）
      var navLabelSet = { '首页': 1, '附近': 1, '消息': 1, '我的': 1 };
      var navAnchors = {};
      for (var snt = 0; snt < checkTexts.length; snt++) {
        var sntx = checkTexts[snt];
        if (!navLabelSet[String(sntx.characters).trim()]) continue;
        var sntb = null;
        try { sntb = sntx.absoluteBoundingBox; } catch (e) { sntb = null; }
        if (!sntb || sntb.y < absPos(fr).y + frH - 140) continue;
        var sanc = sntx, sdep = 0;
        while (sanc && sanc.parent && sanc.parent.id !== fr.id && sdep < 8) { sanc = sanc.parent; sdep++; }
        if (sanc && sanc.parent && sanc.parent.id === fr.id) navAnchors[sanc.id] = sanc;
      }
      var sancIds = Object.keys(navAnchors);
      if (sancIds.length >= 3) {
        var snx1 = 1e9, sny1 = 1e9, snx2 = -1e9, sny2 = -1e9;
        for (var snk = 0; snk < sancIds.length; snk++) {
          var snb = null;
          try { snb = navAnchors[sancIds[snk]].absoluteBoundingBox; } catch (e) { snb = null; }
          if (!snb) continue;
          snx1 = Math.min(snx1, snb.x); sny1 = Math.min(sny1, snb.y); snx2 = Math.max(snx2, snb.x + snb.width); sny2 = Math.max(sny2, snb.y + snb.height);
        }
        var navBar = null;
        for (var nb = 0; nb < kids0.length; nb++) {
          var nbk = kids0[nb];
          if (nbk.type !== 'RECTANGLE' && nbk.type !== 'FRAME') continue;
          if (nbk.height < 55 || nbk.height > 100) continue;
          if (nbk.width < frW * 0.45 || nbk.width >= frW) continue;
          var nbc = (snx1 + snx2) / 2, nby = (sny1 + sny2) / 2;
          var nbr = null;
          try { nbr = nbk.absoluteBoundingBox; } catch (e) { nbr = null; }
          if (!nbr) continue;
          if (nbc < nbr.x - 10 || nbc > nbr.x + nbr.width + 10) continue;
          if (nby < nbr.y - 30 || nby > nbr.y + nbr.height + 10) continue;
          navBar = nbr; break;
        }
        if (navBar) {
          var navBottomGap = (absPos(fr).y + frH) - (navBar.y + navBar.height);
          if (navBottomGap < safeInset - 1) safeBad.push('nav-bar|bottomGap=' + Math.round(navBottomGap) + '<inset=' + safeInset);
        }
      }
      // ② 底部输入条里的占位文案不能进入安全区
      for (var ibt = 0; ibt < checkTexts.length; ibt++) {
        var ibtx = checkTexts[ibt];
        if (!/写下你的评论|发消息|回复 @/.test(String(ibtx.characters))) continue;
        var ibb = null;
        try { ibb = ibtx.absoluteBoundingBox; } catch (e) { ibb = null; }
        if (!ibb) continue;
        if (ibb.y + ibb.height > absPos(fr).y + frH - safeInset + 2) safeBad.push('input-below-safe|' + ibtx.id + '|textBottom=' + Math.round(ibb.y + ibb.height - absPos(fr).y));
      }
      if (safeBad.length && violations.length < MAX_ITEMS) {
        violations.push({ rule: 'safe-area-bottom', page: page.name, screen: fr.name, count: safeBad.length, samples: safeBad.slice(0, 6), hint: '底部导航/输入条未留出 home indicator 安全区（移动端 34px、平板 20px）：导航整体上移、输入条加高并把内容上移' });
      }
    }

    // ---------- 规则 15：顶部安全区（状态栏 / 灵动岛） ----------
    var topInset = pgKey === '02' ? 47 : (pgKey === '03' ? 24 : 0);
    if (topInset > 0) {
      var topBad = [];
      var frAbsT = absPos(fr);
      for (var tki = 0; tki < kids0.length; tki++) {
        var tk0 = kids0[tki];
        if (tk0.visible === false) continue;
        var tw0 = tk0.width || 0, th0 = tk0.height || 0;
        if (tw0 >= fr.width * 0.99) continue; // 全宽背景/头部底条允许通栏延伸到 y=0
        if (tk0.y <= 0) continue;             // 从屏外延入的全出血媒体（地图瓦片/照片）
        if (tk0.y >= topInset - 1) continue;
        if (th0 > 120 || tw0 > fr.width * 0.9) continue;
        topBad.push(tk0.id + '|' + String(tk0.name).slice(0, 14) + '|y=' + Math.round(tk0.y) + '<inset=' + topInset);
      }
      if (topBad.length && violations.length < MAX_ITEMS) {
        violations.push({ rule: 'safe-area-top', page: page.name, screen: fr.name, count: topBad.length, samples: topBad.slice(0, 6), hint: '顶部元素进入状态栏/灵动岛安全区（移动端 47px、平板 24px）：标准页扩高背景并把内容整体下移，沉浸页（地图/照片）仅下移顶部悬浮控件' });
      }
    }

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

    // ---------- 规则 7：卡片内文案不得溢出卡片 / 不得压住右侧行内图标 ----------
    if (page.name.indexOf('01') < 0) {
      var cardBoxes = fr.findAll(function (n) {
        if (n.type !== 'RECTANGLE' || n.visible === false) return false;
        if (typeof n.cornerRadius !== 'number' || n.cornerRadius < 16) return false;
        if (n.width < 200 || n.width > 420) return false;
        if (n.height < 100 || n.height > 700) return false;
        return true;
      });
      var rowIcons = fr.findAll(function (n) {
        return n.type === 'FRAME' && n.visible !== false && n.width >= 10 && n.width <= 16 && Math.abs(n.width - n.height) <= 2;
      });
      for (var cvi = 0; cvi < cardBoxes.length; cvi++) {
        var vcard = cardBoxes[cvi];
        var vcb = renderBox(vcard);
        if (!vcb) continue;
        var vTexts = fr.findAll(function (n) { return n.type === 'TEXT' && n.visible !== false; });
        for (var vti = 0; vti < vTexts.length; vti++) {
          var vt = vTexts[vti];
          if (vt.textAlignHorizontal && vt.textAlignHorizontal !== 'LEFT') continue;
          var vtb = renderBox(vt);
          if (!vtb) continue;
          if (vtb.x < vcb.x + 4 || vtb.x > vcb.x + vcb.w - 2) continue;
          if (vtb.y < vcb.y - 1 || vtb.y > vcb.y + vcb.h) continue;
          stats.cardTexts++;
          // 默认允许到达卡片右缘；若右侧同行有 10-16px 图标，再让出图标宽度 + 4px 间距
          var limit = vcb.x + vcb.w - 2;
          for (var vri = 0; vri < rowIcons.length; vri++) {
            if (rowIcons[vri].parent && vt.parent && rowIcons[vri].parent.id !== vt.parent.id) continue;
            var ric = renderBox(rowIcons[vri]);
            if (!ric) continue;
            if (ric.x <= vtb.x) continue;
            if (Math.abs(ric.cy - vtb.cy) > 8) continue;
            var iconLimit = ric.x - 4;
            if (iconLimit < limit) limit = iconLimit;
          }
          var vRight = vtb.x + vtb.w;
          if (vRight > limit + 2 && violations.length < MAX_ITEMS) {
            violations.push({
              rule: 'card-text-overflow',
              page: page.name,
              screen: fr.name,
              card: vcard.id,
              text: vt.id + '|' + String(vt.characters).slice(0, 14),
              overflow: round1(vRight - limit),
              hint: '卡片内文案渲染超出可用宽度（卡片右内边距 ' + CARD_PAD + 'px，或压住右侧行内图标）：应缩短文案并加省略号，或开启换行/截断'
            });
          }
        }
        // 卡片内方形行带（r<=2 的实心矩形）——生成端常见噪音
        var vKids = fr.children;
        for (var vki = 0; vki < vKids.length; vki++) {
          var vk = vKids[vki];
          if (vk.visible === false || vk.type !== 'RECTANGLE' || vk.id === vcard.id) continue;
          if (typeof vk.cornerRadius === 'number' && vk.cornerRadius > 2) continue;
          var vf = (vk.fills || [])[0];
          if (!vf || vf.type !== 'SOLID') continue;
          var vkb = renderBox(vk);
          if (!vkb) continue;
          if (vkb.w < vcard.width * 0.55 || vkb.h < 24) continue;
          if (vkb.x < vcb.x + 1 || vkb.x + vkb.w > vcb.x + vcb.w - 1) continue;
          if (vkb.y < vcb.y + 1 || vkb.y + vkb.h > vcb.y + vcb.h - 1) continue;
          if (warnings.length < MAX_ITEMS) {
            warnings.push({
              rule: 'card-inner-band',
              page: page.name,
              screen: fr.name,
              card: vcard.id,
              rect: vk.id + '|' + round1(vkb.w) + 'x' + round1(vkb.h),
              insetL: round1(vkb.x - vcb.x),
              insetR: round1(vcb.x + vcb.w - (vkb.x + vkb.w)),
              hint: '圆角卡片内出现方形行带矩形（圆角 0、左右内缩不对称）：要么移除改为扁平列表，要么统一圆角 12 + 左右内缩相同'
            });
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
    checkedCardTexts: stats.cardTexts,
    checkedColorPaints: stats.colorPaints || 0,
    checkedTexts: stats.checkTexts || 0,
    violations: violations.length,
    warnings: warnings.length,
    byRule: { violations: countBy(violations, 'rule'), warnings: countBy(warnings, 'rule') },
    thresholds: { rowSpread: ROW_LIMIT, headerCenter: HEADER_LIMIT, cardChipInset: 6, navIndent: NAV_LIMIT, overlayCover: OVERLAY_COVER, cardPad: CARD_PAD },
    pass: violations.length === 0
  },
  violations: violations,
  warnings: warnings.slice(0, 40)
};