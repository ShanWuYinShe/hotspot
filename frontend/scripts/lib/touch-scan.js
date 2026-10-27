// 浏览器侧：390 触控目标清单（有效热区 = 视觉尺寸 + hit-slop/hit-slop-y 扩展）。
// window.__TOUCH_ALLOWLIST 由 browser-touch.sh 在每页加载后注入。
(() => {
  var SEL = 'a, button, [role="button"], input, select, textarea, summary, [tabindex]:not([tabindex="-1"])';
  var raw = window.__TOUCH_ALLOWLIST;
  var allow = Array.isArray(raw) ? raw : raw && Array.isArray(raw.rules) ? raw.rules : [];
  function matchAllow(el) {
    for (var i = 0; i < allow.length; i++) {
      try {
        if (allow[i].selector && el.matches(allow[i].selector)) return allow[i];
      } catch (e) {}
    }
    return null;
  }
  var seen = [];
  var out = [];
  document.querySelectorAll(SEL).forEach(function (el) {
    if (seen.indexOf(el) >= 0) return;
    seen.push(el);
    var r = el.getBoundingClientRect();
    if (r.width < 1 || r.height < 1) return;
    var cs = getComputedStyle(el);
    if (cs.visibility === "hidden" || cs.display === "none") return;
    var cls = typeof el.className === "string" ? el.className : "";
    var slop = cls.indexOf("hit-slop-44") >= 0 ? "44" : cls.indexOf("hit-slop-y") >= 0 ? "y" : cls.indexOf("hit-slop") >= 0 ? "both" : "";
    var ex = slop === "44" ? 26 : slop ? 12 : 0;
    var ew = r.width + (slop === "both" || slop === "44" ? ex : 0);
    var eh = r.height + (slop === "44" ? 26 : slop ? 12 : 0);
    var eff = Math.min(ew, eh);
    var rule = matchAllow(el);
    var isForm = /^(INPUT|SELECT|TEXTAREA)$/.test(el.tagName);
    var isSkip = el.tagName === "A" && el.getAttribute("href") === "#main-content";
    out.push({
      tag: el.tagName,
      text: (el.getAttribute("aria-label") || (el.textContent || "").trim().slice(0, 24) || el.getAttribute("placeholder") || "").slice(0, 30),
      w: Math.round(r.width),
      h: Math.round(r.height),
      eff: Math.round(eff),
      slop: slop,
      form: isForm,
      skip: isSkip,
      allow: rule ? rule.id : null,
      cls: cls.slice(0, 70)
    });
  });
  return JSON.stringify(out);
})()
