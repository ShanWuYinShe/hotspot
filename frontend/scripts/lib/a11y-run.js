// 浏览器侧：对当前页执行 axe.run，结果异步写入 window.__axe（供 @axe-core 注入后用 eval 轮询读取）。
// 由 scripts/browser-a11y.sh 通过 agent-browser eval --stdin 注入执行。
(() => {
  window.__axe = null;
  if (typeof axe === "undefined") {
    window.__axe = JSON.stringify({ error: "axe-not-loaded" });
    return "started";
  }
  axe.run(document, { resultTypes: ["violations"] })
    .then(function (r) {
      window.__axe = JSON.stringify({
        violations: r.violations.map(function (v) {
          return {
            id: v.id,
            impact: v.impact,
            help: v.help,
            nodes: v.nodes.length,
            targets: v.nodes.slice(0, 3).map(function (n) { return n.target.join(" "); })
          };
        })
      });
    })
    .catch(function (e) {
      window.__axe = JSON.stringify({ error: String((e && e.message) || e) });
    });
  return "started";
})()
