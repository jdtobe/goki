// Shared UI primitives: element builder, toasts, dialogs, menus, tooltips.
import { icon } from "./icons.js";

export const isMac = document.documentElement.classList.contains("is-mac");
export const mod = isMac ? "⌘" : "Ctrl ";

/**
 * Builds an element. attrs may include `class`, `html`, `text`, `dataset`,
 * event handlers as `onclick` etc., and any other attribute.
 */
export function h(tag, attrs = {}, ...children) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k === "class") el.className = v;
    else if (k === "html") el.innerHTML = v;
    else if (k === "text") el.textContent = v;
    else if (k === "dataset") Object.assign(el.dataset, v);
    else if (k.startsWith("on") && typeof v === "function") el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v === true ? "" : v);
  }
  for (const c of children.flat()) {
    if (c == null || c === false) continue;
    el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return el;
}

/** Creates an element from an HTML string containing a single root. */
export function fromHTML(html) {
  const t = document.createElement("template");
  t.innerHTML = html.trim();
  return t.content.firstElementChild;
}

export function esc(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
}

export function iconButton(name, tip, onclick, extra = {}) {
  return h("button", { class: "icon-btn", type: "button", "data-tip": tip, "aria-label": tip, onclick, ...extra, html: icon(name) });
}

// ── Toasts ────────────────────────────────────────────────────────────────
export function toast(message, { type = "success", action, duration = 2600 } = {}) {
  const host = document.getElementById("toasts");
  const ic = type === "error" ? "alert" : type === "info" ? "info" : "check-circle";
  const el = h("div", { class: `toast ${type}`, html: icon(ic) }, h("span", { text: message }));
  if (action) el.append(h("button", { type: "button", text: action.label, onclick: () => { action.run(); dismiss(); } }));
  host.append(el);
  let timer = setTimeout(dismiss, type === "error" ? Math.max(duration, 5000) : duration);
  el.addEventListener("mouseenter", () => clearTimeout(timer));
  el.addEventListener("mouseleave", () => (timer = setTimeout(dismiss, 1200)));
  function dismiss() {
    el.classList.add("is-leaving");
    el.addEventListener("animationend", () => el.remove(), { once: true });
  }
}

// ── Overlays ──────────────────────────────────────────────────────────────
const overlayStack = [];

export function openOverlay(content, { onClose, dismissable = true } = {}) {
  const prevFocus = document.activeElement;
  const overlay = h("div", { class: "overlay" }, content);
  const close = (value) => {
    if (!overlay.isConnected) return;
    overlay.classList.add("is-closing");
    const i = overlayStack.indexOf(entry);
    if (i >= 0) overlayStack.splice(i, 1);
    setTimeout(() => overlay.remove(), 170);
    if (prevFocus && prevFocus.focus) prevFocus.focus({ preventScroll: true });
    onClose && onClose(value);
  };
  const entry = { close, dismissable };
  overlay.addEventListener("mousedown", (e) => {
    if (e.target === overlay && dismissable) close(null);
  });
  overlayStack.push(entry);
  document.body.append(overlay);
  return { overlay, close };
}

export function topOverlay() {
  return overlayStack[overlayStack.length - 1];
}

document.addEventListener(
  "keydown",
  (e) => {
    if (e.key === "Escape" && overlayStack.length) {
      const top = topOverlay();
      if (top.dismissable) {
        e.preventDefault();
        e.stopImmediatePropagation();
        top.close(null);
      }
    }
  },
  true,
);

/**
 * Shows a modal dialog. Returns a promise resolving to the clicked action's
 * value (or the field values for prompts), or null when dismissed.
 */
export function dialog({ title, message, icon: ic = "info", tone = "", fields = [], actions = [], validate }) {
  return new Promise((resolve) => {
    const inputs = {};
    const hints = {};
    const form = h("form", { class: "dialog", role: "dialog", "aria-modal": "true", "aria-label": title });
    form.append(h("div", { class: `dialog-icon ${tone}`, html: icon(ic) }), h("h2", { text: title }));
    if (message) form.append(typeof message === "string" ? h("p", { text: message }) : message);
    if (fields.length) {
      const wrap = h("div", { class: "fields" });
      for (const f of fields) {
        const input = h("input", {
          class: "input", name: f.name, value: f.value || "", placeholder: f.placeholder || "",
          autocomplete: "off", spellcheck: "false", "aria-label": f.label,
        });
        const hint = h("div", { class: "hint" });
        inputs[f.name] = input;
        hints[f.name] = hint;
        wrap.append(h("div", { class: "field" }, h("label", { text: f.label }), input, hint));
        input.addEventListener("input", update);
      }
      form.append(wrap);
    }
    const btns = h("div", { class: "dialog-actions" });
    let primary = null;
    for (const a of actions) {
      const b = h("button", {
        class: `btn ${a.primary ? (tone === "danger" ? "btn-danger" : "btn-primary") : ""}`,
        type: a.primary ? "submit" : "button",
        text: a.label,
      });
      if (a.primary) primary = { button: b, value: a.value };
      else b.addEventListener("click", () => close(a.value ?? null));
      btns.append(b);
    }
    form.append(btns);

    function values() {
      const v = {};
      for (const [k, el] of Object.entries(inputs)) v[k] = el.value.trim();
      return v;
    }
    function update() {
      if (!validate) return true;
      const res = validate(values()) || {};
      let ok = true;
      for (const [k, el] of Object.entries(hints)) {
        const r = res[k];
        el.textContent = r ? r.text || "" : "";
        el.classList.toggle("error", !!(r && r.error));
        if (r && r.error) ok = false;
      }
      if (primary) primary.button.disabled = !ok;
      return ok;
    }
    form.addEventListener("submit", (e) => {
      e.preventDefault();
      if (!update()) return;
      close(fields.length ? values() : primary ? primary.value : true);
    });

    const { close } = openOverlay(form, { onClose: (v) => resolve(v) });
    update();
    requestAnimationFrame(() => {
      const first = Object.values(inputs)[0];
      if (first) {
        first.focus();
        const dot = first.value.lastIndexOf(".");
        const slash = first.value.lastIndexOf("/");
        first.setSelectionRange(slash + 1, dot > slash ? dot : first.value.length);
      } else if (primary) primary.button.focus();
    });
  });
}

export function confirmDialog({ title, message, confirm = "OK", tone = "", icon: ic }) {
  return dialog({
    title, message, tone, icon: ic || (tone === "danger" ? "trash" : "info"),
    actions: [{ label: "Cancel", value: false }, { label: confirm, value: true, primary: true }],
  }).then((v) => v === true);
}

// ── Menus ─────────────────────────────────────────────────────────────────
let openMenu = null;

/** Shows a context menu at an anchor element or {x, y}. */
export function menu(anchor, items) {
  closeMenu();
  const el = h("div", { class: "menu", role: "menu" });
  for (const it of items) {
    if (it === "-") { el.append(h("div", { class: "menu-sep" })); continue; }
    if (it.label && !it.run) { el.append(h("div", { class: "menu-label", text: it.label })); continue; }
    if (it.hidden) continue;
    const b = h("button", { class: `menu-item ${it.danger ? "danger" : ""}`, role: "menuitem", type: "button", html: icon(it.icon || "doc") },
      h("span", { text: it.text }), it.kbd ? h("kbd", { text: it.kbd }) : null);
    b.addEventListener("click", () => { closeMenu(); it.run(); });
    el.append(b);
  }
  document.body.append(el);
  const r = anchor instanceof Element ? anchor.getBoundingClientRect() : { left: anchor.x, right: anchor.x, top: anchor.y, bottom: anchor.y };
  const mw = el.offsetWidth, mh = el.offsetHeight;
  let x = anchor instanceof Element ? r.right - mw : r.left;
  let y = r.bottom + 6;
  if (x < 8) x = Math.min(r.left, innerWidth - mw - 8);
  if (x + mw > innerWidth - 8) x = innerWidth - mw - 8;
  if (y + mh > innerHeight - 8) { y = r.top - mh - 6; el.style.transformOrigin = "bottom right"; }
  el.style.left = Math.max(8, x) + "px";
  el.style.top = Math.max(8, y) + "px";
  if (anchor instanceof Element) anchor.setAttribute("aria-expanded", "true");
  openMenu = { el, anchor };
  el.querySelector(".menu-item")?.focus({ preventScroll: true });
  el.addEventListener("keydown", (e) => {
    const list = [...el.querySelectorAll(".menu-item")];
    const i = list.indexOf(document.activeElement);
    if (e.key === "ArrowDown") { e.preventDefault(); list[(i + 1) % list.length].focus(); }
    if (e.key === "ArrowUp") { e.preventDefault(); list[(i - 1 + list.length) % list.length].focus(); }
    if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); closeMenu(); }
  });
}

export function closeMenu() {
  if (!openMenu) return;
  openMenu.el.remove();
  if (openMenu.anchor instanceof Element) openMenu.anchor.removeAttribute("aria-expanded");
  openMenu = null;
}

document.addEventListener("mousedown", (e) => {
  if (openMenu && !openMenu.el.contains(e.target)) closeMenu();
});
window.addEventListener("blur", closeMenu);
window.addEventListener("resize", closeMenu);

// ── Popovers ──────────────────────────────────────────────────────────────
let openPop = null;

export function popover(anchor, content, { placement = "top-start" } = {}) {
  closePopover();
  const el = h("div", { class: "popover" }, content);
  document.body.append(el);
  const r = anchor.getBoundingClientRect();
  const pw = el.offsetWidth, ph = el.offsetHeight;
  let x = r.left, y = placement.startsWith("top") ? r.top - ph - 10 : r.bottom + 10;
  if (y < 8) y = r.bottom + 10;
  if (x + pw > innerWidth - 8) x = innerWidth - pw - 8;
  el.style.left = Math.max(8, x) + "px";
  el.style.top = Math.max(8, Math.min(y, innerHeight - ph - 8)) + "px";
  openPop = { el, anchor };
  return el;
}

export function closePopover() {
  if (openPop) { openPop.el.remove(); openPop = null; }
}

document.addEventListener("mousedown", (e) => {
  if (openPop && !openPop.el.contains(e.target) && !openPop.anchor.contains(e.target)) closePopover();
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && openPop) closePopover();
});

// ── Tooltips ──────────────────────────────────────────────────────────────
let tipEl = null, tipTimer = 0;

document.addEventListener("pointerover", (e) => {
  const t = e.target.closest && e.target.closest("[data-tip]");
  if (!t || e.pointerType === "touch") return;
  clearTimeout(tipTimer);
  tipTimer = setTimeout(() => showTip(t), tipEl ? 0 : 550);
});
document.addEventListener("pointerout", (e) => {
  const t = e.target.closest && e.target.closest("[data-tip]");
  if (!t || (e.relatedTarget && t.contains(e.relatedTarget))) return;
  clearTimeout(tipTimer);
  hideTip();
});
document.addEventListener("pointerdown", () => { clearTimeout(tipTimer); hideTip(); });

function showTip(target) {
  hideTip();
  if (!target.isConnected) return;
  const kbd = target.dataset.kbd ? target.dataset.kbd.replace("⌘", mod) : "";
  tipEl = h("div", { class: "tooltip" }, target.dataset.tip, kbd ? h("kbd", { text: kbd }) : null);
  document.body.append(tipEl);
  const r = target.getBoundingClientRect();
  const w = tipEl.offsetWidth;
  let x = r.left + r.width / 2 - w / 2;
  x = Math.max(8, Math.min(x, innerWidth - w - 8));
  let y = r.bottom + 8;
  if (y + 30 > innerHeight) y = r.top - 34;
  tipEl.style.left = x + "px";
  tipEl.style.top = y + "px";
}

function hideTip() {
  if (tipEl) { tipEl.remove(); tipEl = null; }
}

// ── Formatting helpers ────────────────────────────────────────────────────
const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

export function timeAgo(date) {
  const d = new Date(date);
  const s = (d.getTime() - Date.now()) / 1000;
  const abs = Math.abs(s);
  if (abs < 45) return "just now";
  const units = [["minute", 60], ["hour", 3600], ["day", 86400], ["week", 604800], ["month", 2629800], ["year", 31557600]];
  let unit = "minute", div = 60;
  for (const [u, n] of units) if (abs >= n * 0.9) { unit = u; div = n; }
  return rtf.format(Math.round(s / div), unit);
}

export function formatDate(date) {
  return new Date(date).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

export function plural(n, word) {
  return `${n} ${word}${n === 1 ? "" : "s"}`;
}
