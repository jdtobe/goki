// Markdown → safe, enhanced HTML. Rendering happens entirely in the browser:
// marked parses GitHub-flavored Markdown, DOMPurify sanitizes the result,
// then the DOM is enhanced (links, callouts, highlighting, math, diagrams).
import { icon } from "./icons.js";
import { esc } from "./ui.js";

const V = document.querySelector('meta[name="generator"]')?.content.split(" ")[1] || "dev";
const vendor = (p) => `/-/static/vendor/${p}?v=${encodeURIComponent(V)}`;

let ready = false;

function setup() {
  if (ready) return;
  ready = true;
  window.marked.use({
    gfm: true,
    breaks: false,
    extensions: [
      {
        name: "mathBlock",
        level: "block",
        start: (src) => src.match(/^\$\$/m)?.index,
        tokenizer(src) {
          const m = /^\$\$[ \t]*\n?([\s\S]+?)\n?[ \t]*\$\$[ \t]*(?:\n+|$)/.exec(src);
          if (m) return { type: "mathBlock", raw: m[0], text: m[1].trim() };
        },
        renderer: (t) => `<div class="math-block" data-tex="${esc(t.text)}">${esc(t.text)}</div>\n`,
      },
      {
        name: "mathInline",
        level: "inline",
        start: (src) => src.indexOf("$"),
        tokenizer(src) {
          const m = /^\$(?![\s$])((?:\\.|[^\\$\n])+?)(?<![\s\\])\$(?![\d$])/.exec(src);
          if (m) return { type: "mathInline", raw: m[0], text: m[1] };
        },
        renderer: (t) => `<span class="math-inline" data-tex="${esc(t.text)}">${esc(t.text)}</span>`,
      },
      {
        name: "wikilink",
        level: "inline",
        start: (src) => src.indexOf("[["),
        tokenizer(src) {
          const m = /^\[\[([^\[\]|\n]+?)(?:\|([^\[\]\n]+?))?\]\]/.exec(src);
          if (m) return { type: "wikilink", raw: m[0], target: m[1].trim(), label: (m[2] || m[1]).trim() };
        },
        renderer: (t) => `<a class="wikilink" data-wikilink="${esc(t.target)}">${esc(t.label)}</a>`,
      },
    ],
  });
  window.DOMPurify.addHook("afterSanitizeAttributes", (node) => {
    if (node.tagName === "A" && node.getAttribute("target") === "_blank") node.setAttribute("rel", "noopener noreferrer");
  });
}

/** Splits YAML front matter from the body. Only simple keys are parsed. */
export function splitFrontMatter(src) {
  const m = /^﻿?---[ \t]*\r?\n([\s\S]*?)\r?\n(?:---|\.\.\.)[ \t]*(?:\r?\n|$)/.exec(src);
  if (!m) return { meta: {}, body: src };
  const meta = {};
  let listKey = null;
  for (const line of m[1].split(/\r?\n/)) {
    const item = /^\s+-\s+(.*)$/.exec(line);
    if (item && listKey) { meta[listKey].push(unquote(item[1])); continue; }
    const kv = /^([A-Za-z0-9_-]+)\s*:\s*(.*)$/.exec(line);
    if (!kv) continue;
    const [, k, v] = kv;
    if (v === "") { meta[k] = []; listKey = k; continue; }
    listKey = null;
    if (/^\[.*\]$/.test(v)) meta[k] = v.slice(1, -1).split(",").map((s) => unquote(s.trim())).filter(Boolean);
    else meta[k] = unquote(v);
  }
  return { meta, body: src.slice(m[0].length) };
}

const unquote = (s) => s.replace(/^(['"])(.*)\1$/, "$2");

export function slugify(text) {
  return text.toLowerCase().trim().replace(/[^\p{L}\p{N}\s_-]/gu, "").replace(/\s+/g, "-");
}

const routeOf = (file) => file.replace(/(^|\/)(index|readme)\.md$/i, "").replace(/\.md$/i, "");
export const encodePath = (p) => p.split("/").map(encodeURIComponent).join("/");

/**
 * Renders Markdown. ctx: { file, index } where file is the page's relative
 * file path (for resolving relative links) and index is the wiki route index.
 * Returns { el, meta, headings }.
 */
export function render(src, ctx = {}) {
  setup();
  const { meta, body } = splitFrontMatter(src || "");
  const html = window.marked.parse(body);
  const clean = window.DOMPurify.sanitize(html, {
    ADD_ATTR: ["target"],
    FORBID_TAGS: ["style", "form"],
    ALLOW_UNKNOWN_PROTOCOLS: false,
  });
  const el = document.createElement("div");
  el.className = "prose";
  el.innerHTML = clean;

  const dir = (ctx.file || "").includes("/") ? ctx.file.slice(0, ctx.file.lastIndexOf("/") + 1) : "";
  const base = new URL("/" + encodePath(dir), location.origin);

  const headings = enhanceHeadings(el);
  enhanceFrontMatter(el, meta);
  enhanceLinks(el, base, ctx.index);
  enhanceMedia(el, base);
  enhanceCallouts(el);
  enhanceTasks(el);
  enhanceTables(el);
  enhanceCode(el);
  return { el, meta, headings };
}

function enhanceHeadings(el) {
  const seen = new Map();
  const out = [];
  for (const hd of el.querySelectorAll("h1, h2, h3, h4, h5, h6")) {
    let id = slugify(hd.textContent) || "section";
    const n = seen.get(id) || 0;
    seen.set(id, n + 1);
    if (n) id += "-" + n;
    hd.id = id;
    const a = document.createElement("a");
    a.className = "anchor";
    a.href = "#" + id;
    a.setAttribute("aria-label", "Link to this section");
    a.textContent = "#";
    hd.prepend(a);
    out.push({ id, depth: +hd.tagName[1], text: hd.textContent.replace(/^#/, "").trim() });
  }
  return out;
}

function enhanceFrontMatter(el, meta) {
  const tags = Array.isArray(meta.tags) ? meta.tags : typeof meta.tags === "string" ? meta.tags.split(/[,\s]+/).filter(Boolean) : [];
  if (!tags.length && !meta.description) return;
  const h1 = el.querySelector("h1");
  const anchorEl = h1 && h1 === el.firstElementChild ? h1 : null;
  const frag = document.createDocumentFragment();
  if (meta.description) {
    const p = document.createElement("p");
    p.className = "lede";
    p.textContent = meta.description;
    frag.append(p);
  }
  if (tags.length) {
    const wrap = document.createElement("div");
    wrap.className = "frontmatter";
    for (const t of tags) {
      const s = document.createElement("span");
      s.className = "tag";
      s.textContent = t;
      wrap.append(s);
    }
    frag.append(wrap);
  }
  if (anchorEl) anchorEl.after(frag);
  else el.prepend(frag);
}

const hasScheme = /^[a-z][a-z0-9+.-]*:/i;

function enhanceLinks(el, base, index) {
  for (const a of el.querySelectorAll("a[data-wikilink]")) {
    const target = a.dataset.wikilink;
    const [pathPart, hash] = target.split("#");
    const resolved = index ? index.resolveWiki(pathPart, base) : null;
    const route = resolved ?? pathPart.replace(/\.md$/i, "").replace(/^\//, "");
    a.href = "/" + encodePath(route) + (hash ? "#" + slugify(hash) : "");
    a.dataset.internal = "";
    if (!resolved && index) a.classList.add("missing");
  }
  for (const a of el.querySelectorAll("a[href]")) {
    if (a.classList.contains("anchor") || a.dataset.wikilink) continue;
    const href = a.getAttribute("href");
    if (href.startsWith("#")) continue;
    if (hasScheme.test(href) || href.startsWith("//")) {
      if (!/^mailto:|^tel:/i.test(href)) {
        a.target = "_blank";
        a.rel = "noopener noreferrer";
        a.classList.add("external");
      }
      continue;
    }
    let url;
    try { url = new URL(href, base); } catch { continue; }
    let path = decodeURIComponent(url.pathname).replace(/^\//, "");
    const ext = (/\.([^./]+)$/.exec(path) || [])[1];
    if (ext && ext.toLowerCase() !== "md") {
      a.href = url.pathname + url.search + url.hash;
      continue;
    }
    path = routeOf(path).replace(/\/$/, "");
    a.href = "/" + encodePath(path) + url.search + url.hash;
    a.dataset.internal = "";
    if (index && !index.has(path)) a.classList.add("missing");
  }
}

function enhanceMedia(el, base) {
  for (const img of el.querySelectorAll("img[src], video[src], source[src]")) {
    const src = img.getAttribute("src");
    if (!hasScheme.test(src) && !src.startsWith("//") && !src.startsWith("/")) {
      try { img.setAttribute("src", new URL(src, base).pathname); } catch {}
    }
    if (img.tagName === "IMG") { img.loading = "lazy"; img.decoding = "async"; }
  }
}

const CALLOUTS = {
  note: ["Note", "info"],
  tip: ["Tip", "lightbulb"],
  important: ["Important", "star"],
  warning: ["Warning", "alert"],
  caution: ["Caution", "octagon"],
};

function enhanceCallouts(el) {
  for (const bq of el.querySelectorAll("blockquote")) {
    const p = bq.firstElementChild;
    if (!p || p.tagName !== "P") continue;
    const first = p.firstChild;
    if (!first || first.nodeType !== Node.TEXT_NODE) continue;
    const m = /^\s*\[!(\w+)\][ \t]*([^\n]*)\n?/.exec(first.textContent);
    if (!m) continue;
    const kind = m[1].toLowerCase();
    const [label, ic] = CALLOUTS[kind] || CALLOUTS.note;
    first.textContent = first.textContent.slice(m[0].length);
    if (p.firstChild && p.firstChild.nodeName === "BR") p.firstChild.remove();
    if (!p.textContent.trim() && !p.querySelector("img")) p.remove();
    const div = document.createElement("div");
    div.className = "callout " + (CALLOUTS[kind] ? kind : "note");
    const title = document.createElement("div");
    title.className = "callout-title";
    title.innerHTML = icon(ic);
    title.append(m[2].trim() || label);
    div.append(title, ...bq.childNodes);
    bq.replaceWith(div);
  }
}

function enhanceTasks(el) {
  for (const box of el.querySelectorAll('li > input[type="checkbox"], li > p:first-child > input[type="checkbox"]')) {
    const li = box.closest("li");
    li.classList.add("task");
    if (box.checked) li.classList.add("is-done");
  }
}

function enhanceTables(el) {
  for (const t of el.querySelectorAll("table")) {
    const wrap = document.createElement("div");
    wrap.className = "table-wrap";
    t.replaceWith(wrap);
    wrap.append(t);
  }
}

function enhanceCode(el) {
  for (const code of el.querySelectorAll("pre > code")) {
    const pre = code.parentElement;
    const lang = ((/language-([\w+#.-]+)/.exec(code.className) || [])[1] || "").toLowerCase();
    if (lang === "mermaid") {
      const div = document.createElement("div");
      div.className = "mermaid-block";
      div.dataset.src = code.textContent;
      pre.replaceWith(div);
      continue;
    }
    if (lang === "math" || lang === "latex" || lang === "tex") {
      const div = document.createElement("div");
      div.className = "math-block";
      div.dataset.tex = code.textContent;
      div.textContent = code.textContent;
      pre.replaceWith(div);
      continue;
    }
    if (window.hljs && lang && window.hljs.getLanguage(lang)) {
      try { window.hljs.highlightElement(code); } catch {}
    }
    const wrap = document.createElement("div");
    wrap.className = "code-block";
    pre.replaceWith(wrap);
    wrap.append(pre);
    if (lang) {
      const l = document.createElement("span");
      l.className = "code-lang";
      l.textContent = lang;
      wrap.append(l);
    }
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "code-copy";
    btn.setAttribute("aria-label", "Copy code");
    btn.innerHTML = icon("copy");
    btn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(code.textContent);
        btn.innerHTML = icon("check");
        btn.classList.add("done");
        setTimeout(() => { btn.innerHTML = icon("copy"); btn.classList.remove("done"); }, 1400);
      } catch {}
    });
    wrap.append(btn);
  }
}

// ── Lazy enhancements: math and diagrams ──────────────────────────────────
const loaded = new Map();

function loadScript(src) {
  if (!loaded.has(src)) {
    loaded.set(src, new Promise((resolve, reject) => {
      const s = document.createElement("script");
      s.src = src;
      s.onload = resolve;
      s.onerror = () => reject(new Error("failed to load " + src));
      document.head.append(s);
    }));
  }
  return loaded.get(src);
}

function loadCSS(href) {
  if (document.querySelector(`link[href="${href}"]`)) return;
  const l = document.createElement("link");
  l.rel = "stylesheet";
  l.href = href;
  document.head.append(l);
}

let mermaidSeq = 0;

/** Runs asynchronous enhancements (KaTeX, Mermaid) on a rendered element. */
export async function enhance(el) {
  const math = el.querySelectorAll(".math-inline, .math-block");
  if (math.length) {
    loadCSS(vendor("katex/katex.min.css"));
    try {
      await loadScript(vendor("katex/katex.min.js"));
      for (const m of math) {
        try {
          window.katex.render(m.dataset.tex, m, { displayMode: m.classList.contains("math-block"), throwOnError: false, output: "htmlAndMathml" });
        } catch {}
      }
    } catch {}
  }
  await renderDiagrams(el);
}

export async function renderDiagrams(el) {
  const blocks = el.querySelectorAll(".mermaid-block[data-src]");
  if (!blocks.length) return;
  try {
    await loadScript(vendor("mermaid.min.js"));
  } catch {
    return;
  }
  const dark = getComputedStyle(document.documentElement).colorScheme.includes("dark");
  const c = themeColors(el);
  window.mermaid.initialize({
    startOnLoad: false,
    securityLevel: "strict",
    theme: "base",
    fontFamily: getComputedStyle(el).getPropertyValue("--font-ui") || "system-ui",
    themeVariables: {
      darkMode: dark,
      background: c.surface,
      primaryColor: c.node,
      primaryTextColor: c.text,
      primaryBorderColor: c.accent,
      secondaryColor: c.node,
      tertiaryColor: c.cluster,
      lineColor: c.line,
      textColor: c.text,
      clusterBkg: c.cluster,
      clusterBorder: c.border,
      edgeLabelBackground: c.surface,
      nodeTextColor: c.text,
      mainBkg: c.node,
      nodeBorder: c.accent,
    },
  });
  for (const b of blocks) {
    try {
      const { svg } = await window.mermaid.render("mmd-" + ++mermaidSeq, b.dataset.src);
      b.innerHTML = svg;
      b.classList.remove("is-error");
      fitDiagram(b.querySelector("svg"));
    } catch (e) {
      b.classList.add("is-error");
      b.textContent = "Diagram error: " + (e && e.message ? e.message : e);
    }
  }
}

// themeColors resolves theme tokens to plain rgb() strings Mermaid can parse,
// compositing translucent tokens over the page background.
function themeColors(el) {
  // A fresh probe per token: reusing one can return a stale computed color.
  const read = (expr) => {
    const probe = document.createElement("span");
    probe.style.color = expr;
    el.append(probe);
    const color = getComputedStyle(probe).color;
    probe.remove();
    return color;
  };
  const parse = (str) => {
    const m = str.match(/[\d.]+/g) || [];
    if (str.startsWith("color(")) return [m[0] * 255, m[1] * 255, m[2] * 255, m[3] ?? 1].map(Number);
    return [m[0], m[1], m[2], m[3] ?? 1].map(Number);
  };
  const bg = parse(read("var(--bg)"));
  const solid = (expr, over = bg) => {
    const [r, g, b, a] = parse(read(expr));
    const mix = (x, y) => Math.round(x * a + y * (1 - a));
    return `rgb(${mix(r, over[0])}, ${mix(g, over[1])}, ${mix(b, over[2])})`;
  };
  const surface = solid("var(--surface)");
  const surfaceRGB = parse(surface);
  const out = {
    surface,
    node: solid("var(--surface-3)", surfaceRGB),
    cluster: solid("var(--surface-2)", surfaceRGB),
    text: solid("var(--text)", surfaceRGB),
    line: solid("var(--text-3)", surfaceRGB),
    border: solid("var(--border-strong)", surfaceRGB),
    accent: solid("var(--accent)", surfaceRGB),
  };
  return out;
}

// Mermaid can over-estimate a diagram's bounds while measuring off-screen,
// leaving it tiny inside a huge canvas. Re-fit the viewBox to the content.
function fitDiagram(svg) {
  if (!svg || !svg.isConnected) return;
  let box;
  try { box = svg.getBBox(); } catch { return; }
  if (!box.width || !box.height) return;
  const pad = 8;
  svg.setAttribute("viewBox", `${box.x - pad} ${box.y - pad} ${box.width + pad * 2} ${box.height + pad * 2}`);
  svg.style.maxWidth = `${Math.ceil(box.width + pad * 2)}px`;
  svg.removeAttribute("height");
}

/**
 * Toggles the nth task-list checkbox in Markdown source and returns the new
 * source, or null if it could not be located. Code fences are skipped so the
 * numbering matches the rendered checkboxes.
 */
export function toggleTask(src, n, checked) {
  const lines = src.split("\n");
  let fence = null, count = 0;
  const { body } = splitFrontMatter(src);
  const offset = src.length - body.length;
  const skipLines = offset ? src.slice(0, offset).split("\n").length - 1 : 0;
  for (let i = skipLines; i < lines.length; i++) {
    const line = lines[i];
    const f = /^\s*(```+|~~~+)/.exec(line);
    if (f) {
      if (!fence) fence = f[1][0];
      else if (f[1][0] === fence) fence = null;
      continue;
    }
    if (fence) continue;
    const m = /^((?:\s*>)*\s*(?:[-*+]|\d+[.)])\s+)\[([ xX])\]/.exec(line);
    if (!m) continue;
    if (count++ === n) {
      lines[i] = m[1] + (checked ? "[x]" : "[ ]") + line.slice(m[0].length);
      return lines.join("\n");
    }
  }
  return null;
}
