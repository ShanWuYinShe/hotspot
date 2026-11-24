// 浏览器侧：对当前页执行 axe.run，结果异步写入 window.__axe（供 @axe-core 注入后用 eval 轮询读取）。
// 由 scripts/browser-a11y.sh 通过 agent-browser eval --stdin 注入执行。
//
// 媒体卡压字层（[data-card-footer] / [data-card-children]）的背景是「图片 + 主题渐变遮罩」：
// 遮罩是 absolute 兄弟节点，axe 只能回溯到卡片容器的 surface 背景色，于是白字（text-content-inverse）
// 在浅色主题被误判成 white-on-light（主题矩阵 forest/inkwash light 各 1~2 条 serious）。
// 这些文本的真实背景是遮罩渐变（最弱区间 rgba(10,18,12,0.64) 叠白底，白字对比度仍 ≈6.2:1），
// 属工具无法合成的已知误报；仅对 color-contrast 规则过滤命中媒体压字层的节点，其余检查照旧执行。
(() => {
  window.__axe = null;
  if (typeof axe === "undefined") {
    window.__axe = JSON.stringify({ error: "axe-not-loaded" });
    return "started";
  }
  function inMediaOverlay(node) {
    var targets = node.target || [];
    for (var i = 0; i < targets.length; i++) {
      try {
        var el = document.querySelector(targets[i]);
        if (el && el.closest("[data-card-footer],[data-card-children]")) return true;
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