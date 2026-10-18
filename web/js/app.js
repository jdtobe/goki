// goki browser application: routing, sidebar, page views and editing.
import { api, subscribe, ApiError } from "./api.js";
import { createEditor } from "./editor.js";
import { icon, hydrateIcons } from "./icons.js";
import { render, enhance, renderDiagrams, toggleTask, encodePath } from "./markdown.js";
import { openPalette } from "./palette.js";
import { initThemes, apply as applyTheme, allThemes, openAppearance, onThemeChange } from "./themes.js";
import {
  h, fromHTML, toast, dialog, confirmDialog, menu, iconButton, timeAgo, formatDate, plural, mod, topOverlay, closePopover,
} from "./ui.js";

const $ = (s) => document.querySelector(s);
const root = document.documentElement;

const state = {
  config: null,
  tree: null,
  index: null,
  path: "",
  edit: false,
  current: null, // last /-/api/page response
  editor: null,
  open: new Set(load("goki.openDirs", [])),
  recent: load("goki.recent", []),
};

// ── Route index ───────────────────────────────────────────────────────────
class WikiIndex {
  constructor(tree) {
    this.routes = new Map();
    this.lower = new Map();
    this.list = [];
    this.root = tree;
    const walk = (node) => {
      for (const c of node.children || []) {
        const entry = { path: c.path, title: c.title, file: c.file, type: c.type, node: c };
        this.routes.set(c.path, entry);
        this.lower.set(c.path.toLowerCase(), entry);
        this.list.push(entry);
        if (c.type === "dir") walk(c);
      }
    };
    walk(tree);
    if (tree.file) this.list.unshift({ path: "", title: "Home", file: tree.file, type: "page", node: tree });
  }
  get(path) {
    if (path === "") return { path: "", title: "Home", file: this.root.file, type: "dir", node: this.root };
    return this.routes.get(path) || this.lower.get(path.toLowerCase());
  }
  has(path) {
    return path === "" || this.routes.has(path) || this.lower.has(path.toLowerCase());
  }
  /** Resolves [[Target]] by path, then relative path, then by name/title. */
  resolveWiki(target, base) {
    const t = target.trim().replace(/\.md$/i, "").replace(/^\/+|\/+$/g, "");
    if (!t) return "";
    const exact = this.get(t);
    if (exact) return exact.path;
    const baseDir = decodeURIComponent(base.pathname).replace(/^\/|\/$/g, "");
    const rel = this.get(baseDir ? `${baseDir}/${t}` : t);
    if (rel) return rel.path;
    const norm = (s) => s.toLowerCase().replace(/[\s_-]+/g, " ").trim();
    const want = norm(t.split("/").pop());
    const hits = this.list.filter((e) => e.path && (norm(e.path.split("/").pop()) === want || norm(e.title) === want));
    if (!hits.length) return null;
    const near = hits.find((e) => e.path.startsWith(baseDir ? baseDir + "/" : "") && !e.path.slice(baseDir.length + 1).includes("/"));
    return (near || hits[0]).path;
  }
}

// ── Boot ──────────────────────────────────────────────────────────────────
async function boot() {
  hydrateIcons();
  try {
    state.config = await api.config();
  } catch (e) {
    $("#view").replaceChildren(emptyState("alert", "Can't reach goki", "The server isn't responding. Is goki still running?", null));
    return;
  }
  const cfg = state.config;
  initThemes(cfg);
  const forced = new URLSearchParams(location.search).get("theme");
  applyTheme(forced || localGet("goki.theme") || cfg.theme, false);
  root.classList.toggle("read-only", cfg.readOnly);
  $("#brand-title").textContent = cfg.title;
  document.querySelectorAll("[data-kbd-mod]").forEach((k) => (k.textContent = mod.trim() + "K"));

  await loadTree();
  wireChrome();
  wireKeyboard();
  wireLinks();
  window.addEventListener("popstate", () => route());
  window.addEventListener("beforeunload", (e) => {
    if (state.editor && state.editor.dirty) { e.preventDefault(); e.returnValue = ""; }
  });
  onThemeChange(() => {
    const doc = $("#view .prose");
    if (doc) renderDiagrams(doc);
  });
  if (cfg.watch) subscribe(onServerChange);
  await route();
}

async function loadTree() {
  try {
    state.tree = await api.tree();
  } catch (e) {
    toast("Couldn't load pages: " + e.message, { type: "error" });
    state.tree = { type: "dir", path: "", children: [] };
  }
  state.index = new WikiIndex(state.tree);
  renderTree();
}

// ── Routing ───────────────────────────────────────────────────────────────
function currentLocation() {
  let path = decodeURIComponent(location.pathname).replace(/^\/+|\/+$/g, "");
  const params = new URLSearchParams(location.search);
  if (/\.md$/i.test(path)) {
    path = path.replace(/(^|\/)(index|readme)\.md$/i, "").replace(/\.md$/i, "");
    params.delete("raw");
    history.replaceState(null, "", "/" + encodePath(path) + (params.toString() ? "?" + params : "") + location.hash);
  }
  return { path, edit: params.has("edit"), q: params.get("q") || "" };
}

async function navigate(path, { edit = false, replace = false, hash = "", q = "" } = {}) {
  if (!(await leaveEditor())) return;
  const params = new URLSearchParams();
  if (edit) params.set("edit", "");
  if (q) params.set("q", q);
  const qs = params.toString().replace(/=$/, "").replace(/=&/g, "&");
  const url = "/" + encodePath(path) + (qs ? "?" + qs : "") + (hash || "");
  if (replace) history.replaceState(null, "", url);
  else history.pushState(null, "", url);
  root.classList.remove("sidebar-open");
  await route();
}

/** Asks before abandoning unsaved edits. Resolves false to stay. */
async function leaveEditor() {
  const ed = state.editor;
  if (!ed || !ed.dirty) return true;
  const choice = await dialog({
    title: "Save changes?",
    message: `You have unsaved changes to “${ed.file}”.`,
    icon: "pencil",
    actions: [
      { label: "Keep Editing", value: "stay" },
      { label: "Discard", value: "discard" },
      { label: "Save", value: "save", primary: true },
    ],
  });
  if (choice === "save") return await saveEditor();
  if (choice === "discard") { ed.discardDraft(); return true; }
  return false;
}

let routeSeq = 0;

async function route() {
  const seq = ++routeSeq;
  const { path, edit, q } = currentLocation();
  let res;
  try {
    res = await api.page(path);
  } catch (e) {
    if (seq !== routeSeq) return;
    showError(e);
    return;
  }
  if (seq !== routeSeq) return;
  state.path = path;
  state.edit = edit && !state.config.readOnly;
  state.current = res;
  state.editor = null;
  $("#main").classList.toggle("is-editing", state.edit);
  highlightTree();
  if (state.edit) showEditor(res);
  else {
    showView(res, q);
    remember(path, res);
  }
}

function remember(path, res) {
  if (res.kind === "missing") return;
  state.recent = [path, ...state.recent.filter((p) => p !== path)].slice(0, 12);
  save("goki.recent", state.recent);
}

// ── Views ─────────────────────────────────────────────────────────────────
function setTitle(t) {
  document.title = t && t !== state.config.title ? `${t} · ${state.config.title}` : state.config.title;
}

function showView(res, q) {
  const view = $("#view");
  const scroller = $("#scroller");
  const keepScroll = res.__refresh ? scroller.scrollTop : 0;
  let content;
  if (res.kind === "missing") content = missingView(res);
  else if (res.page) content = pageView(res);
  else content = folderView(res);
  renderToolbar(res);
  view.replaceChildren(content);
  view.style.animation = res.__refresh ? "none" : "";
  if (!res.__refresh) { void view.offsetWidth; }
  scroller.scrollTop = keepScroll;
  const prose = view.querySelector(".prose");
  if (prose) enhance(prose).then(() => scrollToHash(q));
  else scrollToHash();
}

function scrollToHash(q) {
  const id = decodeURIComponent(location.hash.slice(1));
  const target = id && document.getElementById(id);
  if (target) target.scrollIntoView({ block: "start" });
  if (q) flashText(q);
}

function flashText(q) {
  const term = q.trim().split(/\s+/)[0];
  const prose = $("#view .prose");
  if (!term || !prose) return;
  const walker = document.createTreeWalker(prose, NodeFilter.SHOW_TEXT);
  const lower = term.toLowerCase();
  while (walker.nextNode()) {
    const n = walker.currentNode;
    const i = n.textContent.toLowerCase().indexOf(lower);
    if (i < 0) continue;
    const range = document.createRange();
    range.setStart(n, i);
    range.setEnd(n, i + term.length);
    const mark = document.createElement("mark");
    range.surroundContents(mark);
    mark.scrollIntoView({ block: "center" });
    setTimeout(() => mark.replaceWith(...mark.childNodes), 2400);
    return;
  }
}

function pageView(res) {
  const page = res.page;
  const { el: prose, headings } = render(page.content, { file: page.file, index: state.index });
  setTitle(page.title);
  if (!state.config.readOnly) wireTasks(prose, res);

  const meta = h("div", { class: "page-meta" },
    h("span", { class: "meta-item", title: formatDate(page.modified), html: icon("clock") }, "Edited " + timeAgo(page.modified)),
    h("a", { class: "meta-item", href: "/" + encodePath(page.file) + "?raw", target: "_blank", "data-tip": "View source", html: icon("file") }, page.file),
    h("span", { class: "spacer" }),
    state.config.readOnly ? null : h("a", { class: "meta-item", href: "#", html: icon("pencil"), onclick: (e) => { e.preventDefault(); startEdit(); } }, "Edit this page"),
  );

  const doc = h("article", { class: "doc" }, prose);
  if (res.kind === "dir" && res.children && res.children.length) {
    doc.append(h("div", { class: "section-title", text: "In this section" }), cards(res.children, res.dir));
  }
  doc.append(meta);

  const toc = tocView(headings);
  return h("div", { class: "page" + (toc ? "" : " no-toc") }, doc, toc);
}

function tocView(headings) {
  const items = headings.filter((x) => x.depth === 2 || x.depth === 3);
  if (items.length < 2) return null;
  const list = h("ol");
  for (const it of items) {
    list.append(h("li", { class: "depth-" + it.depth }, h("a", { href: "#" + it.id, "data-toc": it.id, text: it.text })));
  }
  return h("nav", { class: "toc", "aria-label": "On this page" }, h("div", { class: "toc-title", text: "On this page" }), list);
}

function updateScrollSpy() {
  const links = document.querySelectorAll(".toc a[data-toc]");
  if (!links.length) return;
  const scroller = $("#scroller");
  const top = scroller.getBoundingClientRect().top + 90;
  let active = links[0];
  for (const a of links) {
    const el = document.getElementById(a.dataset.toc);
    if (el && el.getBoundingClientRect().top <= top) active = a;
  }
  if (scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 4) active = links[links.length - 1];
  links.forEach((a) => a.classList.toggle("is-active", a === active));
}

function folderView(res) {
  const isRoot = res.path === "";
  const entry = state.index.get(res.dir);
  const title = isRoot ? state.config.title : entry ? entry.title : res.dir.split("/").pop();
  setTitle(isRoot ? "" : title);
  const n = res.children ? res.children.length : 0;
  const pages = state.index.list.filter((e) => e.type === "page" && (isRoot || e.path.startsWith(res.dir + "/"))).length;
  const canWrite = !state.config.readOnly;

  const hero = h("div", { class: "hero" },
    h("div", { class: "hero-eyebrow", text: isRoot ? "Wiki" : "Folder" }),
    h("h1", { text: isRoot ? title : title }),
    h("p", { text: isRoot
      ? (pages ? `${plural(pages, "page")} of Markdown, served straight from your folder.` : "An empty folder, ready for your first page.")
      : `${plural(n, "item")} in ${res.dir}/` }),
    canWrite ? h("div", { class: "actions" },
      h("button", { class: "btn btn-primary btn-lg", type: "button", html: icon("doc-plus"), onclick: () => newPageDialog(res.dir) }, "New Page"),
      h("button", { class: "btn btn-lg", type: "button", html: icon("pencil"), onclick: () => navigate(res.dir, { edit: true }) }, isRoot ? "Write a Home Page" : "Write an Overview"),
    ) : null,
  );
  const doc = h("div", { class: "doc" }, hero);
  if (n || canWrite) {
    doc.append(h("div", { class: "section-title", text: isRoot ? "Contents" : "In this folder" }), cards(res.children || [], res.dir));
  }
  return h("div", { class: "page no-toc" }, doc);
}

function cards(children, dir) {
  const grid = h("div", { class: "cards" });
  for (const c of children) {
    const meta = c.type === "dir" ? plural(c.count || 0, "item") : c.modified ? "Edited " + timeAgo(c.modified) : "";
    grid.append(h("a", { class: "card", href: "/" + encodePath(c.path), "data-nav": "" },
      h("span", { class: "card-icon", html: icon(c.type === "dir" ? "folder" : "doc") }),
      h("span", { class: "card-title", text: c.title }),
      h("span", { class: "card-meta", text: meta })));
  }
  if (!state.config.readOnly) {
    grid.append(h("button", { class: "card add", type: "button", html: icon("plus"), onclick: () => newPageDialog(dir) }, h("span", { text: "Add a page" })));
  }
  return grid;
}

function missingView(res) {
  setTitle("Not found");
  const name = res.path.split("/").pop() || "Home";
  const canWrite = !state.config.readOnly;
  return emptyState(
    "doc-plus",
    "This page doesn't exist yet",
    canWrite ? `Create “${name}” and it will be saved as ${res.path}.md` : `There is no page at /${res.path}.`,
    canWrite
      ? h("button", { class: "btn btn-primary btn-lg", type: "button", html: icon("pencil"), onclick: () => navigate(res.path, { edit: true, replace: true }) }, "Create Page")
      : h("a", { class: "btn btn-lg", href: "/", "data-nav": "", html: icon("home") }, "Go Home"),
  );
}

function emptyState(ic, title, message, action) {
  return h("div", { class: "empty" },
    h("div", { class: "empty-art", html: icon(ic) }),
    h("h2", { text: title }),
    h("p", { text: message }),
    action);
}

function showError(e) {
  renderToolbar({ kind: "error", path: state.path });
  const msg = e instanceof ApiError && e.status === 400 ? "That address isn't a valid page path." : e.message;
  $("#view").replaceChildren(emptyState("alert", "Something went wrong", msg, h("a", { class: "btn btn-lg", href: "/", "data-nav": "", html: icon("home") }, "Go Home")));
}

// ── Interactive task lists ────────────────────────────────────────────────
function wireTasks(prose, res) {
  const boxes = [...prose.querySelectorAll('li.task > input[type="checkbox"], li.task > p > input[type="checkbox"]')];
  if (!boxes.length) return;
  prose.classList.add("interactive");
  boxes.forEach((box, i) => {
    box.disabled = false;
    box.addEventListener("change", async () => {
      const page = res.page;
      const next = toggleTask(page.content, i, box.checked);
      if (next == null) { box.checked = !box.checked; return; }
      box.closest("li").classList.toggle("is-done", box.checked);
      try {
        const saved = await api.save(page.file, next, { baseHash: page.hash });
        res.page = saved;
      } catch (e) {
        box.checked = !box.checked;
        box.closest("li").classList.toggle("is-done", box.checked);
        toast(e.status === 409 ? "This page changed on disk — reloading" : e.message, { type: "error" });
        if (e.status === 409) refreshCurrent();
      }
    });
  });
}

// ── Toolbar ───────────────────────────────────────────────────────────────
function crumbs(path) {
  const nav = h("nav", { class: "crumbs", "aria-label": "Breadcrumbs" });
  const sep = () => h("span", { class: "sep", html: icon("chevron") });
  nav.append(h("a", { href: "/", "data-nav": "", text: state.config.title }));
  const parts = path ? path.split("/") : [];
  let acc = "";
  parts.forEach((seg, i) => {
    acc = acc ? acc + "/" + seg : seg;
    nav.append(sep());
    const entry = state.index.get(acc);
    const label = entry ? entry.title : seg;
    if (i === parts.length - 1) nav.append(h("span", { class: "here", text: label }));
    else nav.append(h("a", { href: "/" + encodePath(acc), "data-nav": "", text: label }));
  });
  return nav;
}

function sidebarButton() {
  return iconButton("sidebar", "Show sidebar", toggleSidebar, { class: "icon-btn show-sidebar", "data-kbd": "⌘\\" });
}

function renderToolbar(res) {
  const bar = $("#toolbar");
  const canWrite = !state.config.readOnly;
  const items = [sidebarButton(), crumbs(res.path || "")];
  items.push(h("span", { class: "spacer" }));
  if (canWrite && res.kind !== "error") {
    if (res.kind !== "missing") {
      items.push(h("button", { class: "btn", type: "button", html: icon("pencil"), "data-tip": "Edit", "data-kbd": "E", onclick: startEdit }, h("span", { class: "label-wide", text: "Edit" })));
    }
  }
  if (res.kind !== "error") {
    const more = iconButton("more", "More", () => menu(more, pageMenuItems(res)));
    items.push(more);
  }
  bar.replaceChildren(...items);
}

function pageMenuItems(res) {
  const canWrite = !state.config.readOnly;
  const isDir = res.kind === "dir";
  const target = isDir ? res.dir : res.page ? res.page.file : null;
  const dirOf = isDir ? res.dir : parentDir(res.path);
  return [
    { text: "New Page", icon: "doc-plus", kbd: "N", hidden: !canWrite, run: () => newPageDialog(dirOf) },
    { text: "New Folder", icon: "folder-plus", hidden: !canWrite, run: () => newFolderDialog(dirOf) },
    "-",
    { text: "Copy Link", icon: "link", run: () => copy(location.origin + "/" + encodePath(res.path), "Link copied") },
    { text: "View Source", icon: "file", hidden: !res.page, run: () => window.open("/" + encodePath(res.page.file) + "?raw", "_blank") },
    { text: "Print", icon: "print", hidden: !res.page, run: () => window.print() },
    ...(canWrite && target && res.path !== "" ? [
      "-",
      { text: isDir ? "Rename Folder…" : "Rename or Move…", icon: "rename", run: () => renameDialog(isDir ? { type: "dir", path: res.dir } : { type: "page", path: res.path, file: res.page.file }) },
      { text: isDir ? "Delete Folder…" : "Delete Page…", icon: "trash", danger: true, run: () => deleteDialog(isDir ? { type: "dir", path: res.dir, title: state.index.get(res.dir)?.title } : { type: "page", path: res.path, file: res.page.file, title: res.page.title }) },
    ] : []),
  ];
}

// ── Editing ───────────────────────────────────────────────────────────────
function startEdit() {
  if (state.config.readOnly) return;
  navigate(state.path, { edit: true });
}

function showEditor(res) {
  let file, content, hash = "", isNew = false;
  if (res.page) {
    ({ file, content, hash } = res.page);
  } else if (res.kind === "dir") {
    file = (res.dir ? res.dir + "/" : "") + "index.md";
    content = `# ${res.dir ? state.index.get(res.dir)?.title || res.dir.split("/").pop() : state.config.title}\n\n`;
    isNew = true;
  } else {
    file = res.path + ".md";
    const name = res.path.split("/").pop();
    content = `# ${name}\n\n`;
    isNew = true;
  }
  const saveBtn = h("button", { class: "btn btn-primary", type: "button", "data-tip": "Save", "data-kbd": "⌘S", html: icon("check"), onclick: () => saveEditor() }, "Save");
  const ed = createEditor({
    file, content, hash, isNew,
    config: state.config,
    index: state.index,
    onSave: () => saveEditor(),
    onDirtyChange: (d) => { saveBtn.disabled = !d; },
  });
  state.editor = ed;
  setTitle("Editing " + file.split("/").pop());

  const seg = h("div", { class: "segmented hide-narrow", role: "group", "aria-label": "Layout" });
  for (const [id, ic, label] of [["write", "write", "Write"], ["split", "columns", "Split"], ["preview", "eye", "Preview"]]) {
    const b = h("button", { type: "button", "aria-pressed": String(ed.layout === id), html: icon(ic), "data-tip": label },
      h("span", { class: "label-wide", text: label }));
    b.addEventListener("click", () => {
      ed.setLayout(id);
      seg.querySelectorAll("button").forEach((x) => x.setAttribute("aria-pressed", String(x === b)));
    });
    seg.append(b);
  }
  saveBtn.disabled = !ed.dirty;
  const done = h("button", { class: "btn btn-ghost", type: "button", "data-tip": "Stop editing", "data-kbd": "Esc", onclick: finishEditing }, "Done");
  $("#toolbar").replaceChildren(sidebarButton(), crumbs(res.path || ""), h("span", { class: "spacer" }), seg, h("span", { class: "toolbar-divider hide-narrow" }), done, saveBtn);
  $("#view").replaceChildren(ed.el);
  requestAnimationFrame(() => ed.focus());
}

async function finishEditing() {
  const ed = state.editor;
  if (!ed) return;
  if (ed.dirty && !(await leaveEditor())) return;
  state.editor = null;
  const { path } = currentLocation();
  const res = await api.page(path);
  navigate(res.kind === "missing" ? parentDir(path) : path, { replace: true });
}

let saving = null;

async function saveEditor() {
  const ed = state.editor;
  if (!ed) return false;
  if (saving) return saving;
  saving = (async () => {
    ed.setSaving();
    const isNew = !ed.baseHash;
    try {
      const page = await api.save(ed.file, ed.value, { create: isNew, baseHash: ed.baseHash });
      ed.markSaved(page);
      ownWrites.set(page.file, page.hash);
      toast(isNew ? "Page created" : "Saved");
      if (isNew) {
        const route = routeFor(page.file);
        history.replaceState(null, "", "/" + encodePath(route) + "?edit");
        state.path = route;
        await loadTree();
      }
      return true;
    } catch (e) {
      ed.restoreStatus();
      if (e.status === 409 && e.body && e.body.page) return await resolveConflict(ed, e.body.page);
      if (e.status === 409) toast(`A page already exists at ${ed.file}`, { type: "error" });
      else toast("Couldn't save: " + e.message, { type: "error" });
      return false;
    } finally {
      saving = null;
    }
  })();
  return saving;
}

async function resolveConflict(ed, theirs) {
  const choice = await dialog({
    title: "This page changed on disk",
    message: `“${ed.file}” was modified outside this editor since you opened it.`,
    icon: "alert",
    actions: [
      { label: "Cancel", value: null },
      { label: "Use Disk Version", value: "theirs" },
      { label: "Overwrite", value: "mine", primary: true },
    ],
  });
  if (choice === "theirs") {
    ed.replaceAll(theirs.content, theirs.hash);
    ed.markSaved(theirs);
    return false;
  }
  if (choice === "mine") {
    try {
      const page = await api.save(ed.file, ed.value, { baseHash: theirs.hash });
      ed.markSaved(page);
      ownWrites.set(page.file, page.hash);
      toast("Saved");
      return true;
    } catch (e) {
      toast("Couldn't save: " + e.message, { type: "error" });
    }
  }
  return false;
}

// ── Sidebar tree ──────────────────────────────────────────────────────────
function renderTree() {
  const host = $("#tree");
  const frag = document.createDocumentFragment();
  const t = state.tree;
  if (t.file) frag.append(treeNode({ type: "page", path: "", title: "Home", file: t.file, home: true }, 0));
  for (const c of t.children || []) frag.append(treeNode(c, 0));
  if (!t.file && !(t.children || []).length) {
    frag.append(h("div", { class: "tree-empty", html: state.config.readOnly ? "No pages yet." : "No pages yet.<br>Create one to get started." }));
  }
  host.replaceChildren(frag);
  highlightTree();
}

function treeNode(node, depth) {
  const isDir = node.type === "dir";
  const wrap = h("div", { class: "tree-node" + (isDir && state.open.has(node.path) ? " is-open" : ""), "data-path": node.path });
  const chevron = h("span", { class: "tree-chevron" + (isDir ? "" : " spacer"), html: icon("chevron") });
  const more = h("button", { class: "tree-more", type: "button", "aria-label": "More", tabindex: "-1", html: icon("more") });
  const row = h("a", {
    class: "tree-row", href: "/" + encodePath(node.path), "data-type": node.type, "data-nav": "",
    style: `--depth:${depth}`, draggable: state.config.readOnly || node.home ? "false" : "true", title: node.title,
  }, chevron, fromHTML(icon(node.home ? "home" : isDir ? "folder" : "doc", "ic tree-icon")), h("span", { class: "tree-label", text: node.title }), more);
  wrap.append(row);

  chevron.addEventListener("click", (e) => {
    if (!isDir) return;
    e.preventDefault();
    e.stopPropagation();
    toggleDir(node.path);
  });
  more.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    menu(more, treeMenuItems(node));
  });
  row.addEventListener("contextmenu", (e) => {
    e.preventDefault();
    menu({ x: e.clientX, y: e.clientY }, treeMenuItems(node));
  });
  if (isDir) {
    row.addEventListener("click", () => {
      if (!state.open.has(node.path)) toggleDir(node.path, true);
    });
  }

  // Drag & drop to move pages and folders.
  if (!state.config.readOnly) {
    row.addEventListener("dragstart", (e) => {
      e.dataTransfer.setData("application/x-goki", JSON.stringify({ type: node.type, path: node.path, file: node.file }));
      e.dataTransfer.effectAllowed = "move";
      row.classList.add("dragging");
    });
    row.addEventListener("dragend", () => row.classList.remove("dragging"));
    if (isDir) acceptDrops(row, node.path);
  }

  if (isDir) {
    const kids = h("div", { class: "tree-children", role: "group" });
    for (const c of node.children || []) kids.append(treeNode(c, depth + 1));
    wrap.append(kids);
  }
  return wrap;
}

function acceptDrops(el, dir) {
  let timer = 0;
  el.addEventListener("dragover", (e) => {
    if (!e.dataTransfer.types.includes("application/x-goki")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    el.classList.add("drop-target");
    if (dir && !state.open.has(dir) && !timer) timer = setTimeout(() => toggleDir(dir, true), 700);
  });
  el.addEventListener("dragleave", () => { el.classList.remove("drop-target"); clearTimeout(timer); timer = 0; });
  el.addEventListener("drop", async (e) => {
    e.preventDefault();
    el.classList.remove("drop-target");
    clearTimeout(timer); timer = 0;
    let item;
    try { item = JSON.parse(e.dataTransfer.getData("application/x-goki")); } catch { return; }
    const from = item.type === "dir" ? item.path : item.file;
    const name = from.split("/").pop();
    const to = dir ? `${dir}/${name}` : name;
    if (to === from || parentDir(from) === dir) return;
    await moveEntry(item, from, to);
  });
}

async function moveEntry(item, from, to) {
  try {
    const res = await api.move(from, to);
    toast(`Moved to ${res.path}`);
    await loadTree();
    const affected = state.path === item.path || state.path.startsWith(item.path + "/");
    if (affected) {
      const newRoute = res.route + state.path.slice(item.path.length);
      await navigate(newRoute, { replace: true });
    }
  } catch (e) {
    toast(e.message, { type: "error" });
  }
}

const rootDrop = $("#tree");
function wireRootDrop() {
  if (state.config.readOnly) return;
  rootDrop.addEventListener("dragover", (e) => {
    if (e.target === rootDrop && e.dataTransfer.types.includes("application/x-goki")) { e.preventDefault(); rootDrop.classList.add("drop-root"); }
  });
  rootDrop.addEventListener("drop", async (e) => {
    if (e.target !== rootDrop) return;
    e.preventDefault();
    let item;
    try { item = JSON.parse(e.dataTransfer.getData("application/x-goki")); } catch { return; }
    const from = item.type === "dir" ? item.path : item.file;
    if (!from.includes("/")) return;
    await moveEntry(item, from, from.split("/").pop());
  });
}

function treeMenuItems(node) {
  const canWrite = !state.config.readOnly;
  const isDir = node.type === "dir";
  const dir = isDir ? node.path : parentDir(node.path);
  return [
    { text: "Open", icon: isDir ? "folder" : "doc", run: () => navigate(node.path) },
    { text: "Edit", icon: "pencil", hidden: !canWrite, run: () => navigate(node.path, { edit: true }) },
    "-",
    { text: isDir ? "New Page in Folder" : "New Page Here", icon: "doc-plus", hidden: !canWrite, run: () => newPageDialog(dir) },
    { text: "New Folder…", icon: "folder-plus", hidden: !canWrite, run: () => newFolderDialog(dir) },
    "-",
    { text: "Copy Link", icon: "link", run: () => copy(location.origin + "/" + encodePath(node.path), "Link copied") },
    ...(canWrite && !node.home ? [
      { text: isDir ? "Rename Folder…" : "Rename or Move…", icon: "rename", run: () => renameDialog(node) },
      "-",
      { text: isDir ? "Delete Folder…" : "Delete Page…", icon: "trash", danger: true, run: () => deleteDialog(node) },
    ] : []),
  ];
}

function toggleDir(path, force) {
  const open = force ?? !state.open.has(path);
  if (open) state.open.add(path);
  else state.open.delete(path);
  save("goki.openDirs", [...state.open]);
  const el = document.querySelector(`.tree-node[data-path="${CSS.escape(path)}"]`);
  if (el) el.classList.toggle("is-open", open);
}

function highlightTree() {
  const path = state.path;
  // Expand ancestors of the current page.
  const parts = path.split("/");
  let changed = false;
  for (let i = 1; i < parts.length + (state.index.get(path)?.type === "dir" ? 1 : 0); i++) {
    const p = parts.slice(0, i).join("/");
    if (p && !state.open.has(p)) { state.open.add(p); changed = true; }
  }
  if (changed) {
    save("goki.openDirs", [...state.open]);
    for (const p of state.open) {
      const el = document.querySelector(`.tree-node[data-path="${CSS.escape(p)}"]`);
      if (el) el.classList.add("is-open");
    }
  }
  let current = null;
  for (const row of document.querySelectorAll(".tree-row")) {
    const is = row.parentElement.dataset.path === path;
    row.classList.toggle("is-current", is);
    if (is) { row.setAttribute("aria-current", "page"); current = row; }
    else row.removeAttribute("aria-current");
  }
  if (current) {
    const r = current.getBoundingClientRect();
    const t = $("#tree").getBoundingClientRect();
    if (r.top < t.top || r.bottom > t.bottom) current.scrollIntoView({ block: "nearest" });
  }
}

// ── Dialogs: create, rename, delete ───────────────────────────────────────
const badPath = (p) => !p || p.split("/").some((s) => !s || s === "." || s === ".." || s.startsWith(".")) || p.startsWith("-/") || p === "-";

async function newPageDialog(dir = "") {
  if (state.config.readOnly) return;
  const prefix = dir ? dir + "/" : "";
  const v = await dialog({
    title: "New Page",
    message: "Use slashes to create the page inside folders.",
    icon: "doc-plus",
    fields: [{ name: "path", label: "Page", value: prefix, placeholder: "folder/Page name" }],
    actions: [{ label: "Cancel", value: null }, { label: "Create", primary: true }],
    validate: ({ path }) => {
      const p = path.replace(/\.md$/i, "").replace(/^\/+/, "");
      if (!p || p.endsWith("/")) return { path: { text: "Type a page name" } };
      if (badPath(p)) return { path: { text: "Names can't be empty or start with a dot", error: true } };
      if (state.index.has(p) && state.index.get(p).type === "page") return { path: { text: "A page with this name already exists", error: true } };
      return { path: { text: "Creates " + p + ".md" } };
    },
  });
  if (!v) return;
  const p = v.path.replace(/\.md$/i, "").replace(/^\/+|\/+$/g, "");
  if (!p || badPath(p)) return;
  await navigate(p, { edit: true });
}

async function newFolderDialog(dir = "") {
  if (state.config.readOnly) return;
  const v = await dialog({
    title: "New Folder",
    message: "Folders become sections of your wiki.",
    icon: "folder-plus",
    fields: [{ name: "path", label: "Folder", value: dir ? dir + "/" : "", placeholder: "folder name" }],
    actions: [{ label: "Cancel", value: null }, { label: "Create", primary: true }],
    validate: ({ path }) => {
      const p = path.replace(/^\/+|\/+$/g, "");
      if (!p) return { path: { text: "Type a folder name" } };
      if (badPath(p)) return { path: { text: "Names can't be empty or start with a dot", error: true } };
      if (state.index.has(p)) return { path: { text: "Something with this name already exists", error: true } };
      return { path: { text: "Creates " + p + "/" } };
    },
  });
  if (!v) return;
  const p = v.path.replace(/^\/+|\/+$/g, "");
  try {
    await api.mkdir(p);
    state.open.add(p);
    await loadTree();
    toast("Folder created");
    navigate(p);
  } catch (e) {
    toast(e.message, { type: "error" });
  }
}

async function renameDialog(node) {
  const isDir = node.type === "dir";
  const from = isDir ? node.path : node.file;
  const current = isDir ? node.path : node.file.replace(/\.md$/i, "");
  const v = await dialog({
    title: isDir ? "Rename Folder" : "Rename or Move Page",
    message: "Edit the name, or change the folders to move it.",
    icon: "rename",
    fields: [{ name: "path", label: "Location", value: current }],
    actions: [{ label: "Cancel", value: null }, { label: "Rename", primary: true }],
    validate: ({ path }) => {
      const p = path.replace(/\.md$/i, "").replace(/^\/+|\/+$/g, "");
      if (!p) return { path: { text: "Type a name", error: true } };
      if (badPath(p)) return { path: { text: "Names can't be empty or start with a dot", error: true } };
      if (p !== current && state.index.has(p) && p.toLowerCase() !== current.toLowerCase()) return { path: { text: "Something with this name already exists", error: true } };
      return { path: { text: (isDir ? p + "/" : p + ".md") } };
    },
  });
  if (!v) return;
  const to = v.path.replace(/\.md$/i, "").replace(/^\/+|\/+$/g, "");
  if (to === current) return;
  await moveEntry(node, from, isDir ? to : to + ".md");
}

async function deleteDialog(node) {
  const isDir = node.type === "dir";
  let message = `“${node.title || node.path}” will be permanently deleted from disk.`;
  if (isDir) {
    const count = state.index.list.filter((e) => e.path.startsWith(node.path + "/")).length;
    message = `The folder “${node.path}” and ${plural(count, "item")} inside it will be permanently deleted from disk.`;
  }
  if (state.config.git) message += " You can recover it from git history.";
  const ok = await confirmDialog({ title: isDir ? "Delete Folder?" : "Delete Page?", message, confirm: "Delete", tone: "danger" });
  if (!ok) return;
  try {
    await api.remove(isDir ? node.path : node.file, isDir);
    toast(isDir ? "Folder deleted" : "Page deleted");
    if (state.editor && state.editor.file === node.file) state.editor = null;
    await loadTree();
    if (state.path === node.path || state.path.startsWith(node.path + "/")) navigate(parentDir(node.path), { replace: true });
    else if (state.current && state.current.kind === "dir") refreshCurrent();
  } catch (e) {
    toast(e.message, { type: "error" });
  }
}

// ── Live updates ──────────────────────────────────────────────────────────
const ownWrites = new Map();
let treeTimer = 0;

async function onServerChange(ev) {
  clearTimeout(treeTimer);
  treeTimer = setTimeout(loadTree, 120);
  const cur = state.current;
  const file = state.editor ? state.editor.file : cur && cur.page ? cur.page.file : null;
  const touches = (ev.paths || []).some((p) => p === file || (cur && cur.kind === "dir" && parentDir(p) === cur.dir) || (cur && cur.kind === "missing" && p.replace(/\.md$/i, "") === cur.path));
  if (!touches) return;
  if (state.editor) {
    const res = await api.page(state.path).catch(() => null);
    if (!res || !res.page || !state.editor) return;
    if (res.page.hash === state.editor.baseHash || ownWrites.get(res.page.file) === res.page.hash) return;
    if (!state.editor.dirty) {
      state.editor.replaceAll(res.page.content, res.page.hash);
      state.editor.markSaved(res.page);
      toast("Reloaded changes from disk", { type: "info" });
    } else {
      toast("This page changed on disk", { type: "info" });
    }
    return;
  }
  refreshCurrent();
}

async function refreshCurrent() {
  const seq = routeSeq;
  const res = await api.page(state.path).catch(() => null);
  if (!res || seq !== routeSeq || state.editor) return;
  const prev = state.current;
  if (prev && prev.page && res.page && prev.page.hash === res.page.hash && prev.kind === res.kind) return;
  res.__refresh = true;
  state.current = res;
  showView(res);
}

// ── Chrome: sidebar, resizer, global actions ──────────────────────────────
function toggleSidebar() {
  if (matchMedia("(max-width: 860px)").matches) {
    root.classList.toggle("sidebar-open");
    return;
  }
  const hidden = root.classList.toggle("sidebar-hidden");
  try { localStorage.setItem("goki.sidebarHidden", hidden ? "1" : "0"); } catch {}
}

function wireChrome() {
  document.addEventListener("click", (e) => {
    const a = e.target.closest("[data-action]");
    if (!a) return;
    const action = a.dataset.action;
    if (action === "toggle-sidebar") toggleSidebar();
    else if (action === "palette") palette();
    else if (action === "new-page") newPageDialog(currentDir());
    else if (action === "new-folder") newFolderDialog(currentDir());
    else if (action === "appearance") openAppearance(a);
  });

  const scroller = $("#scroller");
  scroller.addEventListener("scroll", () => {
    $("#toolbar").classList.toggle("is-scrolled", scroller.scrollTop > 4);
    updateScrollSpy();
  }, { passive: true });

  // Sidebar resizing.
  const handle = $("#sidebar-resizer");
  handle.addEventListener("pointerdown", (e) => {
    e.preventDefault();
    handle.setPointerCapture(e.pointerId);
    root.classList.add("is-resizing");
    const move = (ev) => {
      const w = Math.round(Math.min(480, Math.max(220, ev.clientX - parseFloat(getComputedStyle(root).getPropertyValue("--frame-gap") || 0))));
      root.style.setProperty("--sidebar-w", w + "px");
    };
    const up = () => {
      root.classList.remove("is-resizing");
      handle.removeEventListener("pointermove", move);
      try { localStorage.setItem("goki.sidebarWidth", parseInt(getComputedStyle(root).getPropertyValue("--sidebar-w"))); } catch {}
    };
    handle.addEventListener("pointermove", move);
    handle.addEventListener("pointerup", up, { once: true });
  });
  handle.addEventListener("dblclick", () => {
    root.style.removeProperty("--sidebar-w");
    try { localStorage.removeItem("goki.sidebarWidth"); } catch {}
  });
  wireRootDrop();
}

function currentDir() {
  const cur = state.current;
  if (!cur) return "";
  if (cur.kind === "dir") return cur.dir;
  return parentDir(state.path);
}

function palette(initial = "") {
  closePopover();
  if (topOverlay()) return;
  openPalette({
    canWrite: !state.config.readOnly,
    pages: () => state.index.list.map((e) => ({ path: e.path, title: e.title, file: e.file, type: e.type })),
    recent: () => state.recent,
    go: (path, q) => navigate(path, { q }),
    createPage: (q) => navigate(q.replace(/\.md$/i, "").replace(/^\/+|\/+$/g, ""), { edit: true }),
    commands,
  }, initial);
}

function commands() {
  const canWrite = !state.config.readOnly;
  const list = [
    canWrite && { title: "New Page…", icon: "doc-plus", kbd: "N", run: () => newPageDialog(currentDir()) },
    canWrite && { title: "New Folder…", icon: "folder-plus", run: () => newFolderDialog(currentDir()) },
    canWrite && state.current && state.current.kind !== "missing" && !state.edit && { title: "Edit This Page", icon: "pencil", kbd: "E", run: startEdit },
    { title: "Go Home", icon: "home", run: () => navigate("") },
    { title: "Toggle Sidebar", icon: "sidebar", kbd: mod + "\\", run: toggleSidebar },
    { title: "Appearance…", icon: "palette", run: () => openAppearance(document.querySelector('[data-action="appearance"]')) },
    { title: "Copy Link to Page", icon: "link", run: () => copy(location.href.split("?")[0], "Link copied") },
    { title: "Keyboard Shortcuts", icon: "keyboard", kbd: "?", run: shortcutsDialog },
    ...allThemes().map((t) => ({ title: `Theme: ${t.label}`, icon: t.scheme === "dark" ? "moon" : "sun", run: () => applyTheme(t.id) })),
    { title: "Theme: Automatic", icon: "sparkle", run: () => applyTheme("auto") },
  ];
  return list.filter(Boolean);
}

function shortcutsDialog() {
  const rows = [
    [mod + "K", "Search & commands"],
    ["/", "Search"],
    ["E", "Edit page"],
    ["N", "New page"],
    [mod + "S", "Save"],
    ["Esc", "Finish editing"],
    [mod + "B / " + mod + "I", "Bold / italic"],
    [mod + "E", "Inline code"],
    [mod + "⇧K", "Insert link"],
    [mod + "\\", "Toggle sidebar"],
  ];
  const table = h("div", { style: "display:grid;grid-template-columns:auto 1fr;gap:10px 18px;align-items:center;margin-bottom:20px;font-size:13.5px" });
  for (const [k, d] of rows) table.append(h("kbd", { text: k, style: "justify-self:start" }), h("span", { text: d }));
  dialog({ title: "Keyboard Shortcuts", icon: "keyboard", message: table, actions: [{ label: "Done", primary: true, value: true }] });
}

function wireKeyboard() {
  document.addEventListener("keydown", (e) => {
    const cmd = e.metaKey || e.ctrlKey;
    const typing = e.target.closest && e.target.closest("input, textarea, select, [contenteditable='true']");
    if (cmd && e.key.toLowerCase() === "k" && !e.shiftKey) { e.preventDefault(); palette(); return; }
    if (cmd && e.key.toLowerCase() === "s") {
      e.preventDefault();
      if (state.editor) saveEditor();
      return;
    }
    if (cmd && e.key === "\\") { e.preventDefault(); toggleSidebar(); return; }
    if (topOverlay()) return;
    if (e.key === "Escape" && state.editor && !e.defaultPrevented) { e.preventDefault(); finishEditing(); return; }
    if (typing || cmd || e.altKey) return;
    if (e.key === "/") { e.preventDefault(); palette(); }
    else if (e.key === "e" && !state.edit && state.current && state.current.kind !== "missing" && !state.config.readOnly) { e.preventDefault(); startEdit(); }
    else if (e.key === "n" && !state.config.readOnly) { e.preventDefault(); newPageDialog(currentDir()); }
    else if (e.key === "?") { e.preventDefault(); shortcutsDialog(); }
  });
}

function wireLinks() {
  document.addEventListener("click", (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const a = e.target.closest("a[href]");
    if (!a || a.target === "_blank") return;
    const href = a.getAttribute("href");
    if (href.startsWith("#")) {
      const el = document.getElementById(decodeURIComponent(href.slice(1)));
      if (el) {
        e.preventDefault();
        el.scrollIntoView({ block: "start" });
        history.replaceState(null, "", location.pathname + location.search + href);
      }
      return;
    }
    if (!("nav" in a.dataset) && !("internal" in a.dataset)) return;
    const url = new URL(a.href);
    if (url.origin !== location.origin) return;
    e.preventDefault();
    const path = decodeURIComponent(url.pathname).replace(/^\/+|\/+$/g, "");
    if (path === state.path && !state.edit && url.hash) {
      history.replaceState(null, "", url.pathname + url.hash);
      scrollToHash();
      return;
    }
    navigate(path, { hash: url.hash });
  });
}

// ── Utilities ─────────────────────────────────────────────────────────────
function parentDir(p) {
  const i = p.lastIndexOf("/");
  return i < 0 ? "" : p.slice(0, i);
}

function routeFor(file) {
  return file.replace(/(^|\/)(index|readme)\.md$/i, "").replace(/\.md$/i, "");
}

async function copy(text, msg) {
  try {
    await navigator.clipboard.writeText(text);
    toast(msg);
  } catch {
    toast("Couldn't copy to the clipboard", { type: "error" });
  }
}

function load(key, fallback) {
  try {
    const v = localStorage.getItem(key);
    return v ? JSON.parse(v) : fallback;
  } catch {
    return fallback;
  }
}

function save(key, value) {
  try { localStorage.setItem(key, JSON.stringify(value)); } catch {}
}

function localGet(key) {
  try { return localStorage.getItem(key); } catch { return null; }
}

boot();
