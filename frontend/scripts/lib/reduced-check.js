// 浏览器侧：reduced-motion 检查——关键类残留 + 非允许 CSS 动画时长（>20ms）。
(() => {
  var residualSel = ".page-enter,.state-pop,.content-fade";
  var residual = [];
  document.querySelectorAll(residualSel).forEach(function (el) {
    residual.push(el.className);
  });
  var allowed = '.animate-spin,.spinner,.skeleton,[data-skeleton],[role="status"],[role="progressbar"],.animate-ping';
  var long = [];
  var all = document.getAnimations ? document.getAnimations() : [];
  all.forEach(function (a) {
    try {
      if (a.constructor && a.constructor.name === "CSSTransition") return;
      var t = a.effect && a.effect.getTiming ? a.effect.getTiming() : null;
      var d = t && typeof t.duration === "number" ? t.duration : 0;
      if (d <= 20) return;
      var el = a.effect && a.effect.target;
      if (el && el.closest && el.closest(allowed)) return;
      long.push({
        tag: el ? el.tagName : "",
        cls: el && typeof el.className === "string" ? el.className.slice(0, 60) : "",
        duration: Math.round(d),
        name: a.animationName || ""
      });
    } catch (e) {}
  });
  return JSON.stringify({ residual: residual.slice(0, 5), long: long.slice(0, 10), longCount: long.length, animations: all.length });
})()
