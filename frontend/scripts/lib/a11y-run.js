// 浏览器侧：对当前页执行 axe.run，结果异步写入 window.__axe（供 @axe-core 注入后用 eval 轮询读取）。
// 由 scripts/browser-a11y.sh 通过 agent-browser eval --stdin 注入执行。
//
// 媒体卡压字层（[data-card-footer] / [data-card-children]）的背景是「图片 + 主题渐变遮罩」：
// 遮罩是 absolute 兄弟节点，axe 只能回溯到卡片容器的 surface 背景色，于是白字（text-content-inverse）
// 在浅色主题被误判成 white-on-light（主题矩阵 forest/inkwash light 各 1~2 条 serious）。
// 这些文本的真实背景是遮罩渐变（最弱区间叠纯白图后白字对比度 5.60~5.97:1，见 docs/frontend-ui-audit.md），
// 属工具无法合成的已知误报。豁免范围必须收紧到「确有主图 + 纵排压字层」：无图/加载失败卡是 surface
// 实底，横排卡 footer 是 bg-surface-card-menu 实底面板，axe 对它们的背景回溯有效，不能一并吞掉。
(() => {
  window.__axe = null;
  if (typeof axe === "undefined") {
    window.__axe = JSON.stringify({ error: "axe-not-loaded" });
    return "started";
  }
  function isTransparent(el) {
    var bg = getComputedStyle(el).backgroundColor;
    return !bg || bg === "transparent" || bg === "rgba(0, 0, 0, 0)";
  }
  function mainImage(card) {
    // MediaCard 主图是 article 的直接子节点；头像等嵌套 img 不算，避免把无图卡误判成图片卡。
    for (var i = 0; i < card.children.length; i++) {
      if (card.children[i].tagName === "IMG") return card.children[i];
    }
    return null;
  }
  function inMediaOverlay(node) {
    var targets = node.target || [];
    for (var i = 0; i < targets.length; i++) {
      try {
        var el = document.querySelector(targets[i]);
        if (!el) continue;
        var layer = el.closest("[data-card-footer],[data-card-children]");
        if (!layer) continue;
        var card = layer.closest("article");
        if (!card || !mainImage(card)) continue;
        if (layer.closest("[data-card-footer]")) {
          // 纵排 footer 直属于卡片且透明；横排 footer 是实底面板，不豁免。
          var panel = null;
          for (var j = 0; j < card.children.length; j++) {
            if (card.children[j].hasAttribute("data-card-footer")) { panel = card.children[j]; break; }
          }
          if (!panel || !isTransparent(panel)) continue;
        }
        return true;
      } catch (e) {}
    }
    return false;
  }
  axe.run(document, { resultTypes: ["violations"] })
    .then(function (r) {
      var ignored = 0;
      var violations = [];
      r.violations.forEach(function (v) {
        var nodes = v.nodes;
        if (v.id === "color-contrast") {
          var kept = nodes.filter(function (n) { return !inMediaOverlay(n); });
          ignored += nodes.length - kept.length;
          if (kept.length === 0) return;
          nodes = kept;
        }
        violations.push({
          id: v.id,
          impact: v.impact,
          help: v.help,
          nodes: nodes.length,
          targets: nodes.slice(0, 3).map(function (n) { return n.target.join(" "); })
        });
      });
      window.__axe = JSON.stringify({ violations: violations, contrastMediaOverlayIgnored: ignored });
    })
    .catch(function (e) {
      window.__axe = JSON.stringify({ error: String((e && e.message) || e) });
    });
  return "started";
})()
