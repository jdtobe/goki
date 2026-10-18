// A small, consistent line-icon set (24×24, 1.7px strokes, round joins).
const P = {
  sidebar: '<rect x="3" y="4.5" width="18" height="15" rx="3.5"/><path d="M9.5 4.5v15M5.5 8.5h1.5M5.5 11.5h1.5"/>',
  search: '<circle cx="10.75" cy="10.75" r="6.25"/><path d="m15.5 15.5 4.5 4.5"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  folder: '<path d="M3.5 7.5a2 2 0 0 1 2-2h3.6a2 2 0 0 1 1.5.7l1 1.1a2 2 0 0 0 1.5.7h5.4a2 2 0 0 1 2 2v7.5a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2z"/>',
  "folder-plus": '<path d="M3.5 7.5a2 2 0 0 1 2-2h3.6a2 2 0 0 1 1.5.7l1 1.1a2 2 0 0 0 1.5.7h5.4a2 2 0 0 1 2 2v7.5a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2z"/><path d="M12 11v5M9.5 13.5h5"/>',
  doc: '<path d="M7 3.5h6.5L18.5 8.5v10a2 2 0 0 1-2 2h-9.5a2 2 0 0 1-2-2v-13a2 2 0 0 1 2-2z"/><path d="M13.5 3.5v3a2 2 0 0 0 2 2h3M8.5 12.5h7M8.5 16h5"/>',
  "doc-plus": '<path d="M13.5 3.5H7a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h9.5a2 2 0 0 0 2-2v-10z"/><path d="M13.5 3.5v3a2 2 0 0 0 2 2h3M11.75 11v6M8.75 14h6"/>',
  home: '<path d="M4 10.5 12 4l8 6.5V18a2 2 0 0 1-2 2h-3.5v-5.5h-5V20H6a2 2 0 0 1-2-2z"/>',
  chevron: '<path d="m9.5 6 6 6-6 6"/>',
  "chevron-down": '<path d="m6 9.5 6 6 6-6"/>',
  pencil: '<path d="M14.5 5.5 18.5 9.5M4.5 19.5l1-4.5L16 4.5a2.1 2.1 0 0 1 3 0l.5.5a2.1 2.1 0 0 1 0 3L9 18.5z"/>',
  trash: '<path d="M4.5 7h15M9.5 7V5.5a1.5 1.5 0 0 1 1.5-1.5h2a1.5 1.5 0 0 1 1.5 1.5V7M6.5 7l.8 11.2a2 2 0 0 0 2 1.8h5.4a2 2 0 0 0 2-1.8L17.5 7M10 11v5.5M14 11v5.5"/>',
  more: '<circle cx="6" cy="12" r="1.3" fill="currentColor"/><circle cx="12" cy="12" r="1.3" fill="currentColor"/><circle cx="18" cy="12" r="1.3" fill="currentColor"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7.5"/>',
  "check-circle": '<circle cx="12" cy="12" r="8.5"/><path d="m8.5 12.3 2.5 2.5 4.8-5"/>',
  x: '<path d="M6.5 6.5l11 11M17.5 6.5l-11 11"/>',
  eye: '<path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z"/><circle cx="12" cy="12" r="3"/>',
  columns: '<rect x="3.5" y="4.5" width="17" height="15" rx="3"/><path d="M12 4.5v15"/>',
  write: '<path d="M4 6.5h16M4 11h16M4 15.5h10M4 20h6"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.7.3l3-3a4 4 0 0 0-5.7-5.7l-1.2 1.2"/><path d="M14 10a4 4 0 0 0-5.7-.3l-3 3a4 4 0 0 0 5.7 5.7l1.2-1.2"/>',
  external: '<path d="M13.5 4.5h6v6M19.5 4.5l-8 8M18 14v3.5a2 2 0 0 1-2 2H6.5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2H10"/>',
  copy: '<rect x="8.5" y="8.5" width="11" height="11" rx="2.5"/><path d="M15.5 8.5V6.5a2 2 0 0 0-2-2h-7a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h2"/>',
  palette: '<path d="M12 3.5a8.5 8.5 0 1 0 0 17c1.3 0 2-.8 2-1.8 0-1.4-1.3-1.6-1.3-2.9 0-1 .8-1.8 1.8-1.8h2.2a3.8 3.8 0 0 0 3.8-3.8C20.5 6.6 16.7 3.5 12 3.5z"/><circle cx="7.5" cy="12" r="1.1" fill="currentColor"/><circle cx="9.5" cy="7.8" r="1.1" fill="currentColor"/><circle cx="14.5" cy="7.8" r="1.1" fill="currentColor"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2.5v2M12 19.5v2M4.6 4.6 6 6M18 18l1.4 1.4M2.5 12h2M19.5 12h2M4.6 19.4 6 18M18 6l1.4-1.4"/>',
  moon: '<path d="M19.5 14.5A8 8 0 0 1 9.5 4.5a8 8 0 1 0 10 10z"/>',
  bold: '<path d="M7.5 5h5.25a3.5 3.5 0 0 1 0 7H7.5zM7.5 12h6a3.5 3.5 0 0 1 0 7h-6z"/>',
  italic: '<path d="M10.5 5h7M6.5 19h7M14 5l-4 14"/>',
  strike: '<path d="M4.5 12h15M16 7.5c-.6-1.6-2.1-2.5-4-2.5-2.5 0-4 1.3-4 3 0 1.3.9 2.3 2.4 2.9M8 16.5c.6 1.6 2.1 2.5 4.2 2.5 2.6 0 4-1.4 4-3.2"/>',
  code: '<path d="m8.5 7.5-5 4.5 5 4.5M15.5 7.5l5 4.5-5 4.5M13.5 5l-3 14"/>',
  heading: '<path d="M6 5v14M18 5v14M6 12h12"/>',
  list: '<path d="M9.5 6.5h10M9.5 12h10M9.5 17.5h10"/><circle cx="5" cy="6.5" r="1.1" fill="currentColor"/><circle cx="5" cy="12" r="1.1" fill="currentColor"/><circle cx="5" cy="17.5" r="1.1" fill="currentColor"/>',
  "list-ordered": '<path d="M10 6.5h9.5M10 12h9.5M10 17.5h9.5M4.5 5l1.25-.75V9M4.3 14.2c.3-.6.9-.9 1.5-.8.8.1 1.2.8.9 1.5L4.3 18h2.6"/>',
  checklist: '<rect x="3.5" y="4.5" width="5" height="5" rx="1.3"/><path d="m4.8 15.8 1.3 1.3 2.4-2.6M11.5 7h8.5M11.5 16h8.5"/>',
  quote: '<path d="M9.5 7.5h-3a2 2 0 0 0-2 2V12h5V7.5zM9.5 12v1.5a4 4 0 0 1-4 4M19.5 7.5h-3a2 2 0 0 0-2 2V12h5V7.5zM19.5 12v1.5a4 4 0 0 1-4 4"/>',
  table: '<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"/><path d="M3.5 9.5h17M3.5 14.5h17M9.5 9.5v10"/>',
  image: '<rect x="3.5" y="4.5" width="17" height="15" rx="3"/><circle cx="9" cy="10" r="1.75"/><path d="m4 17.5 4.5-4.5 3.5 3.5 2.5-2.5 5.5 5"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  file: '<path d="M7 3.5h6.5L18.5 8.5v10a2 2 0 0 1-2 2h-9.5a2 2 0 0 1-2-2v-13a2 2 0 0 1 2-2z"/><path d="M13.5 3.5v3a2 2 0 0 0 2 2h3"/>',
  move: '<path d="M3.5 7.5a2 2 0 0 1 2-2h3.6a2 2 0 0 1 1.5.7l1 1.1a2 2 0 0 0 1.5.7h5.4a2 2 0 0 1 2 2v7.5a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2z"/><path d="M9 13.5h6M13 11l2.5 2.5L13 16"/>',
  rename: '<path d="M4 7.5h9M4 12h6M4 16.5h5M14.5 18.5l1-3.5 4.5-4.5a1.6 1.6 0 0 1 2.3 2.3L17.8 17.3z"/>',
  info: '<circle cx="12" cy="12" r="8.5"/><path d="M12 11v5.5"/><circle cx="12" cy="7.8" r="1" fill="currentColor"/>',
  lightbulb: '<path d="M9 18.5h6M10 21h4M12 3.5a6 6 0 0 0-3.5 10.9c.6.5 1 1.2 1 2V16.5h5V16.4c0-.8.4-1.5 1-2A6 6 0 0 0 12 3.5z"/>',
  alert: '<path d="M10.3 4.6 2.9 17.5A2 2 0 0 0 4.6 20.5h14.8a2 2 0 0 0 1.7-3L13.7 4.6a2 2 0 0 0-3.4 0z"/><path d="M12 9.5v4.5"/><circle cx="12" cy="17" r="1" fill="currentColor"/>',
  octagon: '<path d="M8.3 3.5h7.4l4.8 4.8v7.4l-4.8 4.8H8.3l-4.8-4.8V8.3z"/><path d="M12 8v4.5"/><circle cx="12" cy="15.8" r="1" fill="currentColor"/>',
  star: '<path d="m12 4 2.4 5 5.4.6-4 3.7 1.1 5.4L12 16l-4.9 2.7 1.1-5.4-4-3.7 5.4-.6z"/>',
  keyboard: '<rect x="2.5" y="6" width="19" height="12" rx="2.5"/><path d="M6 10h.5M9.25 10h.5M12.5 10h.5M15.75 10h.5M18 10h.01M7.5 14h9"/>',
  refresh: '<path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5V9H15"/>',
  arrow: '<path d="M5 12h14M13 6l6 6-6 6"/>',
  back: '<path d="M19 12H5M11 6l-6 6 6 6"/>',
  sparkle: '<path d="M12 3.5c.5 3.9 2.6 6 6.5 6.5-3.9.5-6 2.6-6.5 6.5-.5-3.9-2.6-6-6.5-6.5 3.9-.5 6-2.6 6.5-6.5zM18.5 15.5c.2 1.6 1 2.4 2.5 2.5-1.5.2-2.3 1-2.5 2.5-.2-1.5-1-2.3-2.5-2.5 1.5-.1 2.3-.9 2.5-2.5z"/>',
  print: '<path d="M7 9V4.5h10V9M7 17H5.5a2 2 0 0 1-2-2v-4a2 2 0 0 1 2-2h13a2 2 0 0 1 2 2v4a2 2 0 0 1-2 2H17"/><rect x="7" y="14" width="10" height="6" rx="1"/>',
  terminal: '<rect x="3" y="4.5" width="18" height="15" rx="3"/><path d="m7 9.5 3 2.5-3 2.5M12.5 15h4.5"/>',
};

/** Returns an SVG string for the named icon. */
export function icon(name, cls = "ic") {
  const body = P[name] || P.doc;
  return `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${body}</svg>`;
}

/** Replaces every <i data-icon="name"> placeholder under root with its SVG. */
export function hydrateIcons(root = document) {
  for (const el of root.querySelectorAll("i[data-icon]")) {
    el.outerHTML = icon(el.dataset.icon);
  }
}
