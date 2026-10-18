// The Markdown editor: a fast plain-text source pane with formatting helpers,
// list continuation, image paste/drop upload and a live preview.
import { api } from "./api.js";
import { icon } from "./icons.js";
import { render, enhance } from "./markdown.js";
import { h, iconButton, mod, toast, plural } from "./ui.js";

const LAYOUT_KEY = "goki.editorLayout";
const draftKey = (file) => "goki.draft:" + file;

export function createEditor({ file, content, hash, isNew, config, index, onSave, onDirtyChange }) {
  let saved = content;
  let baseHash = hash || "";
  let layout = safeGet(LAYOUT_KEY) || config.editor.layout || "split";
  if (matchMedia("(max-width: 860px)").matches && layout === "split") layout = "write";

  const source = h("textarea", {
    class: "source" + (config.editor.lineWrap ? "" : " nowrap"),
    spellcheck: config.editor.spellcheck ? "true" : "false",
    placeholder: "Start writing…",
    "aria-label": "Markdown source",
    autocapitalize: "sentences",
  });
  source.value = content;
  source.style.tabSize = config.editor.tabSize;

  const previewDoc = h("div", { class: "doc" });
  const paneSource = h("div", { class: "pane pane-source" }, source);
  const panePreview = h("div", { class: "pane pane-preview" }, previewDoc);
  const statusDot = h("span", { class: "status-dot" });
  const statusText = h("span", { text: isNew ? "New page" : "Saved" });
  const counts = h("span");
  const banner = h("div");

  const fmt = (name, tip, run, kbd) => iconButton(name, tip, (e) => { e.preventDefault(); run(); source.focus(); }, kbd ? { "data-kbd": kbd } : {});
  const formatbar = h("div", { class: "formatbar" },
    fmt("heading", "Heading", () => toggleLinePrefix("## ")),
    fmt("bold", "Bold", () => wrap("**"), "⌘B"),
    fmt("italic", "Italic", () => wrap("_"), "⌘I"),
    fmt("strike", "Strikethrough", () => wrap("~~")),
    fmt("code", "Code", codeAction, "⌘E"),
    h("span", { class: "sep" }),
    fmt("link", "Link", linkAction, "⌘⇧K"),
    fmt("image", "Insert image", pickImage),
    fmt("table", "Table", () => insertBlock("| Column | Column |\n| ------ | ------ |\n| Cell   | Cell   |")),
    h("span", { class: "sep" }),
    fmt("list", "Bulleted list", () => toggleLinePrefix("- ")),
    fmt("list-ordered", "Numbered list", () => toggleLinePrefix("1. ")),
    fmt("checklist", "Checklist", () => toggleLinePrefix("- [ ] ")),
    fmt("quote", "Quote", () => toggleLinePrefix("> ")),
    h("span", { class: "editor-title", text: file }),
  );

  const el = h("div", { class: "editor", "data-layout": layout },
    formatbar, banner,
    h("div", { class: "editor-panes" }, paneSource, panePreview),
    h("div", { class: "statusbar" },
      h("span", {}, statusDot, statusText),
      counts,
      h("span", { class: "spacer" }),
      h("span", { class: "hide-narrow", html: `Markdown · <kbd>${mod}S</kbd> save · <kbd>Esc</kbd> done` }),
    ),
  );

  // ── State ───────────────────────────────────────────────────────────────
  let dirty = false;
  const setDirty = (d) => {
    if (d === dirty) return;
    dirty = d;
    statusDot.className = "status-dot" + (d ? " dirty" : "");
    statusText.textContent = d ? "Edited" : "Saved";
    onDirtyChange && onDirtyChange(d);
  };
  if (isNew) setDirty(true);

  let previewTimer = 0, draftTimer = 0;
  const onInput = () => {
    setDirty(isNew || source.value !== saved);
    clearTimeout(previewTimer);
    previewTimer = setTimeout(updatePreview, 120);
    clearTimeout(draftTimer);
    draftTimer = setTimeout(() => {
      if (source.value !== saved) safeSet(draftKey(file), JSON.stringify({ content: source.value, base: baseHash, at: Date.now() }));
      else safeDel(draftKey(file));
    }, 400);
    updateCounts();
    autosize();
  };
  source.addEventListener("input", onInput);

  // The textarea grows with its content so the pane is the only scroller,
  // which keeps scroll sync and caret visibility simple.
  function autosize() {
    const top = paneSource.scrollTop;
    source.style.height = "auto";
    source.style.height = source.scrollHeight + "px";
    paneSource.scrollTop = top;
  }
  new ResizeObserver(() => autosize()).observe(paneSource);

  let lastEnhance = 0;
  function updatePreview() {
    if (layout === "write") return;
    const { el: rendered } = render(source.value, { file, index });
    previewDoc.replaceChildren(rendered);
    clearTimeout(lastEnhance);
    lastEnhance = setTimeout(() => enhance(rendered), 350);
  }

  function updateCounts() {
    const text = source.value;
    const words = (text.match(/[\p{L}\p{N}][\p{L}\p{N}'’_-]*/gu) || []).length;
    const mins = Math.max(1, Math.round(words / 230));
    counts.textContent = `${plural(words, "word")} · ${mins} min read`;
  }

  // Proportional scroll sync from source to preview.
  paneSource.addEventListener("scroll", () => {
    if (layout !== "split") return;
    const max = paneSource.scrollHeight - paneSource.clientHeight;
    const ratio = max > 0 ? paneSource.scrollTop / max : 0;
    panePreview.scrollTop = ratio * (panePreview.scrollHeight - panePreview.clientHeight);
  }, { passive: true });

  // ── Text helpers (execCommand keeps native undo working) ────────────────
  function replaceRange(start, end, text, selStart, selEnd) {
    source.focus();
    source.setSelectionRange(start, end);
    if (!document.execCommand("insertText", false, text)) {
      source.setRangeText(text, start, end, "end");
      source.dispatchEvent(new Event("input"));
    }
    if (selStart != null) source.setSelectionRange(selStart, selEnd ?? selStart);
  }

  function wrap(marker, placeholder = "text") {
    const { selectionStart: s, selectionEnd: e, value } = source;
    const sel = value.slice(s, e);
    if (sel.startsWith(marker) && sel.endsWith(marker) && sel.length >= marker.length * 2) {
      const inner = sel.slice(marker.length, sel.length - marker.length);
      return replaceRange(s, e, inner, s, s + inner.length);
    }
    if (value.slice(s - marker.length, s) === marker && value.slice(e, e + marker.length) === marker) {
      return replaceRange(s - marker.length, e + marker.length, sel, s - marker.length, e - marker.length);
    }
    const inner = sel || placeholder;
    replaceRange(s, e, marker + inner + marker, s + marker.length, s + marker.length + inner.length);
  }

  function codeAction() {
    const { selectionStart: s, selectionEnd: e, value } = source;
    if (value.slice(s, e).includes("\n")) insertBlock("```\n" + value.slice(s, e) + "\n```", true);
    else wrap("`", "code");
  }

  function linkAction() {
    const { selectionStart: s, selectionEnd: e, value } = source;
    const sel = value.slice(s, e);
    if (/^https?:\/\//.test(sel)) return replaceRange(s, e, `[link](${sel})`, s + 1, s + 5);
    const text = sel || "link text";
    replaceRange(s, e, `[${text}](url)`, s + text.length + 3, s + text.length + 6);
  }

  function lineBounds() {
    const { selectionStart: s, selectionEnd: e, value } = source;
    const start = value.lastIndexOf("\n", s - 1) + 1;
    let end = value.indexOf("\n", e === s ? e : e - 1);
    if (end < 0) end = value.length;
    return [start, end];
  }

  function toggleLinePrefix(prefix) {
    const [start, end] = lineBounds();
    const lines = source.value.slice(start, end).split("\n");
    const strip = /^(\s*)(#{1,6} |[-*+] \[[ xX]\] |[-*+] |\d+[.)] |> )/;
    const all = lines.every((l) => l.trimStart().startsWith(prefix.trim()) || !l.trim());
    let n = 0;
    const out = lines.map((l) => {
      if (!l.trim()) return l;
      const m = strip.exec(l);
      const bare = m ? m[1] + l.slice(m[0].length) : l;
      if (all) return bare;
      n++;
      const p = prefix === "1. " ? `${n}. ` : prefix;
      return bare.replace(/^(\s*)/, `$1${p}`);
    });
    const text = out.join("\n");
    replaceRange(start, end, text, start + text.length);
  }

  function insertBlock(block, select) {
    const { selectionStart: s, selectionEnd: e, value } = source;
    const before = value.slice(0, s);
    const pre = !before || before.endsWith("\n\n") ? "" : before.endsWith("\n") ? "\n" : "\n\n";
    const text = pre + block + "\n";
    replaceRange(s, e, text, s + text.length);
    if (select) source.setSelectionRange(s + pre.length, s + pre.length + block.length);
  }

  // ── Keyboard ────────────────────────────────────────────────────────────
  source.addEventListener("keydown", (e) => {
    const cmd = e.metaKey || e.ctrlKey;
    if (cmd && !e.shiftKey && !e.altKey) {
      const k = e.key.toLowerCase();
      if (k === "b") { e.preventDefault(); wrap("**"); return; }
      if (k === "i") { e.preventDefault(); wrap("_"); return; }
      if (k === "e") { e.preventDefault(); codeAction(); return; }
    }
    if (cmd && e.shiftKey && e.key.toLowerCase() === "k") { e.preventDefault(); linkAction(); return; }
    if (e.key === "Tab" && !cmd && !e.altKey) {
      e.preventDefault();
      indent(e.shiftKey);
      return;
    }
    if (e.key === "Enter" && !e.shiftKey && !cmd && !e.isComposing) continueList(e);
  });

  function indent(outdent) {
    const unit = " ".repeat(config.editor.tabSize);
    const { selectionStart: s, selectionEnd: e } = source;
    if (s === e && !outdent) {
      const [ls] = lineBounds();
      const line = source.value.slice(ls, source.value.indexOf("\n", ls) >>> 0);
      if (!/^\s*([-*+]|\d+[.)])\s/.test(line)) return replaceRange(s, e, unit, s + unit.length);
    }
    const [start, end] = lineBounds();
    const lines = source.value.slice(start, end).split("\n");
    const out = lines.map((l) => (outdent ? l.replace(new RegExp(`^( {1,${unit.length}}|\\t)`), "") : unit + l));
    const text = out.join("\n");
    const delta = out[0].length - lines[0].length;
    replaceRange(start, end, text, Math.max(start, s + delta), start + text.length - (end - e));
  }

  function continueList(e) {
    const { selectionStart: s, selectionEnd: end, value } = source;
    if (s !== end) return;
    const ls = value.lastIndexOf("\n", s - 1) + 1;
    const line = value.slice(ls, s);
    const m = /^(\s*(?:>\s*)*)([-*+]|(\d+)([.)]))(\s+)(\[[ xX]\]\s+)?/.exec(line);
    if (!m) return;
    e.preventDefault();
    if (line.trim() === m[0].trim()) {
      // Empty item: end the list.
      return replaceRange(ls, s, "", ls);
    }
    const bullet = m[3] ? `${+m[3] + 1}${m[4]}` : m[2];
    const next = `\n${m[1]}${bullet}${m[5]}${m[6] ? "[ ] " : ""}`;
    replaceRange(s, s, next, s + next.length);
  }

  // ── Uploads: paste, drop and picker ─────────────────────────────────────
  const pageDir = file.includes("/") ? file.slice(0, file.lastIndexOf("/")) : "";

  async function uploadFiles(files) {
    if (!config.features.upload) return;
    for (const f of files) {
      const max = config.maxUploadMB * 1024 * 1024;
      if (f.size > max) { toast(`${f.name} is larger than ${config.maxUploadMB} MB`, { type: "error" }); continue; }
      const placeholder = `![Uploading ${f.name || "image"}…]()`;
      const pos = source.selectionStart;
      replaceRange(pos, source.selectionEnd, placeholder, pos + placeholder.length);
      try {
        const res = await api.upload(pageDir, f);
        const rel = pageDir ? res.path.slice(pageDir.length + 1) : res.path;
        const url = rel.split("/").map(encodeURIComponent).join("/");
        const alt = res.name.replace(/\.[^.]+$/, "").replace(/[-_]+/g, " ");
        const md = f.type.startsWith("image/") ? `![${alt}](${url})` : `[${res.name}](${url})`;
        const at = source.value.indexOf(placeholder);
        if (at >= 0) replaceRange(at, at + placeholder.length, md, at + md.length);
      } catch (err) {
        const at = source.value.indexOf(placeholder);
        if (at >= 0) replaceRange(at, at + placeholder.length, "", at);
        toast(err.message || "Upload failed", { type: "error" });
      }
    }
  }

  source.addEventListener("paste", (e) => {
    const files = [...(e.clipboardData?.files || [])];
    if (files.length) { e.preventDefault(); uploadFiles(files); }
  });
  el.addEventListener("dragover", (e) => {
    if ([...e.dataTransfer.types].includes("Files")) { e.preventDefault(); el.classList.add("is-dragover"); }
  });
  el.addEventListener("dragleave", (e) => { if (!el.contains(e.relatedTarget)) el.classList.remove("is-dragover"); });
  el.addEventListener("drop", (e) => {
    el.classList.remove("is-dragover");
    const files = [...(e.dataTransfer?.files || [])];
    if (!files.length) return;
    e.preventDefault();
    const pos = source.selectionStart;
    source.focus();
    source.setSelectionRange(pos, pos);
    uploadFiles(files);
  });

  function pickImage() {
    const input = h("input", { type: "file", accept: "image/*,video/*,application/pdf", multiple: true });
    input.addEventListener("change", () => uploadFiles([...input.files]));
    input.click();
  }

  // ── Draft recovery ──────────────────────────────────────────────────────
  const draft = safeJSON(safeGet(draftKey(file)));
  if (draft && draft.content !== content && draft.content.trim()) {
    const when = new Date(draft.at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
    const b = h("div", { class: "banner", html: icon("clock") },
      h("span", { text: `You have unsaved changes from ${when}.` }),
      h("span", { class: "spacer" }),
      h("button", { class: "btn btn-ghost", type: "button", text: "Discard", onclick: () => { safeDel(draftKey(file)); b.remove(); } }),
      h("button", { class: "btn btn-primary", type: "button", text: "Restore", onclick: () => { replaceRange(0, source.value.length, draft.content, 0); b.remove(); } }),
    );
    banner.replaceWith(b);
  }

  // ── Public interface ────────────────────────────────────────────────────
  const api_ = {
    el,
    get value() { return source.value; },
    get dirty() { return dirty; },
    get baseHash() { return baseHash; },
    get layout() { return layout; },
    file,
    setLayout(l) {
      layout = l;
      el.dataset.layout = l;
      safeSet(LAYOUT_KEY, l);
      if (l !== "write") updatePreview();
      if (l !== "preview") source.focus();
    },
    focus() {
      autosize();
      source.focus({ preventScroll: true });
      if (isNew) {
        // Place the caret after the generated heading.
        const at = source.value.indexOf("\n\n");
        const p = at >= 0 ? at + 2 : source.value.length;
        source.setSelectionRange(p, p);
      } else {
        source.setSelectionRange(0, 0);
        paneSource.scrollTop = 0;
        source.scrollTop = 0;
      }
    },
    markSaved(page) {
      saved = page.content;
      baseHash = page.hash;
      isNew = false;
      safeDel(draftKey(file));
      dirty = source.value !== saved;
      api_.restoreStatus();
      onDirtyChange && onDirtyChange(dirty);
    },
    setSaving() {
      statusDot.className = "status-dot saving";
      statusText.textContent = "Saving…";
    },
    restoreStatus() {
      statusDot.className = "status-dot" + (dirty ? " dirty" : "");
      statusText.textContent = dirty ? "Edited" : "Saved";
    },
    replaceAll(text, hash) {
      replaceRange(0, source.value.length, text, 0);
      if (hash) baseHash = hash;
    },
    discardDraft() { safeDel(draftKey(file)); },
    rerender: updatePreview,
    save: () => onSave(api_),
  };
  updateCounts();
  updatePreview();
  return api_;
}

function safeGet(k) { try { return localStorage.getItem(k); } catch { return null; } }
function safeSet(k, v) { try { localStorage.setItem(k, v); } catch {} }
function safeDel(k) { try { localStorage.removeItem(k); } catch {} }
function safeJSON(s) { try { return s ? JSON.parse(s) : null; } catch { return null; } }
