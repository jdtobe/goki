// ⌘K command palette: jump to pages, search content, run commands.
import { api } from "./api.js";
import { icon } from "./icons.js";
import { h, esc, openOverlay, isMac } from "./ui.js";

/** Scores a fuzzy subsequence match; returns null when q doesn't match. */
export function fuzzy(q, text) {
  if (!q) return { score: 0, marks: [] };
  const t = text.toLowerCase();
  const query = q.toLowerCase();
  const direct = t.indexOf(query);
  if (direct >= 0) {
    const marks = [];
    for (let i = 0; i < query.length; i++) marks.push(direct + i);
    const wordStart = direct === 0 || /[\s/_-]/.test(t[direct - 1]);
    return { score: 100 - direct + (wordStart ? 40 : 0) + (t.length === query.length ? 50 : 0), marks };
  }
  let ti = 0, score = 0, last = -2;
  const marks = [];
  for (const ch of query) {
    if (ch === " ") continue;
    const at = t.indexOf(ch, ti);
    if (at < 0) return null;
    score += at === last + 1 ? 6 : 1;
    if (at === 0 || /[\s/_-]/.test(t[at - 1])) score += 4;
    marks.push(at);
    last = at;
    ti = at + 1;
  }
  return { score: score - t.length * 0.05, marks };
}

function highlight(text, marks) {
  if (!marks || !marks.length) return esc(text);
  const set = new Set(marks);
  let out = "";
  let open = false;
  [...text].forEach((ch, i) => {
    if (set.has(i) && !open) { out += "<mark>"; open = true; }
    if (!set.has(i) && open) { out += "</mark>"; open = false; }
    out += esc(ch);
  });
  return out + (open ? "</mark>" : "");
}

/**
 * Opens the palette. app provides: pages() → [{path,title,file}],
 * recent() → [path], commands() → [{id,title,icon,kbd,run}], go(path).
 */
export function openPalette(app, initial = "") {
  const input = h("input", { class: "palette-input", placeholder: "Search pages, content and commands…", spellcheck: "false", autocomplete: "off", "aria-label": "Search" });
  const results = h("div", { class: "palette-results", role: "listbox" });
  const box = h("div", { class: "palette", role: "dialog", "aria-label": "Command palette" },
    h("div", { class: "palette-input-wrap", html: icon("search") }, input),
    results,
    h("div", { class: "palette-foot" },
      h("span", { html: "<kbd>↑</kbd><kbd>↓</kbd> navigate" }),
      h("span", { html: "<kbd>↵</kbd> open" }),
      h("span", { html: "<kbd>esc</kbd> close" }),
      h("span", { style: "margin-left:auto", html: `<kbd>${isMac ? "⌘" : "Ctrl"}</kbd><kbd>K</kbd>` }),
    ),
  );

  const { close } = openOverlay(box);
  let items = [];
  let active = 0;
  let seq = 0;
  let serverHits = [];

  function build() {
    const q = input.value.trim();
    const groups = [];
    const pages = app.pages();
    if (!q) {
      const recent = app.recent().map((p) => pages.find((x) => x.path === p)).filter(Boolean).slice(0, 6);
      if (recent.length) groups.push(["Recent", recent.map(pageItem)]);
      groups.push(["Commands", app.commands().slice(0, 8).map(cmdItem)]);
    } else {
      const matched = pages
        .map((p) => {
          const a = fuzzy(q, p.title);
          const b = fuzzy(q, p.path);
          if (!a && !b) return null;
          const best = !b || (a && a.score >= b.score - 10) ? { ...p, score: a.score + 10, tmarks: a.marks } : { ...p, score: b.score, pmarks: b.marks };
          return best;
        })
        .filter(Boolean)
        .sort((x, y) => y.score - x.score)
        .slice(0, 8);
      if (matched.length) groups.push(["Pages", matched.map(pageItem)]);
      const seen = new Set(matched.map((m) => m.path));
      const content = serverHits.filter((hit) => !seen.has(hit.path)).slice(0, 8);
      if (content.length) groups.push(["In page content", content.map(hitItem)]);
      const cmds = app.commands().map((c) => ({ c, m: fuzzy(q, c.title) })).filter((x) => x.m).sort((a, b) => b.m.score - a.m.score).slice(0, 5);
      if (cmds.length) groups.push(["Commands", cmds.map(({ c, m }) => cmdItem(c, m.marks))]);
      if (app.canWrite) groups.push(["Create", [createItem(q)]]);
    }
    items = [];
    results.replaceChildren();
    for (const [label, list] of groups) {
      results.append(h("div", { class: "palette-group", text: label }));
      for (const it of list) {
        const idx = items.length;
        items.push(it);
        const row = h("div", { class: "palette-item", role: "option", html: `<span class="pi-icon">${icon(it.icon)}</span>` },
          h("div", { class: "pi-text", html: `<div class="pi-title">${it.titleHTML}</div>${it.subHTML ? `<div class="pi-sub">${it.subHTML}</div>` : ""}` }),
          it.kbd ? h("kbd", { text: it.kbd }) : null);
        row.addEventListener("mousemove", () => setActive(idx, false));
        row.addEventListener("click", () => choose(idx));
        it.row = row;
        results.append(row);
      }
    }
    if (!items.length) results.append(h("div", { class: "palette-empty", text: q ? "No results" : "Nothing here yet" }));
    setActive(Math.min(active, Math.max(0, items.length - 1)), false);
  }

  const pageItem = (p) => ({
    icon: p.type === "dir" ? "folder" : "doc",
    titleHTML: highlight(p.title, p.tmarks),
    subHTML: highlight(p.path || "/", p.pmarks),
    run: () => app.go(p.path),
  });
  const hitItem = (hit) => ({
    icon: "search",
    titleHTML: esc(hit.title),
    subHTML: markTerms(hit.snippet, input.value),
    run: () => app.go(hit.path, input.value.trim()),
  });
  const cmdItem = (c, marks) => ({ icon: c.icon, titleHTML: highlight(c.title, marks), kbd: c.kbd, run: c.run });
  const createItem = (q) => ({
    icon: "doc-plus",
    titleHTML: `Create page “${esc(q)}”`,
    subHTML: "",
    run: () => app.createPage(q),
  });

  function markTerms(snippet, q) {
    let html = esc(snippet);
    for (const term of q.trim().split(/\s+/).filter((t) => t.length > 1)) {
      const re = new RegExp("(" + esc(term).replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + ")", "gi");
      html = html.replace(re, "<mark>$1</mark>");
    }
    return html;
  }

  function setActive(i, scroll = true) {
    if (!items.length) return;
    items[active]?.row?.classList.remove("is-active");
    active = (i + items.length) % items.length;
    const row = items[active].row;
    row.classList.add("is-active");
    if (scroll) row.scrollIntoView({ block: "nearest" });
  }

  function choose(i) {
    const it = items[i];
    if (!it) return;
    close();
    it.run();
  }

  let timer = 0;
  input.addEventListener("input", () => {
    active = 0;
    build();
    clearTimeout(timer);
    const q = input.value.trim();
    if (q.length < 2) { serverHits = []; return; }
    const my = ++seq;
    timer = setTimeout(async () => {
      try {
        const hits = await api.search(q);
        if (my !== seq) return;
        serverHits = hits;
        build();
      } catch {}
    }, 140);
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "ArrowDown" || (e.ctrlKey && e.key === "n")) { e.preventDefault(); setActive(active + 1); }
    else if (e.key === "ArrowUp" || (e.ctrlKey && e.key === "p")) { e.preventDefault(); setActive(active - 1); }
    else if (e.key === "Enter") { e.preventDefault(); choose(active); }
  });

  input.value = initial;
  build();
  if (initial) input.dispatchEvent(new Event("input"));
  requestAnimationFrame(() => input.focus());
}
