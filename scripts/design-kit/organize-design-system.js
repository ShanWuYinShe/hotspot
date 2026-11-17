// ============================================================================
// HotSpot Design Kit — 设计规范页（01）整理脚本（可重复执行，非破坏式）
//
// 用途：插件生成/重生成「01 · 设计规范」后，把散落的 500+ 顶层节点整理成：
//   00 概览 / 01 主题 / 02 语义色 / 03 字号 / 04 圆角 / 05 动效 / 06 组件
//   七个命名区块（FRAME），语义色 token、字号样本、圆角样本、动效时长行、组件类目
//   各自内聚为子 FRAME；同时统一区块间距、修正 CJK 字体与异常窄文本。
//   已整理过的页面再次执行会跳过对应步骤（幂等）。
//
// 运行（MCP 通道，脚本在 Figma 服务端沙盒执行）：
//   FILE_KEY="<fileKey>" SESSION_ID="<sessionId>" \
//   bash .workwork/figma-capture/use-figma.sh \
//     scripts/design-kit/organize-design-system.js "organize ds page" "resource:figma-use"
// ============================================================================
var PAGE_MATCH = "01 · ";
var SECTION_W = 1650;
var SECTION_GAP = 96;
var FONT_REG = { family: "Noto Sans SC", style: "Regular" };
var FONT_MED = { family: "Noto Sans SC", style: "Medium" };
var FONT_BOLD = { family: "Noto Sans SC", style: "Bold" };
var PRIMARY = { r: 22 / 255, g: 35 / 255, b: 58 / 255 };
var SECONDARY = { r: 82 / 255, g: 100 / 255, b: 127 / 255 };

function hasCJK(s) { return /[\u4e00-\u9fff]/.test(String(s || "")); }
function hexOf(color) {
  function h(v) { var s = Math.round(v * 255).toString(16); return s.length < 2 ? "0" + s : s; }
  return "#" + h(color.r) + h(color.g) + h(color.b);
}
function notoStyleFor(styleName) {
  var s = String(styleName || "");
  if (s.indexOf("Bold") >= 0 || s.indexOf("Black") >= 0) return "Bold";
  if (s.indexOf("Medium") >= 0 || s.indexOf("SemiBold") >= 0) return "Medium";
  return "Regular";
}

var page = null;
for (var pi = 0; pi < figma.root.children.length; pi++) {
  var pg = figma.root.children[pi];
  if (pg.type === "PAGE" && String(pg.name).indexOf(PAGE_MATCH) === 0) page = pg;
}
var out = { page: null, sections: [], grouped: {}, skipped: {}, fixed: {} };
if (!page) return { error: "page 01 not found" };
out.page = page.id + "|" + page.name;
await figma.setCurrentPageAsync(page);

// ---------- 0. CJK 字体校正（Inter/PingFang → Noto Sans SC） ----------
var cjkFixed = 0;
var allTexts = page.findAll(function (n) { return n.type === "TEXT"; });
for (var fi = 0; fi < allTexts.length; fi++) {
  var ft = allTexts[fi];
  if (!hasCJK(ft.characters)) continue;
  var fam = (ft.fontName || {}).family;
  if (fam === "Noto Sans SC") continue;
  try {
    var target = notoStyleFor((ft.fontName || {}).style);
    await figma.loadFontAsync({ family: "Noto Sans SC", style: target });
    ft.fontName = { family: "Noto Sans SC", style: target };
    cjkFixed++;
  } catch (e) {}
}
out.fixed.cjkFonts = cjkFixed;

// ---------- 1. 区块分组（已分组则跳过） ----------
var SECTION_NAMES = {
  "主题 × 明暗（6 模式）": "01 主题 / Themes",
  "语义色 Semantic Colors": "02 语义色 / Semantic Colors",
  "字号 Typography": "03 字号 / Typography",
  "圆角 Radius": "04 圆角 / Radius",
  "动效规范与节奏 Motion & Rhythm": "05 动效 / Motion & Rhythm",
  "组件 Components": "06 组件 / Components"
};
function isSectionFrame(n) {
  if (n.type !== "FRAME") return false;
  var name = String(n.name);
  return /^0[0-6] /.test(name);
}
var topKids = page.children.slice();
var hasSections = false;
for (var si = 0; si < topKids.length; si++) { if (isSectionFrame(topKids[si])) { hasSections = true; break; } }
if (!hasSections) {
  // 以 fs>=28 的标题切分区块；主标题（fs>=40）归入概览
  var cuts = [];
  var titles = [];
  for (var ti = 0; ti < topKids.length; ti++) {
    var tk = topKids[ti];
    if (tk.type !== "TEXT" || (tk.fontSize || 0) < 28) continue;
    titles.push(tk);
  }
  titles.sort(function (a, b) { return a.y - b.y; });
  var boundaries = [0];
  for (var bi = 0; bi < titles.length; bi++) {
    if ((titles[bi].fontSize || 0) >= 40) continue;
    boundaries.push(Math.round(titles[bi].y));
  }
  var maxBottom = 0;
  for (var mi = 0; mi < topKids.length; mi++) { maxBottom = Math.max(maxBottom, topKids[mi].y + topKids[mi].height); }
  boundaries.push(Math.ceil(maxBottom));
  var madeSections = [];
  for (var s2 = 0; s2 < boundaries.length - 1; s2++) {
    var top = boundaries[s2], bottom = boundaries[s2 + 1];
    var nodes = [];
    for (var ni = 0; ni < topKids.length; ni++) {
      var node = topKids[ni];
      if (node.y >= top && node.y < bottom) nodes.push(node);
    }
    if (!nodes.length) continue;
    var secName = s2 === 0 ? "00 概览 / Overview" : "07 其它 / Misc";
    if (s2 > 0) {
      var foundTitle = null;
      for (var t2 = 0; t2 < titles.length; t2++) {
        if (Math.round(titles[t2].y) === top) { foundTitle = titles[t2]; break; }
      }
      if (foundTitle && SECTION_NAMES[String(foundTitle.characters).trim()]) secName = SECTION_NAMES[String(foundTitle.characters).trim()];
    }
    var minY = 1e9, maxX = 0;
    for (var ni2 = 0; ni2 < nodes.length; ni2++) {
      if (nodes[ni2].y < minY) minY = nodes[ni2].y;
      if (nodes[ni2].x + nodes[ni2].width > maxX) maxX = nodes[ni2].x + nodes[ni2].width;
    }
    var frame = figma.createFrame();
    frame.name = secName;
    frame.fills = [];
    frame.clipsContent = true;
    page.appendChild(frame);
    frame.x = 0; frame.y = Math.round(minY);
    frame.resize(Math.max(SECTION_W, Math.round(maxX) + 24), Math.round(bottom - minY) + 48);
    for (var ni3 = 0; ni3 < nodes.length; ni3++) {
      var nn = nodes[ni3];
      var ox = nn.x, oy = nn.y;
      frame.appendChild(nn);
      nn.x = ox;
      nn.y = oy - minY;
    }
    madeSections.push(frame);
    out.sections.push(secName);
  }
  out.grouped.sections = madeSections.length;
} else {
  out.skipped.sections = "already grouped";
}

// ---------- 工具：在 section 内按序堆叠子 FRAME ----------
function restack(container, gap, startY, orderPrefix) {
  var frames = container.children.filter(function (k) {
    return k.type === "FRAME" && (!orderPrefix || String(k.name).indexOf(orderPrefix) === 0);
  });
  frames.sort(function (a, b) { return (a.y - b.y) || (a.x - b.x); });
  var cursor = startY;
  for (var i = 0; i < frames.length; i++) {
    frames[i].x = 0;
    frames[i].y = cursor;
    cursor += frames[i].height + gap;
  }
  return cursor - gap;
}

// ---------- 2. 语义色 token 分组（已分组则跳过） ----------
function findSection(prefix) {
  for (var i = 0; i < page.children.length; i++) {
    var n = page.children[i];
    if (n.type === "FRAME" && String(n.name).indexOf(prefix) === 0) return n;
  }
  return null;
}
var secSemantic = findSection("02 语义色");
if (secSemantic) {
  var tokenFrames = secSemantic.children.filter(function (k) { return k.type === "FRAME" && String(k.name).indexOf("token/") === 0; });
  if (tokenFrames.length >= 20) {
    out.skipped.semanticTokens = "already grouped (" + tokenFrames.length + ")";
  } else {
    var swatches = secSemantic.children.filter(function (k) { return k.type === "RECTANGLE" && Math.round(k.width) === 56 && Math.round(k.height) === 44; });
    var texts = secSemantic.children.filter(function (k) { return k.type === "TEXT"; });
    var used = {};
    for (var swi = 0; swi < swatches.length; swi++) {
      var sw = swatches[swi];
      var group = [sw];
      var name = null;
      for (var txi = 0; txi < texts.length; txi++) {
        var tx = texts[txi];
        if (used[tx.id]) continue;
        if (tx.x >= sw.x + 56 && tx.x <= sw.x + 300 && tx.y >= sw.y - 12 && tx.y <= sw.y + 60) {
          group.push(tx);
          used[tx.id] = true;
          if (!name) name = String(tx.characters);
        }
      }
      var maxRight = sw.x + 200, maxBottom = sw.y + 44;
      for (var g2 = 0; g2 < group.length; g2++) {
        maxRight = Math.max(maxRight, group[g2].x + group[g2].width);
        maxBottom = Math.max(maxBottom, group[g2].y + group[g2].height);
      }
      var tf = figma.createFrame();
      tf.name = "token/" + (name || ("c" + swi));
      tf.fills = [];
      tf.clipsContent = false;
      secSemantic.appendChild(tf);
      tf.x = sw.x; tf.y = sw.y;
      tf.resize(Math.round(maxRight - sw.x) + 8, Math.round(maxBottom - sw.y) + 6);
      for (var gi = 0; gi < group.length; gi++) {
        var gn = group[gi];
        var gx = gn.x, gy = gn.y;
        tf.appendChild(gn);
        gn.x = gx - sw.x;
        gn.y = gy - sw.y;
      }
    }
    out.grouped.semanticTokens = swatches.length;
  }
}

// ---------- 3. 字号 / 圆角样本分组 ----------
var secType = findSection("03 字号");
if (secType) {
  var wrapped = 0;
  var loose = secType.children.filter(function (k) { return k.type === "TEXT" && String(k.characters).indexOf("字号") < 0; });
  for (var li = 0; li < loose.length; li++) {
    var lt = loose[li];
    var m = String(lt.characters).match(/(\d+)px/);
    var wf = figma.createFrame();
    wf.name = "type/" + (m ? m[1] + "px" : "sample");
    wf.fills = [];
    wf.clipsContent = false;
    secType.appendChild(wf);
    wf.x = lt.x; wf.y = lt.y;
    wf.resize(Math.round(lt.width) + 4, Math.round(lt.height) + 2);
    var lx = lt.x, ly = lt.y;
    wf.appendChild(lt);
    lt.x = 0; lt.y = 0;
    wrapped++;
  }
  out.grouped.typography = wrapped;
}
var secRadius = findSection("04 圆角");
if (secRadius) {
  var rects = secRadius.children.filter(function (k) { return k.type === "RECTANGLE" && Math.round(k.width) === 90 && Math.round(k.height) === 72; });
  var labels = secRadius.children.filter(function (k) { return k.type === "TEXT"; });
  var rw = 0;
  for (var ri = 0; ri < rects.length; ri++) {
    var rr = rects[ri];
    var rf = figma.createFrame();
    var rlabel = null;
    for (var rj = 0; rj < labels.length; rj++) {
      if (Math.abs(labels[rj].x - rr.x) <= 12 && labels[rj].y >= rr.y + 60 && labels[rj].y <= rr.y + 100) { rlabel = labels[rj]; break; }
    }
    rf.name = "radius/" + (rlabel ? String(rlabel.characters).replace("radius ", "") : ("r" + ri));
    rf.fills = [];
    rf.clipsContent = false;
    secRadius.appendChild(rf);
    rf.x = rr.x; rf.y = rr.y;
    rf.resize(90, 100);
    var rnodes = rlabel ? [rr, rlabel] : [rr];
    for (var rn = 0; rn < rnodes.length; rn++) {
      var rnode = rnodes[rn];
      var rx = rnode.x, ry = rnode.y;
      rf.appendChild(rnode);
      rnode.x = rx - rr.x;
      rnode.y = ry - rr.y;
    }
    rw++;
  }
  out.grouped.radius = rw;
}

// ---------- 4. 动效：窄文案修正 + 时长行分组 ----------
var secMotion = findSection("05 动效");
if (secMotion) {
  var motionFixed = 0;
  var looseTexts = secMotion.children.filter(function (k) { return k.type === "TEXT" && String(k.characters).length > 6 && k.width < 60; });
  for (var mi2 = 0; mi2 < looseTexts.length; mi2++) {
    var mtxt = looseTexts[mi2];
    var host = null;
    for (var hj = 0; hj < secMotion.children.length; hj++) {
      var hk = secMotion.children[hj];
      if (hk.type !== "RECTANGLE" || hk.width < 400) continue;
      if (mtxt.x >= hk.x - 2 && mtxt.x <= hk.x + hk.width && mtxt.y >= hk.y - 2 && mtxt.y <= hk.y + hk.height) host = hk;
    }
    var newW = host ? (Math.round(host.width) === 780 ? 462 : Math.round(host.width) - 40) : 470;
    try {
      await figma.loadFontAsync(mtxt.fontName);
      mtxt.textAutoResize = "HEIGHT";
      mtxt.resize(newW, mtxt.height);
      motionFixed++;
    } catch (e) {}
  }
  out.fixed.narrowTexts = motionFixed;
  var rows = secMotion.children.filter(function (k) { return k.type === "RECTANGLE" && Math.round(k.width) === 780 && Math.round(k.height) === 42; });
  var others = secMotion.children.filter(function (k) { return rows.indexOf(k) < 0 && k.type !== "FRAME"; });
  var rowFrames = 0;
  for (var rwi = 0; rwi < rows.length; rwi++) {
    var row = rows[rwi];
    var rowGroup = [row];
    var ms = null;
    for (var oi = 0; oi < others.length; oi++) {
      var ob = others[oi];
      if (ob.parent !== secMotion) continue;
      if (ob.y >= row.y - 2 && ob.y + ob.height <= row.y + row.height + 2 && ob.x >= row.x - 2 && ob.x + ob.width <= row.x + row.width + 2) {
        rowGroup.push(ob);
        if (!ms && ob.type === "TEXT" && /^\d+ms$/.test(String(ob.characters).trim())) ms = String(ob.characters).trim();
      }
    }
    var drf = figma.createFrame();
    drf.name = "duration/" + (ms || ("row" + rwi));
    drf.fills = [];
    drf.clipsContent = false;
    secMotion.appendChild(drf);
    drf.x = row.x; drf.y = row.y;
    drf.resize(780, 42);
    for (var dgi = 0; dgi < rowGroup.length; dgi++) {
      var dnode = rowGroup[dgi];
      var dx = dnode.x, dy = dnode.y;
      drf.appendChild(dnode);
      dnode.x = dx - row.x;
      dnode.y = dy - row.y;
      others = others.filter(function (x) { return x.id !== dnode.id; });
    }
    rowFrames++;
  }
  out.grouped.durationRows = rowFrames;
}

// ---------- 5. 组件类目分组（已分组则跳过） ----------
var secComp = findSection("06 组件");
if (secComp) {
  var catFrames = secComp.children.filter(function (k) { return k.type === "FRAME" && String(k.name).indexOf("组件/") === 0; });
  if (catFrames.length >= 3) {
    out.skipped.components = "already grouped (" + catFrames.length + ")";
  } else {
    var CATS = [
      { key: "Buttons", label: "组件/按钮 Buttons", cols: 5 },
      { key: "Inputs", label: "组件/输入 Inputs", cols: 4 },
      { key: "Badges", label: "组件/徽章 Badges", cols: 4 },
      { key: "Tabs", label: "组件/标签页 Tabs", cols: 2 },
      { key: "Lists", label: "组件/列表与卡片 Lists & Cards", cols: 2 },
      { key: "Nav", label: "组件/导航与浮层 Navigation & Overlays", cols: 2 }
    ];
    function catOf(name) {
      if (String(name).indexOf("Button/") === 0) return "Buttons";
      if (String(name).indexOf("Input/") === 0) return "Inputs";
      if (String(name).indexOf("Badge/") === 0) return "Badges";
      if (String(name).indexOf("Tabs/") === 0) return "Tabs";
      if (String(name).indexOf("List/") === 0 || String(name).indexOf("Card/") === 0) return "Lists";
      return "Nav";
    }
    await figma.loadFontAsync(FONT_BOLD);
    var comps = secComp.children.filter(function (k) { return k.type === "COMPONENT"; });
    var bucket = {};
    for (var cbi = 0; cbi < comps.length; cbi++) {
      var ckey = catOf(comps[cbi].name);
      if (!bucket[ckey]) bucket[ckey] = [];
      bucket[ckey].push(comps[cbi]);
    }
    var cursorY = 84;
    var made = 0;
    for (var cc = 0; cc < CATS.length; cc++) {
      var cat = CATS[cc];
      var items = bucket[cat.key];
      if (!items || !items.length) continue;
      var maxW = 0, maxH = 0;
      for (var ii = 0; ii < items.length; ii++) { maxW = Math.max(maxW, items[ii].width); maxH = Math.max(maxH, items[ii].height); }
      var rowsN = Math.ceil(items.length / cat.cols);
      var gapX = 24, gapY = 18;
      var fw = Math.round(cat.cols * maxW + (cat.cols - 1) * gapX + 8);
      var fh = Math.round(40 + rowsN * maxH + (rowsN - 1) * gapY + 8);
      var cf = figma.createFrame();
      cf.name = cat.label;
      cf.fills = [];
      cf.clipsContent = false;
      secComp.appendChild(cf);
      cf.x = 0; cf.y = cursorY;
      cf.resize(fw, fh);
      var lt2 = figma.createText();
      lt2.fontName = FONT_BOLD;
      lt2.fontSize = 18;
      lt2.fills = [{ type: "SOLID", color: PRIMARY }];
      lt2.characters = cat.label.replace("组件/", "").split(" ")[0];
      lt2.textAutoResize = "WIDTH_AND_HEIGHT";
      cf.appendChild(lt2);
      lt2.x = 0; lt2.y = 0;
      items.sort(function (a, b) { return (a.y - b.y) || (a.x - b.x); });
      for (var ij = 0; ij < items.length; ij++) {
        var it = items[ij];
        var col = ij % cat.cols, row = Math.floor(ij / cat.cols);
        cf.appendChild(it);
        it.x = col * (maxW + gapX);
        it.y = 40 + row * (maxH + gapY);
      }
      cursorY += fh + 40;
      made++;
    }
    secComp.resize(SECTION_W, cursorY + 24);
    out.grouped.components = made;
  }
}

// ---------- 6. 区块统一堆叠 ----------
var sections = page.children.filter(isSectionFrame);
sections.sort(function (a, b) {
  var na = String(a.name).slice(0, 2), nb = String(b.name).slice(0, 2);
  return na < nb ? -1 : (na > nb ? 1 : 0);
});
var cursor2 = 0;
for (var s3 = 0; s3 < sections.length; s3++) {
  sections[s3].x = 0;
  sections[s3].y = cursor2;
  cursor2 += sections[s3].height + SECTION_GAP;
}
out.finalSections = sections.map(function (f) { return String(f.name) + "@" + Math.round(f.y) + " h=" + Math.round(f.height); });
out.pageHeight = cursor2 - SECTION_GAP;
return out;