// Applies the saved theme before first paint so there is never a flash of
// the wrong appearance. Loaded synchronously from <head>; keep it tiny.
(function () {
  var root = document.documentElement;
  var pref = root.getAttribute("data-theme-pref") || "auto";
  try {
    pref = localStorage.getItem("goki.theme") || pref;
  } catch (e) {}
  var forced = new URLSearchParams(location.search).get("theme");
  if (forced) pref = forced;
  var dark = window.matchMedia("(prefers-color-scheme: dark)");
  function resolve(p) {
    if (p === "auto" || !p) return dark.matches ? "graphite" : "daylight";
    return p;
  }
  root.setAttribute("data-theme", resolve(pref));
  root.setAttribute("data-theme-choice", pref);
  try {
    var w = localStorage.getItem("goki.sidebarWidth");
    if (w) root.style.setProperty("--sidebar-w", w + "px");
    if (localStorage.getItem("goki.sidebarHidden") === "1") root.classList.add("sidebar-hidden");
    var accent = localStorage.getItem("goki.accent");
    if (accent) ["--accent", "--accent-hover", "--link"].forEach(function (k) { root.style.setProperty(k, accent); });
  } catch (e) {}
  if (/Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)) root.classList.add("is-mac");
  window.__gokiResolveTheme = resolve;
})();
