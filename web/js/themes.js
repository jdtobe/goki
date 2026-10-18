// Theme selection, persistence and the Appearance popover.
import { h, popover } from "./ui.js";

export const BUILTIN = [
  { id: "daylight", label: "Daylight", scheme: "light" },
  { id: "graphite", label: "Graphite", scheme: "dark" },
  { id: "glass", label: "Glass", scheme: "light" },
  { id: "nebula", label: "Nebula", scheme: "dark" },
  { id: "aurora", label: "Aurora", scheme: "dark" },
  { id: "dune", label: "Dune", scheme: "light" },
  { id: "ink", label: "Ink", scheme: "light" },
];

const ACCENTS = ["#0a84ff", "#bf5af2", "#ff375f", "#ff9f0a", "#30d158", "#64d2ff"];

const root = document.documentElement;
const darkQuery = matchMedia("(prefers-color-scheme: dark)");
const listeners = new Set();

let custom = [];
let defaultTheme = root.dataset.themePref || "auto";
let configAccent = "";

export function allThemes() {
  return [...BUILTIN, ...custom];
}

export function initThemes(cfg) {
  custom = cfg.themes || [];
  defaultTheme = cfg.theme || "auto";
  configAccent = cfg.accent || "";
  if (configAccent && !stored("goki.accent")) paintAccent(configAccent);
  darkQuery.addEventListener("change", () => {
    if (choice() === "auto") apply("auto", false);
  });
}

export function choice() {
  return stored("goki.theme") || defaultTheme;
}

export function current() {
  return root.dataset.theme;
}

export function onThemeChange(fn) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/** Applies a theme id (or "auto"); persist=false only updates the page. */
export function apply(id, persist = true) {
  const resolved = window.__gokiResolveTheme ? window.__gokiResolveTheme(id) : id;
  const valid = allThemes().some((t) => t.id === resolved);
  const finalId = valid ? resolved : darkQuery.matches ? "graphite" : "daylight";
  // Cross-fade between themes where supported.
  const swap = () => {
    root.dataset.theme = finalId;
    root.dataset.themeChoice = id;
  };
  if (document.startViewTransition && persist && !matchMedia("(prefers-reduced-motion: reduce)").matches) {
    document.startViewTransition(swap);
  } else {
    swap();
  }
  if (persist) store("goki.theme", id);
  listeners.forEach((fn) => fn(finalId));
}

export function setAccent(color) {
  store("goki.accent", color || null);
  paintAccent(color || configAccent);
  listeners.forEach((fn) => fn(current()));
}

const ACCENT_PROPS = ["--accent", "--accent-hover", "--link"];

function paintAccent(color) {
  for (const k of ACCENT_PROPS) {
    if (color) root.style.setProperty(k, color);
    else root.style.removeProperty(k);
  }
}

/** Opens the Appearance popover anchored to an element. */
export function openAppearance(anchor) {
  const grid = h("div", { class: "swatches", role: "radiogroup", "aria-label": "Theme" });
  const sel = choice();
  const make = (t, auto = false) => {
    const preview = auto
      ? h("div", { class: "swatch-preview" },
          h("div", { "data-theme": "daylight", style: "clip-path: polygon(0 0, 100% 0, 0 100%)" }, miniUI()),
          h("div", { "data-theme": "graphite", style: "clip-path: polygon(100% 0, 100% 100%, 0 100%)" }, miniUI()))
      : h("div", { class: "swatch-preview", "data-theme": t.id }, ...miniUI().childNodes);
    const b = h("button", {
      class: "swatch" + (auto ? " auto" : ""), type: "button", role: "radio",
      "aria-checked": String(sel === t.id),
      onclick: () => {
        apply(t.id);
        grid.querySelectorAll(".swatch").forEach((s) => s.setAttribute("aria-checked", String(s === b)));
      },
    }, preview, h("span", { class: "swatch-name" }, t.label, h("small", { text: auto ? "System" : t.scheme === "dark" ? "Dark" : "Light" })));
    return b;
  };
  grid.append(make({ id: "auto", label: "Automatic" }, true));
  for (const t of allThemes()) grid.append(make(t));
  grid.style.setProperty("--sel", "var(--accent)");

  const accentWrap = h("div", { class: "accents", role: "radiogroup", "aria-label": "Accent color" });
  const curAccent = stored("goki.accent");
  const dot = (c) => {
    const b = h("button", {
      class: "accent-dot" + (c ? "" : " reset"), type: "button", role: "radio",
      style: c ? `--c:${c}` : "", "aria-checked": String((curAccent || "") === (c || "")),
      "data-tip": c ? c : "Theme default",
      onclick: () => {
        setAccent(c);
        accentWrap.querySelectorAll(".accent-dot").forEach((d) => d.setAttribute("aria-checked", String(d === b)));
      },
    });
    return b;
  };
  accentWrap.append(dot(""), ...ACCENTS.map(dot));

  const content = h("div", { class: "appearance" },
    h("h3", {}, "Appearance"),
    grid,
    h("div", { class: "appearance-row" }, h("span", { text: "Accent" }), accentWrap),
  );
  popover(anchor, content);
}

function miniUI() {
  const wrap = h("div");
  wrap.append(
    h("div", { class: "sp-side" }, h("i"), h("i"), h("i"), h("i")),
    h("div", { class: "sp-main" }, h("i"), h("i"), h("i"), h("b")),
  );
  return wrap;
}

function stored(k) {
  try { return localStorage.getItem(k); } catch { return null; }
}

function store(k, v) {
  try {
    if (v == null) localStorage.removeItem(k);
    else localStorage.setItem(k, v);
  } catch {}
}
