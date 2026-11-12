
(() => {
  const VW = innerWidth, VH = innerHeight;
  const out = [];
  const intersect = (a, b) => { const x=Math.max(a.x,b.x), y=Math.max(a.y,b.y), r=Math.min(a.x+a.w,b.x+b.w), bt=Math.min(a.y+a.h,b.y+b.h); return {x,y,w:Math.max(0,r-x),h:Math.max(0,bt-y)}; };
  const cv = document.createElement('canvas'); cv.width = 1; cv.height = 1;
  const ctx = cv.getContext('2d', { willReadFrequently: true });
  const colorCache = new Map();
  const toRgba = (c) => {
    if (!c) return null;
    if (colorCache.has(c)) return colorCache.get(c);
    let out = null;
    try {
      ctx.clearRect(0, 0, 1, 1);
      ctx.fillStyle = '#010203';
      ctx.fillStyle = c;
      ctx.fillRect(0, 0, 1, 1);
      const d = ctx.getImageData(0, 0, 1, 1).data;
      out = { r: d[0], g: d[1], b: d[2], a: +(d[3] / 255).toFixed(3) };
    } catch (e) { out = null; }
    colorCache.set(c, out);
    return out;
  };
  const COLOR_TOKEN_RE = /(?:rgba?\([^)]*\)|color\([^)]*\)|oklab\([^)]*\)|oklch\([^)]*\)|lab\([^)]*\)|lch\([^)]*\)|#[0-9a-fA-F]{3,8})/g;
  const normalizeColors = (str) => {
    if (!str) return str;
    return str.replace(COLOR_TOKEN_RE, (m) => { const q = toRgba(m); return q ? 'rgba(' + q.r + ', ' + q.g + ', ' + q.b + ', ' + q.a + ')' : m; });
  };
  const hex = (c) => { const q=toRgba(c); return q ? '#'+[q.r,q.g,q.b].map(v=>v.toString(16).padStart(2,'0')).join('') : null; };
  const alpha = (c) => { const q=toRgba(c); return q ? +q.a.toFixed(3) : 1; };
  const radiiOf = (st, w, h) => {
    const v = (st.borderTopLeftRadius||'0').split(' ');
    const one = (s) => { if (!s) return 0; if (s.endsWith('%')) return Math.min(w,h) * parseFloat(s) / 100; return parseFloat(s) || 0; };
    const tl = one(st.borderTopLeftRadius), tr = one(st.borderTopRightRadius), br = one(st.borderBottomRightRadius), bl = one(st.borderBottomLeftRadius);
    return (tl||tr||br||bl) ? [ +tl.toFixed(1), +tr.toFixed(1), +br.toFixed(1), +bl.toFixed(1) ] : null;
  };
  const walk = (el, depth, clip) => {
    if (el.nodeType !== 1) return;
    const tag = el.tagName.toLowerCase();
    if (tag==='script'||tag==='style'||tag==='noscript'||tag==='template') return;
    const st = getComputedStyle(el);
    if (st.display==='none'||st.visibility==='hidden') return;
    if (st.opacity === '0') return;
    if (st.clip && st.clip !== 'auto') return;
    if (st.clipPath && st.clipPath !== 'none') return;
    const r = el.getBoundingClientRect();
    if (r.width < 0.5 || r.height < 0.5) { for (const c of el.children) walk(c, depth+1, clip); return; }
    const myClip = intersect(clip, {x:r.x,y:r.y,w:r.width,h:r.height});
    if (myClip.w <= 0.5 || myClip.h <= 0.5) return;
    let childClip = clip;
    if (st.overflow!=='visible'||st.overflowX!=='visible'||st.overflowY!=='visible') {
      const inc = intersect({x:clip.x,y:clip.y,w:clip.w,h:clip.h}, {x:r.x,y:r.y,w:r.width,h:r.height});
      if (inc.w<=0.5||inc.h<=0.5) return;
      const radii = radiiOf(st, r.width, r.height);
      childClip = { x:inc.x, y:inc.y, w:inc.w, h:inc.h, radii: radii || clip.radii || null };
    }
    const node = { tag, x:+r.x.toFixed(1), y:+r.y.toFixed(1), w:+r.width.toFixed(1), h:+r.height.toFixed(1) };
    if (clip.radii) node.clip = { x:+clip.x.toFixed(1), y:+clip.y.toFixed(1), w:+clip.w.toFixed(1), h:+clip.h.toFixed(1), r:clip.radii };
    const bg = toRgba(st.backgroundColor);
    if (bg && bg.a>0.01) { node.bg=hex(st.backgroundColor); if (bg.a<0.999) node.bgA=+bg.a.toFixed(3); }
    if (st.backgroundImage && st.backgroundImage!=='none') node.bgImg = normalizeColors(st.backgroundImage).slice(0,900);
    const bw = parseFloat(st.borderTopWidth)||0;
    if (bw>0) { const bc=toRgba(st.borderTopColor); if (bc&&bc.a>0.01) node.border={w:+bw.toFixed(1),c:hex(st.borderTopColor),a:+bc.a.toFixed(3)}; }
    const ownRadii = radiiOf(st, r.width, r.height);
    if (ownRadii) { node.r = ownRadii; node.radius = st.borderRadius; }
    if (st.boxShadow && st.boxShadow!=='none') node.shadow = st.boxShadow.slice(0,400);
    if (st.backdropFilter && st.backdropFilter!=='none') node.backdrop = st.backdropFilter;
    if (st.opacity!=='1') node.opacity=+st.opacity;
    const tc = toRgba(st.color);
    if (tc && tc.a>0.01) { node.c = hex(st.color); if (tc.a<0.999) node.cA=+tc.a.toFixed(3); }
    let text='';
    for (const cn of el.childNodes) if (cn.nodeType===3) text += cn.nodeValue||'';
    text = text.replace(/\s+/g,' ').trim();
    if (!text && (tag==='input'||tag==='textarea')) {
      text = (el.value || el.getAttribute('placeholder') || '').replace(/\s+/g,' ').trim();
      if (text) node.input = true;
    }
    if (text) {
      node.text = text.slice(0,600);
      node.font = { size:+(parseFloat(st.fontSize)||14).toFixed(1), weight:+st.fontWeight||400, family:st.fontFamily.split(',')[0].replace(/["']/g,''), lh: st.lineHeight==='normal'?0:+(parseFloat(st.lineHeight)||0).toFixed(1), ls: st.letterSpacing==='normal'?0:+(parseFloat(st.letterSpacing)||0).toFixed(2), align:st.textAlign };
      if (st.textDecorationLine && st.textDecorationLine!=='none') node.deco = st.textDecorationLine;
    }
    if (tag==='img') {
      node.img = el.currentSrc||el.src; node.fit = st.objectFit||'cover';
      if (el.complete && el.naturalWidth) {
        try {
          const targetW = Math.max(120, Math.min(480, Math.round(r.width * 2)));
          const cw = Math.min(targetW, el.naturalWidth);
          const ch = Math.max(1, Math.round(el.naturalHeight * cw / el.naturalWidth));
          const cvs = document.createElement('canvas'); cvs.width = cw; cvs.height = ch;
          const c2 = cvs.getContext('2d'); c2.drawImage(el, 0, 0, cw, ch);
          node.thumb = cvs.toDataURL('image/jpeg', 0.66);
        } catch (err) {}
      }
    }
    else if (tag==='svg') node.svg = { outer: el.outerHTML.slice(0,20000) };
    else if (tag==='canvas') node.canvas = true;
    else if (tag==='video') node.video = true;
    const hasPaint = node.bg||node.bgImg||node.border||node.text||node.img||node.svg||node.canvas||node.video;
    if (hasPaint) out.push(node);
    for (const c of el.children) walk(c, depth+1, childClip);
  };
  for (const el of document.body.children) walk(el, 0, {x:0,y:0,w:VW,h:VH,radii:null});
  return { vw:VW, vh:VH, theme:document.documentElement.className, count:out.length, nodes:out };
})()
