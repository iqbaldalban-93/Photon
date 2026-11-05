"use strict";

const S = {
  me: null,
  state: null,
  route: { view: "library" },
  filter: "all",
  q: "",
  sort: localStorage.getItem("gh_sort") || "new",
  density: localStorage.getItem("gh_density") || "m",
  items: [],
  total: 0,
  duplicatesView: false,
  duplicateReturnFilter: "all",
  loading: false,
  selection: new Set(),
  albums: [],
  albumTitles: new Map(),
  syncTimer: null,
  viewer: null,
};

const $app = document.getElementById("app");
const $toasts = document.getElementById("toasts");
const $modalRoot = document.getElementById("modal-root");

function h(html) {
  const t = document.createElement("template");
  t.innerHTML = html.trim();
  return t.content.firstElementChild;
}
function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}
function toast(msg, kind = "", ms = 3800) {
  const el = h(`<div class="toast ${kind}">${esc(msg)}</div>`);
  $toasts.appendChild(el);
  setTimeout(() => el.remove(), ms);
}
async function api(path, opts = {}) {
  const o = { headers: {}, ...opts };
  if (o.json !== undefined) {
    o.method = o.method || "POST";
    o.headers["Content-Type"] = "application/json";
    o.body = JSON.stringify(o.json);
    delete o.json;
  }
  const res = await fetch(path, o);
  if (res.status === 401 && S.me && !S.me.setupNeeded) { renderLogin(); throw new Error("unauthorized"); }
  let data = null;
  try { data = await res.json(); } catch {  }
  if (!res.ok) throw new Error((data && data.error) || `HTTP ${res.status}`);
  return data;
}
function fmtSize(n) {
  if (!n && n !== 0) return "";
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0, v = n;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v < 10 && i > 0 ? v.toFixed(1) : Math.round(v)} ${u[i]}`;
}
function fmtDate(ms) {
  if (!ms) return "—";
  const d = new Date(ms);
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" }) +
    " " + d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}
function fmtDur(ms) {
  if (!ms) return "";
  let s = Math.round(ms / 1000);
  const h = Math.floor(s / 3600);
  s %= 3600;
  const m = Math.floor(s / 60), r = s % 60;
  if (h) return `${h}:${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
  return `${m}:${String(r).padStart(2, "0")}`;
}
const ICONS = {
  images: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2.5"/><circle cx="8.5" cy="8.5" r="1.6"/><path d="m21 15-4.6-4.6a1.5 1.5 0 0 0-2.1 0L5 19.6"/></svg>',
  film: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2.5"/><path d="M3 9h18M3 15h18M8 4v16M16 4v16"/></svg>',
  eye: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/></svg>',
  eyeoff: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M10.6 5.1A10 10 0 0 1 12 5c6.4 0 10 7 10 7a17 17 0 0 1-3.2 4.1M6.6 6.6A17 17 0 0 0 2 12s3.6 7 10 7a9.7 9.7 0 0 0 4-.9"/><path d="M9.9 9.9a3 3 0 0 0 4.2 4.2"/><path d="M3 3l18 18"/></svg>',
  video: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>',
  pause: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 4h4v16H6zM14 4h4v16h-4z"/></svg>',
  vol: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3a4.5 4.5 0 0 0-2.5-4v8a4.5 4.5 0 0 0 2.5-4zM14 3.2v2.1a7 7 0 0 1 0 13.4v2.1a9 9 0 0 0 0-17.6z"/></svg>',
  mute: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M3 9v6h4l5 5V4L7 9H3zm18.6 8.2-2.8-2.8 2.8-2.8-1.4-1.4-2.8 2.8-2.8-2.8-1.4 1.4 2.8 2.8-2.8 2.8 1.4 1.4 2.8-2.8 2.8 2.8z"/></svg>',
  fs: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5"/></svg>',
  photo: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M21 19V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2zM8.5 13.5l2.5 3 3.5-4.5 4.5 6H5l3.5-4.5z"/></svg>',
  check: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M9 16.2 4.8 12l-1.4 1.4L9 19 21 7l-1.4-1.4z"/></svg>',
  star: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="m12 17.3-6.2 3.7 1.7-7L2 9.2l7.1-.6L12 2l2.9 6.6 7.1.6-5.5 4.8 1.7 7z"/></svg>',
  search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></svg>',
  lib: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="3" y="3" width="7.5" height="7.5" rx="1.5"/><rect x="13.5" y="3" width="7.5" height="7.5" rx="1.5"/><rect x="3" y="13.5" width="7.5" height="7.5" rx="1.5"/><rect x="13.5" y="13.5" width="7.5" height="7.5" rx="1.5"/></svg>',
  dup: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>',
  album: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 8V6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8Z"/></svg>',
  rmalbum: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 8V6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8Z"/><path d="M9 14h6"/></svg>',
  gear: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3h.1a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5h.1a1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9v.1a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2m3 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/></svg>',
  restore: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/></svg>',
  download: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 4v12m0 0 4-4m-4 4-4-4"/><path d="M4 19h16"/></svg>',
  archive: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="3" y="4" width="18" height="4" rx="1"/><path d="M5 8v11a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V8m-7 3v5m-2.5-2.5h5"/></svg>',
  plus: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>',
  x: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>',
  left: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m15 18-6-6 6-6"/></svg>',
  right: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 6 6 6-6 6"/></svg>',
  playlist: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 6h12M4 11h12M4 16h7"/><path d="M17 12v7l5-3.5z" fill="currentColor" stroke="none"/></svg>',
  back5: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 12H5"/><path d="m11 18-6-6 6-6"/></svg>',
  fwd5: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m13 6 6 6-6 6"/></svg>',
  logout: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><path d="m16 17 5-5-5-5"/><path d="M21 12H9"/></svg>',
  kebab: '<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',
};
ICONS.copy = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V6a2 2 0 0 1 2-2h9"/></svg>';
ICONS.selectall = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="13" height="13" rx="3"/><path d="M8 20h9a4 4 0 0 0 4-4V7"/><path d="m6.5 9.5 1.7 1.7 3-3.4"/></svg>';
ICONS.select = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="16" height="16" rx="4"/><path d="m8.5 12 2.5 2.5 4.5-5"/></svg>';
const PAGE = 200;
const GV_MIN = 301;
const GV_BUF = 6;

async function boot() {
  try {
    S.me = await api("/api/auth/status");
  } catch {
    renderLogin();
    return;
  }
  if (S.me.setupNeeded) { renderSetup(); return; }
  if (!S.me.loggedIn) { renderLogin(); return; }
  await refreshState();
  api("/api/albums").then(d => setAlbums(d.albums || [])).catch(() => {});
  route();
  addHashListener();
  connectEvents();
}

function connectEvents() {
  if (S.events || typeof EventSource === "undefined") return;
  const es = new EventSource("/api/events");
  S.events = es;
  const parse = (e) => { try { return JSON.parse(e.data); } catch { return null; } };
  es.addEventListener("hello", (e) => {
    const d = parse(e);
    if (d) onAccountChanged(d);
  });
  es.addEventListener("account", (e) => {
    const d = parse(e);
    if (d) onAccountChanged(d);
  });
  es.addEventListener("status", () => syncSoon());
  es.onerror = () => {
    if (es.readyState === EventSource.CLOSED) { S.events = null; }
  };
}

function disconnectEvents() {
  if (S.events) { S.events.close(); S.events = null; }
}

async function onAccountChanged(info, { force = false } = {}) {
  if (!S.me || !S.me.loggedIn) return;
  if (S.switching) { S.switchAgain = info; return; }
  const cur = S.state;
  if (!force && cur && info && cur.epoch === info.epoch && cur.account === info.account) return;
  S.switching = true;
  try {
    const hadAccount = !!(cur && cur.account);
    closeViewer();
    S.selection.clear();
    S.items = [];
    S.total = 0;
    S.q = "";
    S.duplicatesView = false;
    cancelPendingAlbum();
    await refreshState();
    try { const d = await api("/api/albums"); setAlbums(d.albums || []); } catch { setAlbums([]); }
    if (S.route.view === "albums" && S.route.arg) {
      nav("albums");
    } else {
      route();
    }
    if (S.state && S.state.account && hadAccount) toast("Switched to " + S.state.account, "ok");
    syncSoon();
  } finally {
    S.switching = false;
    const again = S.switchAgain;
    S.switchAgain = null;
    if (again) onAccountChanged(again);
  }
}

async function refreshState() {
  try { S.state = await api("/api/state"); } catch (e) { toast(e.message, "err"); }
}

const MEDIA_ROUTES = { photos: "photo", videos: "video" };

function route() {
  const hash = location.hash.replace(/^#\/?/, "");
  const [view, arg] = hash.split("/");
  S.route = { view: view || "library", arg: arg ? decodeURIComponent(arg) : "" };
  const watchView = view === "watched" || view === "unwatched" ? view : "";
  const mediaView = MEDIA_ROUTES[view] || "";
  if (S.mediaRoute && !mediaView && S.filter === S.mediaRoute) S.filter = "all";
  S.mediaRoute = mediaView;
  if (mediaView) {
    S.route = { view: "library", arg: "", media: mediaView };
    S.duplicatesView = false;
    S.filter = mediaView;
  } else if (watchView) {
    S.route = { view: "library", arg: "", watch: watchView };
    S.duplicatesView = false;
    S.filter = watchView;
  } else if (S.filter === "watched" || S.filter === "unwatched") {
    S.filter = "all";
  }
  if (S.route.view !== "library" && S.duplicatesView) {
    S.duplicatesView = false;
    S.filter = S.duplicateReturnFilter;
  }
  S.selection.clear();
  cancelPendingAlbum();
  if (!["library", "albums", "settings"].includes(S.route.view)) S.route.view = "library";
  render();
}

function nav(view, arg) {
  location.hash = "/" + view + (arg ? "/" + encodeURIComponent(arg) : "");
}

function renderAuth({ title, sub, fields, submitLabel, endpoint, rerender, errMsg, preSubmit }) {
  stopTimers();
  $app.innerHTML = "";
  const el = h(`
  <div class="auth-wrap">
    <div class="auth-card">
      <div class="logo">
        <img src="/favicon.svg" alt="">
        <div><h1>${esc(title)}</h1></div>
      </div>
      <p class="sub">${esc(sub)}</p>
      <form id="f">
        ${fields}
        <div class="auth-error">${esc(errMsg || "")}</div>
        <button class="btn primary" type="submit">${esc(submitLabel)}</button>
      </form>
      ${title === "Photon" ? '<div class="auth-foot">Photon · private library</div>' : ""}
    </div>
  </div>`);
  $app.appendChild(el);
  el.querySelector("#f").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    if (preSubmit) {
      const early = preSubmit(fd);
      if (early) { rerender(early); return; }
    }
    try {
      await api(endpoint, { json: { username: fd.get("u"), password: fd.get("p") } });
      S.me = await api("/api/auth/status");
      await refreshState();
      route();
      addHashListener();
    } catch (err) {
      rerender(err.message);
    }
  });
}

function renderLogin(errMsg) {
  renderAuth({
    title: "Photon",
    sub: "Sign in to your private Google Photos console.",
    fields: `
        <label class="field"><span>Username</span><input type="text" name="u" autocomplete="username" required autofocus></label>
        <label class="field"><span>Password</span><input type="password" name="p" autocomplete="current-password" required></label>`,
    submitLabel: "Sign in",
    endpoint: "/api/auth/login",
    rerender: renderLogin,
    errMsg,
  });
}

function renderSetup(errMsg) {
  renderAuth({
    title: "Welcome",
    sub: "Create your login — this instance is for you alone.",
    fields: `
        <label class="field"><span>Username</span><input type="text" name="u" required autofocus></label>
        <label class="field"><span>Password</span><input type="password" name="p" minlength="8" required></label>
        <label class="field"><span>Repeat password</span><input type="password" name="p2" minlength="8" required></label>`,
    submitLabel: "Create login",
    endpoint: "/api/auth/setup",
    rerender: renderSetup,
    errMsg,
    preSubmit: (fd) => (fd.get("p") !== fd.get("p2") ? "Passwords do not match" : null),
  });
}

let hashListenerAdded = false;
function addHashListener() {
  if (hashListenerAdded) return;
  hashListenerAdded = true;
  window.addEventListener("hashchange", route);
}

const FILTERS = [
  ["all", "All"], ["video", "Videos"], ["photo", "Photos"],
  ["favorite", "Favorites"], ["archived", "Archived"], ["trash", "Trash"],
];
const SORTS = [
  ["new", "Newest first"], ["old", "Oldest first"], ["name", "Name A–Z"],
  ["size", "Largest file"], ["dur", "Longest video"],
];
const VIEWMODES = [
  ["xl", "Extra large icons"], ["l", "Large icons"], ["m", "Medium icons"],
  ["s", "Small icons"], ["list", "List"], ["details", "Details"],
  ["tiles", "Tiles"], ["content", "Content"],
];
const VMICONS = {
  xl: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="3" width="18" height="18" rx="2"/></svg>',
  l: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="5" y="5" width="14" height="14" rx="2"/></svg>',
  m: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="7" y="7" width="10" height="10" rx="2"/></svg>',
  s: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="4" y="4" width="6" height="6" rx="1"/><rect x="14" y="4" width="6" height="6" rx="1"/><rect x="4" y="14" width="6" height="6" rx="1"/><rect x="14" y="14" width="6" height="6" rx="1"/></svg>',
  list: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="4" y="4" width="7" height="5" rx="1"/><rect x="4" y="11" width="7" height="5" rx="1"/><rect x="4" y="18" width="7" height="3" rx="1"/><path d="M14 6.5h6M14 13.5h6M14 19.5h6"/></svg>',
  details: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="4" y="4" width="6" height="4" rx="1"/><rect x="4" y="11" width="6" height="4" rx="1"/><rect x="4" y="18" width="6" height="3" rx="1"/><path d="M13 5h7M13 8h5M13 12h7M13 15h5M13 19h7"/></svg>',
  tiles: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="4" width="8" height="7" rx="1"/><rect x="3" y="13" width="8" height="7" rx="1"/><path d="M14 6h7M14 8.5h5M14 15h7M14 17.5h5"/></svg>',
  content: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 6h16M4 10h16M4 14h16M4 18h10"/></svg>',
};

function renderShell(content) {
  const st = S.state || {};
  const counts = st.counts || {};
  const inAlbum = S.route.view === "albums" && !!S.route.arg;
  const sync = st.sync || {};
  const initial = (st.username || "?").slice(0, 1).toUpperCase();
  const navItems = [
    ["library", ICONS.lib, "Library", (counts.all || "")],
    ["photos", ICONS.images, "Photos", (counts.photo || "")],
    ["videos", ICONS.film, "Videos", (counts.video || "")],
    ["unwatched", ICONS.eyeoff, "Unwatched", (counts.unwatched || "")],
    ["watched", ICONS.eye, "Watched", (counts.watched || "")],
    ["albums", ICONS.album, "Albums", (st.stats && st.stats.albums) || ""],
    ["duplicates", ICONS.dup, "Find Duplicates", ""],
    ["settings", ICONS.gear, "Settings", ""],
  ];
  $app.innerHTML = "";
  const shell = h(`
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <img src="/favicon.svg" alt="">
        <div><b>Photon</b><small>Google Photos console</small></div>
      </div>
      <div id="nav"></div>
      <div class="spacer"></div>
      <div class="sync-card" id="syncCard"></div>
    </aside>
    <div class="main">
      <div class="topbar">
        ${inAlbum ? `<button class="btn icon ghost backbtn" id="backBtn" title="Back to albums" aria-label="Back to albums">${ICONS.left}</button>` : ""}
        <h2 id="viewTitle">Library</h2>
        <button class="syncdot-m" id="syncDot" title="Library sync" aria-label="Library sync status"><span class="sync-dot"></span></button>
        <div class="search">
          ${ICONS.search}
          <input type="search" id="q" placeholder="Search by filename…" enterkeyhint="search" autocomplete="off" spellcheck="false" value="${esc(S.q)}">
        </div>
        <div class="grow"></div>
        <label class="sortsel" id="sortWrap" style="display:none">
          <select id="sortSel" title="Sort order">${SORTS.map(([v, l]) =>
            `<option value="${v}" ${S.sort === v ? "selected" : ""}>${l}</option>`).join("")}</select>
        </label>
        <div class="viewmenu-wrap" id="densityWrap" style="display:none">
          <button class="viewdd" id="viewBtn" title="Change view mode">
            <span class="vic-m" id="viewIcon">${VMICONS[S.density] || VMICONS.m}</span>
            <span id="viewLbl">${(VIEWMODES.find(v => v[0] === S.density) || VIEWMODES[2])[1]}</span>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>
          </button>
          <div class="viewmenu" id="viewMenu" hidden>
            ${VIEWMODES.map(([v, l]) =>
              `<button class="vitem ${S.density === v ? "active" : ""}" data-v="${v}">
                 <span class="vic">${VMICONS[v]}</span><span>${l}</span>${ICONS.check.replace('viewBox', 'class="vcheck" viewBox')}
               </button>`).join("")}
          </div>
        </div>
        <div class="userchip">
          <span>${esc(st.account || "no account")}</span>
          <div class="avatar">${esc(initial)}</div>
          <button class="btn ghost small" id="logout" aria-label="Sign out">${ICONS.logout}<span>Sign out</span></button>
        </div>
      </div>
      <div id="content" class="scroller"></div>
    </div>
  </div>`);
  $app.appendChild(shell);

  const navEl = shell.querySelector("#nav");
  for (const [v, icon, label, badge] of navItems) {
    const active = v === "duplicates"
      ? (S.route.view === "library" && S.duplicatesView)
      : (v === "watched" || v === "unwatched") ? S.route.watch === v
      : (v === "photos" || v === "videos") ? (!S.duplicatesView && S.route.media === MEDIA_ROUTES[v])
      : (v === "library" ? (S.route.view === "library" && !S.duplicatesView && !S.route.watch && !S.route.media) : S.route.view === v);
    const b = h(`<button class="nav-item ${active ? "active" : ""}" data-nav="${v}" ${active ? 'aria-current="page"' : ""}>${icon}<span>${label}</span>${badge !== "" ? `<span class="badge">${badge}</span>` : ""}</button>`);
    b.addEventListener("click", () => {
      if (v === "duplicates" || (v === "library" && S.duplicatesView)) { toggleDuplicates(); return; }
      nav(v);
    });
    navEl.appendChild(b);
  }

  renderSyncCard(shell.querySelector("#syncCard"));
  shell.querySelector("#logout").addEventListener("click", async () => {
    await api("/api/auth/logout", { method: "POST" });
    location.hash = "";
    renderLogin();
  });
  shell.querySelector("#syncDot").addEventListener("click", () => nav("settings"));
  shell.querySelector("#backBtn")?.addEventListener("click", () => nav("albums"));

  const qInput = shell.querySelector("#q");
  let qTimer;
  qInput.addEventListener("input", () => {
    clearTimeout(qTimer);
    qTimer = setTimeout(() => {
      S.q = qInput.value.trim();
      if (!S.duplicatesView && (S.route.view === "library" || S.route.view === "albums")) {
        if (S.route.view === "albums" && S.route.arg) { loadItems(); }
        else if (S.route.view === "library") { loadItems(); }
      }
    }, 300);
  });

  const showBrowseControls = ["library", "albums"].includes(S.route.view);
  shell.querySelector("#sortWrap").style.display = showBrowseControls ? "" : "none";
  shell.querySelector("#densityWrap").style.display = showBrowseControls ? "" : "none";
  shell.querySelector("#sortSel").addEventListener("change", (e) => {
    S.sort = e.target.value;
    localStorage.setItem("gh_sort", S.sort);
    loadItems();
  });

  const viewBtn = shell.querySelector("#viewBtn");
  const viewMenu = shell.querySelector("#viewMenu");
  viewBtn.addEventListener("click", (e) => {
    e.stopPropagation();
    viewMenu.hidden = !viewMenu.hidden;
  });
  viewMenu.addEventListener("click", (e) => {
    const item = e.target.closest("[data-v]");
    if (!item) return;
    S.density = item.dataset.v;
    localStorage.setItem("gh_density", S.density);
    viewMenu.querySelectorAll(".vitem").forEach(x => x.classList.toggle("active", x === item));
    shell.querySelector("#viewLbl").textContent = (VIEWMODES.find(v => v[0] === S.density) || VIEWMODES[2])[1];
    const vIcon = shell.querySelector("#viewIcon");
    if (vIcon) vIcon.innerHTML = VMICONS[S.density] || VMICONS.m;
    viewMenu.hidden = true;
    const grid = document.getElementById("grid");
    if (grid) {
      grid.className = "grid grid-" + S.density;
      if (S.duplicatesView) loadDuplicates();
      else drawGrid();
    }
  });
  if (S.viewMenuOutside) document.removeEventListener("click", S.viewMenuOutside);
  S.viewMenuOutside = (e) => {
    if (!viewMenu.hidden && !e.target.closest(".viewmenu-wrap")) viewMenu.hidden = true;
  };
  document.addEventListener("click", S.viewMenuOutside);

  content(shell.querySelector("#content"), shell.querySelector("#viewTitle"));
}

function renderSyncCard(el) {
  const st = S.state || {};
  const sync = st.sync || {};
  const cls = sync.running ? "busy" : (sync.lastError ? "err" : "");
  const label = sync.running
    ? `Syncing… ${sync.phase} · page ${sync.pages}`
    : (sync.lastError ? "Sync error" : "Library up to date");
  const total = st.stats ? st.stats.totalItems : 0;
  el.innerHTML = `<span class="sync-dot ${cls}"></span><b>${esc(label)}</b>
    ${sync.lastError ? `<div class="faint" style="color:var(--danger)">${esc(sync.lastError)}</div>` : ""}
    <div>${total} items indexed</div>
    <div class="faint">${esc(autoSyncLabel(st.autosyncSec))}</div>
    <button class="btn small" style="margin-top:8px;width:100%" id="syncBtn">Sync now</button>`;
  el.querySelector("#syncBtn").addEventListener("click", async () => {
    try {
      await api("/api/library/sync", { json: { full: false } });
      toast("Library sync started", "ok");
      syncSoon();
    } catch (e) { toast(e.message, "err"); }
  });
  const dotBtn = document.getElementById("syncDot");
  if (dotBtn) {
    const d = dotBtn.firstElementChild;
    if (d) d.className = "sync-dot " + cls;
    dotBtn.title = label;
    dotBtn.setAttribute("aria-label", label);
  }
  pollSync();
}

function toggleDuplicates() {
  S.selection.clear();
  if (S.duplicatesView) {
    S.duplicatesView = false;
    S.filter = S.duplicateReturnFilter;
  } else {
    S.duplicateReturnFilter = (S.filter === "watched" || S.filter === "unwatched" || S.route.media) ? "all" : S.filter;
    S.filter = "all";
    S.duplicatesView = true;
  }
  if (S.route.view !== "library" || S.route.watch || S.route.media) location.hash = "/library";
  else render();
}

function renderLibrary(content, titleEl) {
  titleEl.textContent = S.route.view === "albums" && S.route.arg ? "Album"
    : (S.route.watch === "watched" ? "Watched videos" : S.route.watch === "unwatched" ? "Unwatched videos"
    : S.route.media === "photo" ? "Photos" : S.route.media === "video" ? "Videos"
    : (S.duplicatesView ? "Duplicates" : "Library"));
  const counts = (S.state && S.state.counts) || {};

  const chips = h(`<div class="library-controls"><div class="chips">${FILTERS.map(([k, label]) =>
    `<button class="chip ${S.filter === k && !(S.route.view === "albums" && S.route.arg) ? "active" : ""}" data-f="${k}">${label}<span class="n">${counts[k] ?? ""}</span></button>`
  ).join("")}</div></div>`);

  const wrap = h(`<div><div class="grid-wrap"><div class="grid grid-${S.density}" id="grid"></div><div id="sentinel" style="height:1px"></div></div></div>`);
  const holder = h("<div></div>");
  holder.appendChild(chips);
  holder.appendChild(wrap);
  content.appendChild(holder);

  chips.querySelectorAll(".chip").forEach(c => c.addEventListener("click", () => {
    const wasDup = S.duplicatesView;
    if (wasDup) {
      S.duplicatesView = false;
      S.selection.clear();
    }
    S.filter = c.dataset.f;
    if (S.route.media) {
      if (S.route.media === c.dataset.f) return;
      S.mediaRoute = "";
      location.hash = "/library";
      return;
    }
    if ((S.route.view === "albums" && S.route.arg) || S.route.watch) { location.hash = "/library"; return; }
    if (wasDup) { render(); return; }
    chips.querySelectorAll(".chip").forEach(x => x.classList.toggle("active", x === c));
    loadItems();
  }));

  S.gridEl = wrap.querySelector("#grid");
  bindTileContextMenu(S.gridEl);
  S.gridEl.addEventListener("keydown", (e) => {
    if ((e.key === "Enter" || e.key === " ") && e.target.classList && e.target.classList.contains("tile")) {
      e.preventDefault();
      e.target.click();
    }
  });
  S.gridEl.addEventListener("click", (e) => {
    const tile = e.target.closest(".tile");
    if (!tile || !S.gridEl.contains(tile)) return;
    const key = tile.dataset.key;
    if (e.target.closest(".pick")) { toggleSelect(key, tile); return; }
    if (S.duplicatesView || S.selection.size > 0 || S.pendingAlbum) { toggleSelect(key, tile); return; }
    const idx = S.items.findIndex(x => x.mediaKey === key);
    if (idx >= 0) openViewer(idx);
  });
  S.gridEl.addEventListener("error", (e) => {
    if (e.target && e.target.tagName === "IMG") e.target.style.visibility = "hidden";
  }, true);
  S.sentinelEl = wrap.querySelector("#sentinel");
  S.emptyEl = null;
  loadItems();
  setupSelectionBar();
}

async function loadItems(append = false) {
  if (S.duplicatesView) return loadDuplicates();
  const grid = S.gridEl;
  if (!grid) return;
  const prevCount = S.items.length;
  if (!append) {
    S.offset = 0;
    S.items = [];
    S.renderedCount = 0;
    grid.innerHTML = `<div class="thumb-skeleton" style="position:static;aspect-ratio:auto;height:120px;border-radius:12px;grid-column:1/-1"></div>`;
  }
  S.loading = true;
  const epochAtStart = S.state && S.state.epoch;
  try {
    const filter = (S.route.view === "albums" && S.route.arg) ? "album" : S.filter;
    const album = S.route.view === "albums" ? S.route.arg : "";
    const data = await api(`/api/media?filter=${encodeURIComponent(filter)}&album=${encodeURIComponent(album)}&q=${encodeURIComponent(S.q)}&sort=${encodeURIComponent(S.sort)}&offset=${S.offset}&limit=${PAGE}`);
    if ((S.state && S.state.epoch) !== epochAtStart) return;
    const fresh = append
      ? data.items.filter(it => !S.items.some(x => x.mediaKey === it.mediaKey))
      : data.items;
    if (append) S.items = S.items.concat(fresh);
    else S.items = data.items;
    S.total = data.total;
    S.offset = S.items.length;
    drawGrid(append ? prevCount : 0);
  } catch (e) {
    toast(e.message, "err");
  } finally {
    S.loading = false;
  }
}

async function loadDuplicates() {
  const grid = S.gridEl;
  if (!grid) return;
  S.loading = true;
  S.gv = null;
  S.offset = 0;
  S.items = [];
  S.total = 0;
  grid.innerHTML = `<div class="thumb-skeleton" style="position:static;aspect-ratio:auto;height:120px;border-radius:12px;grid-column:1/-1"></div>`;
  if (io) io.disconnect();
  try {
    const data = await api("/api/library/duplicates");
    if (!S.duplicatesView) return;
    const groups = data.groups || [];
    S.items = groups.flatMap(group => group.items || []);
    S.total = S.items.length;
    S.offset = S.total;
    grid.className = "grid grid-" + S.density;
    grid.innerHTML = "";
    if (!groups.length) {
      grid.appendChild(h(`<div class="empty" style="grid-column:1/-1"><div class="big">✓</div><h3>No duplicates found</h3><div>Scanned the full library, including archived items, using the filename text before the first space and ignoring case. Items in Trash are excluded.</div></div>`));
      setupSelectionBar();
      return;
    }
    const frag = document.createDocumentFragment();
    frag.appendChild(h(`<div class="duplicate-summary">Found ${groups.length} duplicate groups across ${S.items.length} items. Select the copies you want to move to Trash.</div>`));
    for (const group of groups) {
      frag.appendChild(h(`<div class="duplicate-group-title" style="grid-column:1/-1">Prefix “${esc(group.prefix)}” · ${group.items.length} items</div>`));
      for (const item of group.items) frag.appendChild(tileEl(item));
    }
    grid.appendChild(frag);
    S.renderedCount = S.items.length;
    setupSelectionBar();
  } catch (e) {
    grid.innerHTML = `<div class="empty" style="grid-column:1/-1"><h3>Could not find duplicates</h3><div>${esc(e.message)}</div></div>`;
    toast(e.message, "err");
  } finally {
    S.loading = false;
  }
}

function thumbURL(it, w = 400, sq = true) {
  return `/api/thumb/${encodeURIComponent(it.mediaKey)}?w=${w}&h=${w}&sq=${sq ? 1 : 0}&v=${it.contentVersion || 0}`;
}

function gvMeasure(grid) {
  const cs = getComputedStyle(grid);
  const tracks = cs.gridTemplateColumns.split(" ").filter(Boolean);
  const cols = Math.max(1, tracks.length);
  const rowGap = parseFloat(cs.rowGap) || 0;
  const trackW = parseFloat(tracks[0]) || 104;
  let tileH;
  if (S.density === "tiles") tileH = 96;
  else if (S.density === "details") tileH = 56;
  else if (S.density === "list") tileH = 44;
  else if (S.density === "content") tileH = 48;
  else tileH = trackW;
  return { cols, rowFull: Math.max(32, tileH + rowGap) };
}

let gvTick = false;
function gvRenderWindow(grid) {
  const gv = S.gv;
  if (!gv || !grid.isConnected) return;
  const n = S.items.length;
  const gRect = grid.getBoundingClientRect();
  let topEdge, botEdge;
  const scroller = grid.closest(".scroller");
  if (scroller) {
    const sRect = scroller.getBoundingClientRect();
    topEdge = Math.max(0, sRect.top - gRect.top);
    botEdge = sRect.bottom - gRect.top;
  } else {
    topEdge = Math.max(0, -gRect.top);
    botEdge = window.innerHeight - gRect.top;
  }
  const totalRows = Math.ceil(n / gv.cols);
  const firstRow = Math.max(0, Math.min(Math.floor(topEdge / gv.rowFull) - GV_BUF, totalRows - 1));
  const lastRow = Math.min(totalRows - 1, Math.floor(botEdge / gv.rowFull) + GV_BUF);
  const first = firstRow * gv.cols;
  const last = Math.min(n, (lastRow + 1) * gv.cols);
  if (first === gv.first && last === gv.last) return;
  gv.first = first; gv.last = last;

  const frag = document.createDocumentFragment();
  frag.appendChild(h(`<div class="gv-spacer" style="height:${firstRow * gv.rowFull}px"></div>`));
  for (let i = first; i < last; i++) frag.appendChild(tileEl(S.items[i]));
  frag.appendChild(h(`<div class="gv-spacer" style="height:${Math.max(0, (totalRows - lastRow - 1) * gv.rowFull)}px"></div>`));
  grid.replaceChildren(frag);
}

function gvInit(grid) {
  S.gv = Object.assign({ first: -1, last: -1 }, gvMeasure(grid));
  gvRenderWindow(grid);
  if (gvInit.bound) return;
  gvInit.bound = true;
  document.addEventListener("scroll", () => {
    if (!S.gv || !S.gridEl?.isConnected || gvTick) return;
    gvTick = true;
    requestAnimationFrame(() => {
      gvTick = false;
      if (S.gv && S.gridEl?.isConnected && !S.duplicatesView) gvRenderWindow(S.gridEl);
    });
  }, { capture: true, passive: true });
}

function drawGrid(from = 0) {
  const grid = S.gridEl;
  if (!grid) return;
  grid.className = "grid grid-" + S.density;

  if (from === 0) {
    grid.innerHTML = "";
    S.renderedCount = 0;
    S.gv = null;
  }

  if (!S.items.length) {
    if (from === 0) {
      const msg = S.route.watch === "unwatched"
        ? ["No unwatched videos", S.total === 0 && S.state && S.state.counts && S.state.counts.watched ? "You have watched every video in your library." : "Videos appear here until you have played at least 2% of them."]
        : S.route.watch === "watched"
        ? ["No watched videos yet", "A video moves here once you have played at least 2% of it."]
        : S.route.media === "photo"
        ? ["No photos yet", "Photos appear here after your library has synced."]
        : S.route.media === "video"
        ? ["No videos yet", "Videos appear here after your library has synced."]
        : S.route.view === "albums" && S.route.arg
        ? ["No items in this album", "Add some from the library using selection."]
        : ["Nothing here yet", S.state && !S.state.sync?.running ? "Sync your library to get started." : "Waiting for sync…"];
      grid.innerHTML = `<div class="empty" style="grid-column:1/-1"><div class="big">🖼️</div><h3>${msg[0]}</h3><div>${msg[1]}</div></div>`;
    }
    setupSelectionBar();
    return;
  }

  if (S.items.length > GV_MIN) {
    setupSelectionBar();
    gvInit(grid);
    observeSentinel();
    S.renderedCount = S.items.length;
    return;
  }

  if (from === 0 || !S.renderedCount) {
    grid.innerHTML = "";
    S.renderedCount = 0;
    const frag = document.createDocumentFragment();
    for (const it of S.items) frag.appendChild(tileEl(it));
    grid.appendChild(frag);
  } else {
    const newItems = S.items.slice(from);
    if (newItems.length) {
      const frag = document.createDocumentFragment();
      for (const it of newItems) frag.appendChild(tileEl(it));
      grid.appendChild(frag);
    }
  }
  S.renderedCount = S.items.length;
  setupSelectionBar();
  observeSentinel();
}

function setAlbums(list) {
  S.albums = list || [];
  S.albumTitles = new Map(S.albums.map(a => [a.key, a.title]));
}
function albNamesOf(it) {
  const ids = it.collectionIds || [];
  if (!ids.length || inAlbumView()) return null;
  const names = [];
  for (const k of ids) {
    const t = S.albumTitles.get(k);
    if (t && !names.includes(t)) names.push(t);
  }
  return names.length ? names : null;
}
function albBadge(it) {
  const n = albNamesOf(it);
  if (!n) return "";
  return `<span class="alb" title="In ${n.length > 1 ? "albums" : "album"}: ${esc(n.join(", "))}">${ICONS.album}${n.length > 1 ? `<i>${n.length}</i>` : ""}</span>`;
}
function albChip(it) {
  const n = albNamesOf(it);
  if (!n) return "";
  return `<span class="alb inline" title="In ${n.length > 1 ? "albums" : "album"}: ${esc(n.join(", "))}">${ICONS.album}<i>${esc(n[0])}${n.length > 1 ? ` +${n.length - 1}` : ""}</i></span>`;
}
function renderViewerAlbumBar(it) {
  document.getElementById("valbums")?.remove();
  const row = document.querySelector("#viewer .viewer-top .trow");
  if (!row || !it) return;
  const albums = [];
  for (const k of it.collectionIds || []) {
    if (!S.albumTitles.has(k) || albums.some(a => a.key === k)) continue;
    albums.push({ key: k, title: S.albumTitles.get(k) || "Untitled album" });
  }
  if (!albums.length) return;
  const bar = h(`<div class="valbums" id="valbums" aria-label="Albums containing this item">
    <span class="vl">${ICONS.album}<b>In ${albums.length > 1 ? "albums" : "album"}</b></span>
    ${albums.map(a => `<span class="vchip-wrap"><button class="vchip" data-key="${esc(a.key)}" title="Open album ${esc(a.title)}">${esc(a.title)}</button><button class="vx" data-key="${esc(a.key)}" aria-label="Remove from album ${esc(a.title)}" title="Remove from this album">×</button></span>`).join("")}
  </div>`);
  bar.addEventListener("click", (e) => {
    const x = e.target.closest(".vx");
    if (x) { removeFromAlbumInViewer(it, x.dataset.key); return; }
    const b = e.target.closest(".vchip");
    if (!b) return;
    closeViewer();
    nav("albums", b.dataset.key);
  });
  row.appendChild(bar);
}
function removeFromAlbumInViewer(it, albumKey) {
  const title = S.albumTitles.get(albumKey) || "this album";
  confirmModal("Remove from album?", `“${title}” will no longer contain this item. It stays in your Google Photos library and in any other album.`, async () => {
    try {
      await api(`/api/albums/${encodeURIComponent(albumKey)}/remove`, { json: { mediaKeys: [it.mediaKey] } });
      applyLocalAlbumMembership([it.mediaKey], albumKey, "", false);
      it.collectionIds = (it.collectionIds || []).filter(x => x !== albumKey);
      toast("Removed from album", "ok");
      if (inAlbumView() && S.route.arg === albumKey) {
        const i = S.items.findIndex(x => x.mediaKey === it.mediaKey);
        if (i >= 0) S.items.splice(i, 1);
        S.total = Math.max(0, S.total - 1);
        S.offset = Math.min(S.offset, S.items.length);
        closeViewer();
        if (S.gridEl) drawGrid(0); else afterMutation();
        return;
      }
      renderViewerAlbumBar(it);
      const cur = S.viewer && S.viewer.list[S.viewer.idx];
      if (cur === it) updateViewerAlbumMeta();
      refreshState().then(updateCountBadges);
    } catch (err) { toast(err.message, "err"); }
  });
}
function applyLocalAlbumMembership(keys, albumKey, albumTitle, add) {
  if (!albumKey) return;
  if (albumTitle) S.albumTitles.set(albumKey, albumTitle);
  const want = new Set(keys);
  for (const it of S.items) {
    if (!want.has(it.mediaKey)) continue;
    const ids = it.collectionIds || (it.collectionIds = []);
    const has = ids.includes(albumKey);
    if (add && !has) ids.push(albumKey);
    if (!add && has) it.collectionIds = ids.filter(x => x !== albumKey);
  }
}
function updateViewerAlbumMeta() {
  const span = document.getElementById("vsub");
  if (!span || !S.viewer) return;
  const it = S.viewer.list[S.viewer.idx];
  renderViewerAlbumBar(it);
  if (it) span.textContent = `${fmtDate(it.timestampMs)} · ${fmtSize(it.size)}${it.width ? ` · ${it.width}×${it.height}` : ""}`;
}

function tileEl(it) {
  const mode = S.density;
  const isVideo = it.type === "video" || it.type === "micro-video";
  const selected = S.selection.has(it.mediaKey);

  let el;
  if (mode === "list") {
    el = h(`
    <div class="tile ${selected ? "selected" : ""}" tabindex="0" role="button" data-key="${esc(it.mediaKey)}" title="${esc(it.fileName)}">
      <div class="pick">${ICONS.check}</div>
      <div class="dname">${it.isFavorite ? `<span class="fav inline">${ICONS.star}</span>` : ""}${esc(it.fileName)}${isVideo ? ` <span class="pill">video</span>` : ""}${it.watched ? ` <span class="pill">watched</span>` : ""}${albChip(it)}</div>
      <div class="dcol ddate">${fmtDate(it.timestampMs)}</div>
      <div class="dcol num dsize">${fmtSize(it.size)}</div>
      <div class="dcol num ddur">${isVideo ? (it.durationMs ? fmtDur(it.durationMs) : "—") : "—"}</div>
    </div>`);
  } else if (mode === "details") {
    el = h(`
    <div class="tile ${selected ? "selected" : ""}" tabindex="0" role="button" data-key="${esc(it.mediaKey)}" title="${esc(it.fileName)}">
      <div class="pick">${ICONS.check}</div>
      <img loading="lazy" decoding="async" alt="" src="${thumbURL(it, 96)}">
      <div class="dname">${it.isFavorite ? `<span class="fav inline">${ICONS.star}</span>` : ""}${esc(it.fileName)}${isVideo ? ` <span class="pill">video</span>` : ""}${it.watched ? ` <span class="pill">watched</span>` : ""}${albChip(it)}</div>
      <div class="dcol ddate">${fmtDate(it.timestampMs)}</div>
      <div class="dcol num dsize">${fmtSize(it.size)}</div>
      <div class="dcol num ddur">${isVideo ? (it.durationMs ? fmtDur(it.durationMs) : "—") : "—"}</div>
    </div>`);
  } else if (mode === "content") {
    el = h(`
    <div class="tile ${selected ? "selected" : ""}" tabindex="0" role="button" data-key="${esc(it.mediaKey)}" title="${esc(it.fileName)}">
      <div class="pick">${ICONS.check}</div>
      <img loading="lazy" decoding="async" alt="" src="${thumbURL(it, 72)}">
      <div class="cinfo">
        <b>${esc(it.fileName)}</b>
        <span>${fmtDate(it.timestampMs)}</span>
        <span>${fmtSize(it.size)}${isVideo && it.durationMs ? ` · ${fmtDur(it.durationMs)}` : ""}</span>
        ${it.isFavorite ? `<span class="fav inline">${ICONS.star}</span>` : ""}
        ${albChip(it)}
      </div>
    </div>`);
  } else if (mode === "tiles") {
    el = h(`
    <div class="tile ${selected ? "selected" : ""}" tabindex="0" role="button" data-key="${esc(it.mediaKey)}" title="${esc(it.fileName)}">
      <img loading="lazy" decoding="async" alt="" src="${thumbURL(it, 264)}">
      <div class="tinfo">
        <b>${esc(it.fileName)}</b>
        <span>${fmtDate(it.timestampMs)}</span>
        <span>${fmtSize(it.size)}${it.width ? ` · ${it.width}×${it.height}` : ""}${isVideo && it.durationMs ? ` · ${fmtDur(it.durationMs)}` : ""}</span>
      </div>
      ${it.isFavorite ? `<span class="fav corner">${ICONS.star}</span>` : ""}
      <div class="pick">${ICONS.check}</div>
      ${albBadge(it)}
    </div>`);
  } else {
    el = h(`
    <div class="tile ${selected ? "selected" : ""} ${it.watched ? "is-watched" : ""}" tabindex="0" role="button" data-key="${esc(it.mediaKey)}" title="${esc(it.fileName)}${it.watched ? " (watched)" : ""}">
      ${it.isFavorite ? `<span class="fav">${ICONS.star}</span>` : ""}
      <div class="pick">${ICONS.check}</div>
      <img loading="lazy" decoding="async" alt="" src="${thumbURL(it, mode === "xl" ? 640 : mode === "l" ? 480 : 400)}">
      ${isVideo ? `<span class="type-badge">${ICONS.video}${mode !== "s" && it.durationMs ? fmtDur(it.durationMs) : ""}</span>` : ""}
      ${albBadge(it)}
      <div class="cap">${esc(it.fileName)}</div>
    </div>`);
  }

  return el;
}

function toggleSelect(key, tile) {
  if (S.selection.has(key)) S.selection.delete(key);
  else S.selection.add(key);
  tile.classList.toggle("selected", S.selection.has(key));
  setupSelectionBar();
}

function albumGroupsForSelection() { return albumGroupsFor(S.selection); }
function albumGroupsFor(keySet) {
  const groups = new Map();
  for (const it of S.items) {
    if (!keySet.has(it.mediaKey)) continue;
    for (const ak of it.collectionIds || []) {
      if (!S.albumTitles.has(ak)) continue;
      let g = groups.get(ak);
      if (!g) groups.set(ak, g = { key: ak, title: S.albumTitles.get(ak) || "Untitled album", keys: [] });
      if (!g.keys.includes(it.mediaKey)) g.keys.push(it.mediaKey);
    }
  }
  return [...groups.values()];
}
async function removeSelectionFromAlbums(groups, keep = false) {
  let n = 0;
  try {
    for (const g of groups) {
      await api(`/api/albums/${encodeURIComponent(g.key)}/remove`, { json: { mediaKeys: g.keys } });
      applyLocalAlbumMembership(g.keys, g.key, "", false);
      n += g.keys.length;
    }
    toast(groups.length === 1 ? `Removed ${n} item(s) from “${groups[0].title}”` : `Removed ${n} album membership(s)`, "ok");
  } catch (err) { toast(err.message, "err"); }
  afterMutation(keep);
}
function removeFromAlbumsModal(groups, keep = false) {
  modal(`<h3>Remove from album</h3>
    <p>Choose which album(s) to remove the selected items from. They stay in your Google Photos library.</p>
    <div class="chklist">${groups.map((g, i) => `<label class="chk"><input type="checkbox" data-i="${i}"><span>${esc(g.title)}</span><small>${g.keys.length} selected</small></label>`).join("")}</div>
    <div class="row"><button class="btn" id="no">Cancel</button><button class="btn danger" id="yes">Remove</button></div>`, (m) => {
    m.querySelector("#no").addEventListener("click", closeModal);
    m.querySelector("#yes").addEventListener("click", async () => {
      const picked = [...m.querySelectorAll("input[data-i]:checked")].map(c => groups[+c.dataset.i]);
      if (!picked.length) { toast("Choose at least one album", "err"); return; }
      closeModal();
      await removeSelectionFromAlbums(picked, keep);
    });
  });
}

let ctxMenuEl = null;
let ctxCleanup = null;
let ctxLongPressAt = 0;
let ctxSuppressClickUntil = 0;

function closeCtxMenu() {
  if (!ctxMenuEl) return;
  ctxMenuEl.remove();
  ctxMenuEl = null;
  document.querySelectorAll(".tile.ctx-target").forEach(t => t.classList.remove("ctx-target"));
  if (ctxCleanup) { ctxCleanup(); ctxCleanup = null; }
}

function refreshTile(it) {
  if (!S.gridEl) return;
  const t = S.gridEl.querySelector(`.tile[data-key="${CSS.escape(it.mediaKey)}"]`);
  if (t) t.replaceWith(tileEl(it));
}

async function copyText(text) {
  try { await navigator.clipboard.writeText(text); return; } catch { }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.style.cssText = "position:fixed;left:-9999px;top:0;opacity:0";
  document.body.appendChild(ta);
  ta.select();
  try { document.execCommand("copy"); } catch { }
  ta.remove();
}

function ctxTargets(key) {
  const fromSelection = S.selection.has(key);
  const want = fromSelection ? S.selection : new Set([key]);
  const items = S.items.filter(it => want.has(it.mediaKey));
  return { items, keys: items.map(it => it.mediaKey), fromSelection };
}

function buildCtxEntries(tile, key) {
  const { items, keys, fromSelection } = ctxTargets(key);
  const n = items.length;
  if (!n) return [];
  const one = n === 1 ? items[0] : null;
  const these = n === 1 ? "" : ` ${n} items`;
  const keep = !fromSelection;
  const after = () => afterMutation(keep);
  const inTrash = S.filter === "trash";
  const E = [];
  const sep = () => { if (E.length && !E[E.length - 1].sep) E.push({ sep: true }); };

  if (inTrash) {
    E.push({ label: `Restore${these}`, icon: ICONS.restore, run: async () => {
      await api("/api/media/restore", { json: { keys } });
      toast(`Restored ${n} item(s)`, "ok"); after();
    } });
    E.push({ label: `Delete forever${these}`, icon: ICONS.trash, danger: true, run: () => {
      confirmModal("Delete permanently?", `${n} item(s) will be deleted forever. This cannot be undone.`, async () => {
        try {
          await api("/api/media/delete", { json: { keys } });
          toast(`Deleted ${n} item(s) forever`, "ok"); after();
        } catch (err) { toast(err.message, "err"); }
      });
    } });
    E.heading = n > 1 ? `${n} items` : "";
    return E;
  }

  const isVideo = (it) => it.type === "video" || it.type === "micro-video";

  if (one && !S.duplicatesView) {
    E.push({ label: isVideo(one) ? "Play" : "Open", icon: isVideo(one) ? ICONS.video : ICONS.photo,
      run: () => { const idx = S.items.indexOf(one); if (idx >= 0) openViewer(idx); } });
  }
  const allFav = items.every(it => it.isFavorite);
  E.push({ label: allFav ? `Remove${these} from favorites` : `Add${these} to favorites`, icon: ICONS.star, run: async () => {
    const nf = !allFav;
    for (const it of items) {
      if (!!it.isFavorite === nf) continue;
      await api(`/api/media/${encodeURIComponent(it.mediaKey)}/favorite`, { json: { favorite: nf } });
      it.isFavorite = nf;
    }
    toast(nf ? `Added ${n} item(s) to favorites` : `Removed ${n} item(s) from favorites`, "ok", 1800);
    if (!nf && S.filter === "favorite") after();
    else items.forEach(refreshTile);
  } });
  const vids = items.filter(it => it.type === "video");
  if (vids.length) {
    const allWatched = vids.every(it => it.watched);
    const nw = !allWatched;
    E.push({ label: nw ? `Mark${these} watched` : `Mark${these} unwatched`, icon: nw ? ICONS.eye : ICONS.eyeoff, run: async () => {
      const watchFilter = !!S.route.watch || S.filter === "watched" || S.filter === "unwatched";
      if (one) {
        await setWatched(one, nw, true);
        if (watchFilter) after(); else refreshTile(one);
      } else {
        const r = await api("/api/media/watched", { json: { keys: vids.map(v => v.mediaKey), watched: nw } });
        toast(r.changed ? `Marked ${r.changed} video(s) as ${nw ? "watched" : "unwatched"}` : "No videos were changed", r.changed ? "ok" : "");
        S.watchedDirty = true;
        after();
      }
    } });
  }

  sep();
  E.push({ label: n === 1 ? "Add to album…" : `Add ${n} items to album…`, icon: ICONS.album,
    run: () => addToAlbumModal(keys, { keepSelection: keep }) });
  const groups = albumGroupsFor(new Set(keys));
  if (inAlbumView() || groups.length) {
    const label = inAlbumView() ? "Remove from this album"
      : groups.length === 1 ? `Remove from “${groups[0].title.length > 22 ? groups[0].title.slice(0, 21) + "…" : groups[0].title}”`
      : "Remove from album…";
    E.push({ label, icon: ICONS.rmalbum, run: () => {
      if (inAlbumView()) {
        const albumKey = S.route.arg;
        const title = S.albumTitles.get(albumKey) || "this album";
        confirmModal("Remove from album?", `${n} item(s) will be removed from “${title}”. They stay in your Google Photos library.`, async () => {
          try {
            await api(`/api/albums/${encodeURIComponent(albumKey)}/remove`, { json: { mediaKeys: keys } });
            toast(`Removed ${n} item(s) from album`, "ok");
            after();
          } catch (err) { toast(err.message, "err"); }
        });
      } else if (groups.length === 1) {
        const g = groups[0];
        confirmModal("Remove from album?", `${g.keys.length} item(s) will be removed from “${g.title}”. They stay in your Google Photos library.`,
          () => removeSelectionFromAlbums(groups, keep));
      } else removeFromAlbumsModal(groups, keep);
    } });
  }

  sep();
  if (one) {
    E.push({ label: "Download original", icon: ICONS.download,
      run: () => window.open(`/api/media/${encodeURIComponent(one.mediaKey)}/download?variant=original`, "_blank") });
  }
  if (fromSelection && n > 1) {
    E.push({ label: "Clear selection", icon: ICONS.x, run: () => { S.selection.clear(); setupSelectionBar(); redrawSelection(); } });
  } else {
    E.push({ label: S.selection.has(key) ? "Deselect" : "Select", icon: ICONS.select, run: () => toggleSelect(key, tile) });
  }
  if (one) {
    E.push({ label: "Copy filename", icon: ICONS.copy, run: async () => { await copyText(one.fileName || ""); toast("Filename copied", "ok", 1500); } });
  }

  sep();
  const archiving = S.filter !== "archived";
  E.push({ label: archiving ? `Archive${these}` : `Unarchive${these}`, icon: ICONS.archive, run: async () => {
    await api("/api/media/archive", { json: { keys, archived: archiving } });
    toast(`${archiving ? "Archived" : "Unarchived"} ${n} item(s)`, "ok"); after();
  } });
  E.push({ label: `Move${these} to trash`, icon: ICONS.trash, danger: true, run: async () => {
    await api("/api/media/trash", { json: { keys } });
    toast(`Moved ${n} item(s) to trash`, "ok"); after();
  } });

  while (E.length && E[E.length - 1].sep) E.pop();
  E.heading = n > 1 ? `${n} items` : "";
  return E;
}

function openTileMenu(tile, x, y) {
  const key = tile.dataset.key;
  const entries = buildCtxEntries(tile, key);
  if (!entries.length) return;
  closeCtxMenu();

  const el = h(`<div class="ctxmenu" role="menu" tabindex="-1">${entries.heading ? `<div class="ctx-head">${esc(entries.heading)}</div>` : ""}${
    entries.map((e, i) => e.sep
      ? `<div class="ctx-sep" role="separator"></div>`
      : `<button type="button" class="ctx-item${e.danger ? " danger" : ""}" role="menuitem" data-i="${i}">${e.icon || ""}<span>${esc(e.label)}</span></button>`).join("")}</div>`);
  document.body.appendChild(el);
  ctxMenuEl = el;
  if (!S.selection.has(key)) tile.classList.add("ctx-target");

  const r = el.getBoundingClientRect();
  const vw = window.innerWidth, vh = window.innerHeight;
  let left = x + r.width > vw - 8 ? x - r.width : x;
  let top = y + r.height > vh - 8 ? y - r.height : y;
  left = Math.min(Math.max(8, left), Math.max(8, vw - r.width - 8));
  top = Math.min(Math.max(8, top), Math.max(8, vh - r.height - 8));
  el.style.left = left + "px";
  el.style.top = top + "px";
  el.focus({ preventScroll: true });

  el.addEventListener("contextmenu", (e) => e.preventDefault());
  el.addEventListener("click", (e) => {
    const b = e.target.closest(".ctx-item");
    if (!b) return;
    const entry = entries[+b.dataset.i];
    closeCtxMenu();
    Promise.resolve().then(() => entry.run()).catch((err) => toast(err.message || String(err), "err"));
  });

  const onDown = (e) => { if (!el.contains(e.target)) closeCtxMenu(); };
  const onKey = (e) => {
    if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); closeCtxMenu(); return; }
    if (e.key === "Tab") { closeCtxMenu(); return; }
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp" && e.key !== "Home" && e.key !== "End") return;
    e.preventDefault();
    const btns = [...el.querySelectorAll(".ctx-item")];
    const cur = btns.indexOf(document.activeElement);
    let next;
    if (e.key === "Home") next = 0;
    else if (e.key === "End") next = btns.length - 1;
    else if (e.key === "ArrowDown") next = cur < 0 ? 0 : (cur + 1) % btns.length;
    else next = cur <= 0 ? btns.length - 1 : cur - 1;
    btns[next]?.focus();
  };
  const onClose = () => closeCtxMenu();
  document.addEventListener("pointerdown", onDown, true);
  document.addEventListener("keydown", onKey, true);
  document.addEventListener("scroll", onClose, { capture: true, passive: true });
  window.addEventListener("resize", onClose);
  window.addEventListener("blur", onClose);
  window.addEventListener("hashchange", onClose);
  ctxCleanup = () => {
    document.removeEventListener("pointerdown", onDown, true);
    document.removeEventListener("keydown", onKey, true);
    document.removeEventListener("scroll", onClose, { capture: true });
    window.removeEventListener("resize", onClose);
    window.removeEventListener("blur", onClose);
    window.removeEventListener("hashchange", onClose);
  };
}

function bindTileContextMenu(grid) {
  grid.addEventListener("contextmenu", (e) => {
    const tile = e.target.closest(".tile");
    if (!tile || !grid.contains(tile)) return;
    e.preventDefault();
    if (Date.now() - ctxLongPressAt < 900) return;
    let x = e.clientX, y = e.clientY;
    if (!x && !y) { const r = tile.getBoundingClientRect(); x = r.left + 12; y = r.top + r.height / 2; }
    openTileMenu(tile, x, y);
  });

  let lp = null;
  const cancelLp = () => { if (lp) { clearTimeout(lp.timer); lp = null; } };
  grid.addEventListener("pointerdown", (e) => {
    cancelLp();
    if (e.pointerType !== "touch") return;
    const tile = e.target.closest(".tile");
    if (!tile || !grid.contains(tile)) return;
    const x = e.clientX, y = e.clientY;
    lp = { x, y, timer: setTimeout(() => {
      lp = null;
      ctxSuppressClickUntil = Date.now() + 800;
      ctxLongPressAt = Date.now();
      if (navigator.vibrate) navigator.vibrate(12);
      openTileMenu(tile, x, y);
    }, 500) };
  });
  grid.addEventListener("pointermove", (e) => {
    if (lp && Math.hypot(e.clientX - lp.x, e.clientY - lp.y) > 10) cancelLp();
  });
  grid.addEventListener("pointerup", cancelLp);
  grid.addEventListener("pointercancel", cancelLp);
  grid.addEventListener("click", (e) => {
    if (Date.now() < ctxSuppressClickUntil) { e.stopPropagation(); e.preventDefault(); }
  }, true);
}

let selbarEl = null;
function setupSelectionBar() {
  selbarEl?.remove();
  selbarEl = null;
  if (!S.selection.size) return;

  const inTrash = S.filter === "trash";
  const allSelected = S.items.length > 0 && S.total <= S.items.length && S.items.every(it => S.selection.has(it.mediaKey));
  const canRmAlbum = inAlbumView() || (!inTrash && albumGroupsForSelection().length > 0);
  const bar = h(`<div class="selbar">
    <span class="count">${S.selection.size} selected</span>
    <button class="btn small" data-a="selectall">${ICONS.selectall} ${allSelected ? "All selected" : `Select all${S.total > S.items.length ? ` (${S.total})` : ""}`}</button>
    <div class="sep"></div>
    ${inTrash
      ? `<button class="btn small" data-a="restore">${ICONS.restore} Restore</button>
         <button class="btn small danger" data-a="delete">${ICONS.trash} Delete forever</button>`
      : `<button class="btn small" data-a="fav">${ICONS.star} Favorite</button>
         ${S.pendingAlbum ? `<button class="btn small primary" data-a="newalbum">${ICONS.album} Create “${esc(S.pendingAlbum)}”</button>` : ""}
         <button class="btn small" data-a="album">${ICONS.album} Add to album</button>
         ${S.filter !== "photo" && S.filter !== "watched" ? `<button class="btn small" data-a="watched">${ICONS.eye} Mark watched</button>` : ""}
         ${S.filter !== "photo" && S.filter !== "unwatched" ? `<button class="btn small" data-a="unwatched">${ICONS.eyeoff} Mark unwatched</button>` : ""}
         ${canRmAlbum ? `<button class="btn small danger" data-a="rmalbum">${ICONS.rmalbum} Remove from album</button>` : ""}
         <button class="btn small" data-a="archive">${ICONS.archive} Archive</button>
         <button class="btn small danger" data-a="trash">${ICONS.trash} Trash</button>`}
    <div class="sep"></div>
    <button class="btn small ghost" data-a="clear">${ICONS.x} Clear</button>
  </div>`);
  document.body.appendChild(bar);
  selbarEl = bar;

  bar.addEventListener("click", async (e) => {
    const btn = e.target.closest("[data-a]");
    if (!btn) return;
    const action = btn.dataset.a;
    const keys = [...S.selection];
    try {
      if (action === "clear") { S.selection.clear(); setupSelectionBar(); redrawSelection(); }
      else if (action === "selectall") { await selectAllItems(btn); }
      else if (action === "fav") {
        for (const k of keys) await api(`/api/media/${encodeURIComponent(k)}/favorite`, { json: { favorite: true } });
        toast(`Added ${keys.length} item(s) to favorites`, "ok"); afterMutation();
      }
      else if (action === "trash") {
        await api("/api/media/trash", { json: { keys } });
        toast(`Moved ${keys.length} item(s) to trash`, "ok"); afterMutation();
      }
      else if (action === "restore") {
        await api("/api/media/restore", { json: { keys } });
        toast(`Restored ${keys.length} item(s)`, "ok"); afterMutation();
      }
      else if (action === "delete") {
        confirmModal("Delete permanently?", `${keys.length} item(s) will be deleted forever. This cannot be undone.`, async () => {
          await api("/api/media/delete", { json: { keys } });
          toast(`Deleted ${keys.length} item(s) forever`, "ok"); afterMutation();
        });
      }
      else if (action === "archive") {
        await api("/api/media/archive", { json: { keys, archived: true } });
        toast(`Archived ${keys.length} item(s)`, "ok"); afterMutation();
      }
      else if (action === "album") {
        addToAlbumModal(keys);
      }
      else if (action === "watched" || action === "unwatched") {
        const r = await api("/api/media/watched", { json: { keys, watched: action === "watched" } });
        toast(r.changed ? `Marked ${r.changed} video(s) as ${action}` : "No videos were changed", r.changed ? "ok" : "");
        afterMutation();
      }
      else if (action === "rmalbum" && !inAlbumView()) {
        const groups = albumGroupsForSelection();
        if (!groups.length) { toast("None of the selected items are in an album"); }
        else if (groups.length === 1) {
          const g = groups[0];
          confirmModal("Remove from album?", `${g.keys.length} item(s) will be removed from “${g.title}”. They stay in your Google Photos library.`,
            () => removeSelectionFromAlbums(groups));
        } else removeFromAlbumsModal(groups);
      }
      else if (action === "rmalbum") {
        const albumKey = S.route.arg;
        confirmModal("Remove from album?", `${keys.length} item(s) will be removed from this album. They stay in your Google Photos library.`, async () => {
          try {
            await api(`/api/albums/${encodeURIComponent(albumKey)}/remove`, { json: { mediaKeys: keys } });
            toast(`Removed ${keys.length} item(s) from album`, "ok");
            afterMutation();
          } catch (err) { toast(err.message, "err"); }
        });
      }
      else if (action === "newalbum") {
        const name = S.pendingAlbum;
        await api("/api/albums", { json: { name, mediaKeys: keys } });
        cancelPendingAlbum();
        toast(`Album “${name}” created with ${keys.length} item(s)`, "ok");
        afterMutation();
      }
    } catch (err) { toast(err.message, "err"); }
  });
}

async function selectAllItems(btn) {
  if (S.selectingAll) return;
  S.selectingAll = true;
  if (btn) btn.disabled = true;
  try {
    if (!S.duplicatesView) {
      let guard = 0;
      while (S.items.length < S.total && guard++ < 500) {
        const before = S.items.length;
        await loadItems(true);
        if (S.items.length === before) break;
      }
    }
    for (const it of S.items) S.selection.add(it.mediaKey);
    redrawSelection();
    toast(`Selected ${S.selection.size} item(s)`, "ok", 1500);
  } finally {
    S.selectingAll = false;
    setupSelectionBar();
  }
}

function redrawSelection() {
  document.querySelectorAll(".tile").forEach(t => {
    t.classList.toggle("selected", S.selection.has(t.dataset.key));
  });
}

function updateCountBadges() {
  const counts = (S.state && S.state.counts) || {};
  document.querySelectorAll(".chip").forEach(c => {
    const n = c.querySelector(".n");
    if (n) n.textContent = counts[c.dataset.f] ?? "";
  });
  const setBadge = (v, val) => {
    const b = document.querySelector(`#nav .nav-item[data-nav="${v}"] .badge`);
    if (b) b.textContent = val ?? "";
  };
  setBadge("library", counts.all);
  setBadge("photos", counts.photo || "");
  setBadge("videos", counts.video || "");
  setBadge("unwatched", counts.unwatched || "");
  setBadge("watched", counts.watched || "");
  if (S.state && S.state.stats) setBadge("albums", S.state.stats.albums);
}

async function afterMutation(keepSelection = false) {
  if (!keepSelection) {
    S.selection.clear();
    setupSelectionBar();
  }
  await refreshState();
  updateCountBadges();
  if (S.duplicatesView && S.route.view === "library") loadDuplicates();
  else loadItems();
}

async function refreshAfterSync() {
  if (S.loading) return;
  await refreshState();
  updateCountBadges();
  if (S.duplicatesView && S.route.view === "library") { loadDuplicates(); return; }
  if (S.route.view === "albums" && !S.route.arg) { refreshAlbumGrid(); return; }
  if (inAlbumView()) {
    try {
      const data = await api("/api/albums");
      setAlbums(data.albums || []);
      const cur = S.albums.find(a => a.key === S.route.arg);
      if (!cur) { toast("This album no longer exists"); nav("albums"); return; }
      const t = document.getElementById("viewTitle");
      if (t) t.textContent = cur.title;
    } catch {  }
  }
  if (S.route.view === "library" || inAlbumView()) loadItems();
}

async function syncAlbumsRefresh() {
  try {
    const data = await api("/api/albums");
    const fresh = data.albums || [];
    const sig = a => JSON.stringify(a.map(x => [x.key, x.title, x.itemCount]));
    const changed = sig(fresh) !== sig(S.albums || []);
    setAlbums(fresh);
    if (changed && S.route.view === "albums" && !S.route.arg) refreshAlbumGrid();
  } catch {  }
}

let io = null;
function observeSentinel() {
  if (io) io.disconnect();
  if (!S.sentinelEl) return;
  if (S.offset >= S.total) return;
  io = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && !S.loading) loadItems(true);
  }, { rootMargin: "800px" });
  io.observe(S.sentinelEl);
}

function renderAlbums(content, titleEl) {
  if (S.route.arg) { renderLibrary(content, titleEl); loadAlbumTitle(); return; }
  titleEl.textContent = "Albums";
  const wrap = h(`<div class="grid-wrap"><div class="album-grid" id="albums"></div></div>`);
  content.appendChild(wrap);
  S.albumGridEl = wrap.querySelector("#albums");
  refreshAlbumGrid();
}

function refreshAlbumGrid() {
  const grid = S.albumGridEl;
  if (!grid || !grid.isConnected) return;
  grid.innerHTML = "";

  const addCard = h(`<div class="album-card" style="display:flex;align-items:center;justify-content:center;min-height:120px;border-style:dashed">
    <div style="text-align:center;color:var(--text-dim)">${ICONS.plus}<div style="margin-top:4px;font-size:13px">New album</div></div>
  </div>`);
  addCard.addEventListener("click", () => createAlbumModal());
  grid.appendChild(addCard);

  api("/api/albums").then(data => {
    setAlbums(data.albums || []);
    for (const al of S.albums) {
      if (!al.title) continue;
      const card = h(`
      <div class="album-card" data-key="${esc(al.key)}">
        <div class="cover">${al.coverMediaKey ? `<img loading="lazy" decoding="async" alt="" src="/api/thumb/${encodeURIComponent(al.coverMediaKey)}?w=480&h=300&sq=0&v=0">` : ""}</div>
        <div class="meta"><b>${esc(al.title)}</b><span>${al.itemCount} items</span><button class="albmenu" title="Album options" aria-label="Album options">${ICONS.kebab}</button></div>
      </div>`);
      card.addEventListener("click", (e) => {
        if (e.target.closest(".albmenu")) { e.stopPropagation(); albumMenu(al); return; }
        nav("albums", al.key);
      });
      card.addEventListener("contextmenu", (e) => { e.preventDefault(); albumMenu(al); });
      grid.appendChild(card);
    }
    if (!S.albums.some(al => al.title)) {
      grid.appendChild(h(`<div class="empty" style="grid-column:1/-1"><div class="big">📁</div><h3>No albums yet</h3><div>Create one, or sync your library.</div></div>`));
    }
  }).catch(e => toast(e.message, "err"));
}

function albumMenu(al) {
  modal(`<h3>${esc(al.title)}</h3>
    <div class="row" style="justify-content:flex-start">
      <button class="btn" id="rn">Rename</button>
      <button class="btn danger" id="del">Delete album</button>
    </div>`, (m) => {
    m.querySelector("#rn").addEventListener("click", () => {
      closeModal();
      promptModal("Rename album", al.title, async (name) => {
        try {
          await api(`/api/albums/${encodeURIComponent(al.key)}/rename`, { json: { name } });
          S.albumTitles.set(al.key, name);
          toast("Album renamed", "ok"); render();
        } catch (e) { toast(e.message, "err"); }
      });
    });
    m.querySelector("#del").addEventListener("click", () => {
      closeModal();
      confirmModal("Delete album?", "Items stay in your library; only the album is removed.", async () => {
        try {
          await api(`/api/albums/${encodeURIComponent(al.key)}`, { method: "DELETE" });
          S.albumTitles.delete(al.key);
          toast("Album deleted", "ok"); render();
        } catch (e) { toast(e.message, "err"); }
      });
    });
  });
}

async function loadAlbumTitle() {
  try {
    const data = await api("/api/albums");
    const al = (data.albums || []).find(a => a.key === S.route.arg);
    const t = document.getElementById("viewTitle");
    if (al && t) t.textContent = al.title;
  } catch {  }
}

function createAlbumModal() {
  promptModal("New album name", "", (name) => {
    S.pendingAlbum = name;
    showPendingAlbumBanner();
    nav("library");
  });
}

let pendingBannerEl = null;
function showPendingAlbumBanner() {
  pendingBannerEl?.remove();
  pendingBannerEl = null;
  if (!S.pendingAlbum) return;
  const el = h(`<div class="selbar" style="top:76px;bottom:auto;animation:none">
    <span class="count">Pick photos for the new album “${esc(S.pendingAlbum)}”</span>
    <button class="btn small ghost" data-a="cancelnew">${ICONS.x} Cancel</button>
  </div>`);
  el.querySelector("[data-a=cancelnew]").addEventListener("click", cancelPendingAlbum);
  document.body.appendChild(el);
  pendingBannerEl = el;
}

function cancelPendingAlbum() {
  S.pendingAlbum = null;
  pendingBannerEl?.remove();
  pendingBannerEl = null;
}

function inAlbumView() {
  return S.route.view === "albums" && !!S.route.arg;
}

function addToAlbumModal(keys, opts = {}) {
  api("/api/albums").then(data => {
    const albums = (data.albums || []).filter(a => a.title);
    const m = modal(`<h3>Add ${keys.length} item(s) to album</h3>
      <label class="field"><span>Choose album</span>
        <select id="al">${albums.map(a => `<option value="${esc(a.key)}">${esc(a.title)}</option>`).join("")}</select>
      </label>
      <label class="field"><span>…or create a new album</span><input type="text" id="newal" placeholder="New album name"></label>
      <div class="row"><button class="btn" id="c">Cancel</button><button class="btn primary" id="go">Add</button></div>`, (mm) => {
      mm.querySelector("#c").addEventListener("click", closeModal);
      mm.querySelector("#go").addEventListener("click", async () => {
        const newName = mm.querySelector("#newal").value.trim();
        if (!newName && !mm.querySelector("#al").value) { toast("Choose an album or type a new album name", "err"); return; }
        try {
          if (newName) {
            const created = await api("/api/albums", { json: { name: newName, mediaKeys: keys } });
            applyLocalAlbumMembership(keys, created.key, newName, true);
          } else {
            const key = mm.querySelector("#al").value;
            await api(`/api/albums/${encodeURIComponent(key)}/add`, { json: { mediaKeys: keys } });
            const known = albums.find(a => a.key === key);
            applyLocalAlbumMembership(keys, key, known ? known.title : "", true);
          }
          closeModal();
          toast(`Added ${keys.length} item(s) to album`, "ok");
          if (opts.inViewer) updateViewerAlbumMeta();
          else afterMutation(opts.keepSelection);
        } catch (e) { toast(e.message, "err"); }
      });
    });
  });
}

function openViewer(idx) {
  if (idx < 0 || idx >= S.items.length) return;
  S.viewer = { list: S.items, idx, vlist: null };
  S.viewer.vlist = S.items
    .map((x, i) => ({ x, i }))
    .filter(e => e.x.type === "video" || e.x.type === "micro-video");
  drawViewer();
  document.removeEventListener("keydown", viewerKeys);
  document.addEventListener("keydown", viewerKeys);
  ensureAllItems().then(() => {
    if (!S.viewer) return;
    S.viewer.list = S.items;
    S.viewer.vlist = S.items
      .map((x, i) => ({ x, i }))
      .filter(e => e.x.type === "video" || e.x.type === "micro-video");
    const aside = document.getElementById("vplaylist");
    if (!aside || S.viewer.vlist.length < 2) return;
    aside.outerHTML = buildPlaylistHTML(S.viewer, S.viewer.list[S.viewer.idx]);
    wirePlaylist(document);
    mountPlaylist(S.viewer);
  });
}

async function ensureAllItems() {
  if (S.fetchingAll || S.total <= S.items.length) return;
  S.fetchingAll = true;
  try {
    const seen = new Set(S.items.map(x => x.mediaKey));
    let cursor = S.items.length;
    while (cursor < S.total) {
      const filter = (S.route.view === "albums" && S.route.arg) ? "album" : S.filter;
      const album = S.route.view === "albums" ? S.route.arg : "";
      const data = await api(`/api/media?filter=${encodeURIComponent(filter)}&album=${encodeURIComponent(album)}&q=${encodeURIComponent(S.q)}&sort=${encodeURIComponent(S.sort)}&offset=${cursor}&limit=1000`);
      if (!data.items || !data.items.length) break;
      for (const it of data.items) {
        if (!seen.has(it.mediaKey)) {
          seen.add(it.mediaKey);
          S.items.push(it);
        }
      }
      cursor += data.items.length;
    }
  } catch {  }
  finally { S.fetchingAll = false; }
}

let activeHls = null;
function stopActiveHls() {
  if (activeHls) {
    try { activeHls.destroy(); } catch {  }
    activeHls = null;
  }
}

function closeViewer() {
  S.viewer = null;
  stopActiveHls();
  document.getElementById("viewer")?.remove();
  document.removeEventListener("keydown", viewerKeys);
  if (S.watchedDirty) {
    S.watchedDirty = false;
    if (S.route.view === "library" && !S.duplicatesView && (S.filter === "watched" || S.filter === "unwatched")) loadItems();
  }
}

function viewerKeys(e) {
  if (!S.viewer) return;
  if ($modalRoot.firstChild) return;
  if (e.target && (e.target.tagName === "INPUT" || e.target.tagName === "SELECT" || e.target.tagName === "TEXTAREA" || e.target.tagName === "BUTTON")) return;
  const it = S.viewer.list[S.viewer.idx];
  const isVideo = it && (it.type === "video" || it.type === "micro-video");
  const vid = document.querySelector("#viewer video");

  if (isVideo && vid) {
    if (!e.shiftKey && (e.key === "ArrowLeft" || e.key === "ArrowRight")) {
      e.preventDefault();
      if (vid.__clearPending) vid.__clearPending();
      const delta = e.key === "ArrowRight" ? 5 : -5;
      if (isFinite(vid.duration)) {
        vid.currentTime = Math.min(Math.max(0, vid.currentTime + delta), vid.duration || 1e9);
      } else {
        vid.currentTime = Math.max(0, vid.currentTime + delta);
      }
      showSeekFlash(delta > 0 ? "+5s" : "−5s");
      return;
    }
    if (!e.shiftKey && (e.key === "ArrowUp" || e.key === "ArrowDown")) {
      e.preventDefault();
      const step = e.key === "ArrowUp" ? 0.05 : -0.05;
      if (step > 0) vid.muted = false;
      vid.volume = Math.min(1, Math.max(0, Math.round((vid.volume + step) * 100) / 100));
      showSeekFlash(vid.muted || vid.volume === 0 ? "Muted" : `Volume ${Math.round(vid.volume * 100)}%`);
      return;
    }
    if (e.key === " ") {
      e.preventDefault();
      if (vid.paused) vid.play().catch(() => {});
      else vid.pause();
      return;
    }
  }

  if (e.key === "Escape") closeViewer();
  if (e.key === "ArrowRight" || e.key === "PageDown") stepViewer(1);
  if (e.key === "ArrowLeft" || e.key === "PageUp") stepViewer(-1);
}

let seekFlashEl = null;
function showSeekFlash(text) {
  if (!seekFlashEl || !seekFlashEl.isConnected) {
    seekFlashEl = h(`<div class="seek-flash"></div>`);
    document.getElementById("viewer")?.querySelector(".viewer-stage")?.appendChild(seekFlashEl);
  }
  seekFlashEl.textContent = text;
  seekFlashEl.classList.remove("show");
  void seekFlashEl.offsetWidth;
  seekFlashEl.classList.add("show");
}

function stepViewer(d) {
  const v = S.viewer;
  if (!v) return;
  const it = v.list[v.idx];
  const isVideo = it && (it.type === "video" || it.type === "micro-video");

  if (isVideo && v.vlist && v.vlist.length) {
    const pos = v.vlist.findIndex(e => e.i === v.idx);
    const next = v.vlist[pos + d];
    if (next) { v.idx = next.i; drawViewer(); }
    return;
  }
  const next = v.idx + d;
  if (next < 0 || next >= v.list.length) return;
  v.idx = next;
  drawViewer();
}

function jumpTo(idx) {
  if (!S.viewer || idx < 0 || idx >= S.viewer.list.length) return;
  S.viewer.idx = idx;
  drawViewer();
}

function wirePlaylist(root) {
  root.querySelector("#plHide")?.addEventListener("click", hidePlaylist);
  root.querySelector(".pl-body")?.addEventListener("click", (e) => {
    const row = e.target.closest(".pl-item");
    if (row) jumpTo(parseInt(row.dataset.i, 10));
  });
}

function hidePlaylist() {
  localStorage.setItem("gh_playlist", "0");
  S.plv = null;
  document.getElementById("vplaylist")?.remove();
  const t = document.getElementById("plToggle");
  if (t) { t.classList.add("off"); t.title = "Show playlist"; }
}

function showPlaylist() {
  const v = S.viewer;
  const el = document.getElementById("viewer");
  if (!v || !el || document.getElementById("vplaylist")) return;
  if (!(v.vlist && v.vlist.length > 1)) return;
  localStorage.setItem("gh_playlist", "1");
  const aside = h(buildPlaylistHTML(v, v.list[v.idx]));
  el.querySelector(".viewer-body").appendChild(aside);
  wirePlaylist(aside);
  mountPlaylist(v);
  const t = el.querySelector("#plToggle");
  if (t) { t.classList.remove("off"); t.title = "Hide playlist"; }
}

function toggleVideoFS(stage) {
  if (document.fullscreenElement) { document.exitFullscreen().catch(() => {}); return; }
  if (stage.requestFullscreen) { stage.requestFullscreen().catch(() => {}); return; }
  const v = stage.querySelector("video");
  if (v && v.webkitEnterFullscreen) { try { v.webkitEnterFullscreen(); } catch {  } }
}

function wireViewerGestures(el, isVideo) {
  const layer = el.querySelector(".viewer-stage");
  const stage = el.querySelector("#stage");
  if (!layer || !stage) return;
  const touchUI = matchMedia("(hover: none)").matches;
  const media = () => layer.querySelector(isVideo ? "video" : "img.viewer-img");
  const zoom = () => (S.viewer.zoom || (S.viewer.zoom = { s: 1, x: 0, y: 0 }));

  const pts = new Map();
  let mode = null;
  let axis = null;
  let sx = 0, sy = 0, lx = 0, ly = 0;
  let pinch = null;
  let lastTap = 0, tapX = 0, tapY = 0, tapTimer = null;
  let captured = false;
  const ensureCapture = () => {
    if (captured) return;
    captured = true;
    for (const id of pts.keys()) {
      try { layer.setPointerCapture(id); } catch {  }
    }
  };

  const setTransform = (noAnim) => {
    const m = media(); if (!m) return;
    const z = zoom();
    m.classList.toggle("dragging", !!noAnim);
    m.style.transform = z.s > 1 ? `translate(${z.x}px, ${z.y}px) scale(${z.s})` : "";
  };
  const resetZoom = () => { S.viewer.zoom = { s: 1, x: 0, y: 0 }; setTransform(false); };
  const zoomTo = (s) => { zoom().s = s; setTransform(false); };
  const clampPan = () => {
    const m = media(); const z = zoom();
    if (!m) return;
    const mx = Math.max(0, (m.offsetWidth * z.s - layer.clientWidth) / 2 + 30);
    const my = Math.max(0, (m.offsetHeight * z.s - layer.clientHeight) / 2 + 30);
    z.x = Math.min(mx, Math.max(-mx, z.x));
    z.y = Math.min(my, Math.max(-my, z.y));
  };
  const eatClick = () =>
    el.addEventListener("click", (c) => c.stopImmediatePropagation(), { capture: true, once: true });
  const slideIn = (dir) => {
    const ns = document.querySelector("#viewer #stage");
    if (!ns) return;
    ns.style.animation = `slideFrom${dir > 0 ? "R" : "L"} .18s ease-out`;
    ns.addEventListener("animationend", () => { ns.style.animation = ""; }, { once: true });
  };

  layer.addEventListener("pointerdown", (e) => {
    if (!S.viewer || e.target.closest(".vbar, .playlist, .vnav, button, input, select, a")) return;
    pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (pts.size === 1) {
      mode = "drag"; axis = null;
      sx = lx = e.clientX; sy = ly = e.clientY;
      captured = false;
    } else if (pts.size === 2 && !isVideo) {
      const [a, b] = [...pts.values()];
      pinch = { d: Math.hypot(a.x - b.x, a.y - b.y) || 1, s: zoom().s };
      mode = "pinch";
      captured = true;
      for (const id of pts.keys()) {
        try { layer.setPointerCapture(id); } catch {  }
      }
    }
  });

  layer.addEventListener("pointermove", (e) => {
    if (!pts.has(e.pointerId) || !S.viewer) return;
    pts.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (mode === "pinch" && pts.size >= 2) {
      const [a, b] = [...pts.values()];
      const z = zoom();
      z.s = Math.min(4.5, Math.max(1, pinch.s * (Math.hypot(a.x - b.x, a.y - b.y) / pinch.d)));
      if (z.s <= 1.01) { z.x = 0; z.y = 0; }
      setTransform(true);
      return;
    }
    if (mode !== "drag" || pts.size > 1) return;

    const dx = e.clientX - sx, dy = e.clientY - sy;
    if (!captured && (Math.abs(dx) > 6 || Math.abs(dy) > 6)) ensureCapture();
    if (!axis) {
      if (Math.abs(dx) > 12 && Math.abs(dx) > Math.abs(dy) * 1.15) axis = "x";
      else if (Math.abs(dy) > 12) axis = "y";
    }
    const z = zoom();
    if (z.s > 1) {
      z.x += e.clientX - lx; z.y += e.clientY - ly;
      lx = e.clientX; ly = e.clientY;
      clampPan();
      setTransform(true);
      return;
    }
    lx = e.clientX; ly = e.clientY;
    const m = media();
    if (!m) return;
    if (axis === "x") { m.classList.add("dragging"); m.style.transform = `translateX(${dx}px)`; }
    else if (axis === "y" && dy > 0) { m.classList.add("dragging"); m.style.transform = `translateY(${dy}px)`; }
  });

  const finishTap = (e) => {
    if (!touchUI) return;
    const now = performance.now();
    const isDouble = now - lastTap < 320 && Math.hypot(e.clientX - tapX, e.clientY - tapY) < 44;
    if (isVideo) {
      if (isDouble) {
        clearTimeout(tapTimer); tapTimer = null; lastTap = 0;
        toggleVideoFS(stage);
      } else {
        lastTap = now; tapX = e.clientX; tapY = e.clientY;
        clearTimeout(tapTimer);
        tapTimer = setTimeout(() => {
          tapTimer = null;
          const vv = media();
          if (vv) { if (vv.paused) vv.play().catch(() => {}); else vv.pause(); }
        }, 290);
      }
      return;
    }
    if (isDouble) {
      lastTap = 0;
      eatClick();
      if (media()) { if (zoom().s > 1) resetZoom(); else zoomTo(2.5); }
    } else {
      lastTap = now; tapX = e.clientX; tapY = e.clientY;
    }
  };

  const up = (e) => {
    if (!pts.has(e.pointerId)) return;
    pts.delete(e.pointerId);
    if (!S.viewer) { mode = null; axis = null; return; }

    if (mode === "pinch") {
      if (pts.size < 2) {
        mode = null; pinch = null;
        if (zoom().s <= 1.02) resetZoom(); else setTransform(false);
      }
      return;
    }
    if (mode !== "drag" || pts.size) return;
    mode = null;

    const dx = e.clientX - sx, dy = e.clientY - sy;
    const axisWas = axis; axis = null;

    if (!axisWas) { finishTap(e); return; }
    eatClick();
    const m = media();
    if (!m) return;
    if (zoom().s > 1) {
      setTransform(false);
      return;
    }
    if (axisWas === "x" && Math.abs(dx) > 70) {
      const dir = dx < 0 ? 1 : -1;
      slideIn(dir);
      stepViewer(dir);
      return;
    }
    if (axisWas === "y" && dy > 110) { closeViewer(); return; }
    setTransform(false);
  };

  layer.addEventListener("pointerup", up);
  layer.addEventListener("pointercancel", (e) => {
    pts.delete(e.pointerId);
    mode = null; axis = null; pinch = null;
    if (S.viewer && zoom().s <= 1.02) setTransform(false);
  });

  if (!isVideo && !touchUI) {
    layer.addEventListener("dblclick", (e) => {
      if (e.target.closest("button, .vbar, .playlist")) return;
      if (zoom().s > 1) resetZoom(); else zoomTo(2.5);
    });
  }
}

function drawViewer() {
  const v = S.viewer;
  const it = v.list[v.idx];
  stopActiveHls();
  document.getElementById("viewer")?.remove();

  const isVideo = it.type === "video" || it.type === "micro-video";
  const canPlaylist = isVideo && v.vlist && v.vlist.length > 1;
  const plOpen = canPlaylist && localStorage.getItem("gh_playlist") !== "0";
  v.zoom = { s: 1, x: 0, y: 0 };
  let canPrev = true, canNext = true;
  if (isVideo && v.vlist && v.vlist.length) {
    const pos = v.vlist.findIndex(e => e.i === v.idx);
    canPrev = pos > 0;
    canNext = pos >= 0 && pos < v.vlist.length - 1;
  } else {
    canPrev = v.idx > 0;
    canNext = v.idx < v.list.length - 1;
  }

  const el = h(`
  <div class="overlay" id="viewer">
    <div class="viewer-top">
      <div class="t"><div class="trow"><b>${esc(it.fileName)}</b></div><span id="vsub">${fmtDate(it.timestampMs)} · ${fmtSize(it.size)}${it.width ? ` · ${it.width}×${it.height}` : ""}</span></div>
      <button class="btn icon ghost" id="fav" title="Favorite (star)" style="color:${it.isFavorite ? "var(--star)" : "var(--text-dim)"}">${ICONS.star}</button>
      ${it.type === "video" ? `<button class="btn icon ghost" id="wToggle" title="${it.watched ? "Watched: click to mark unwatched" : "Unwatched: click to mark watched"}" aria-label="Toggle watched" style="color:${it.watched ? "var(--text)" : "var(--text-dim)"}">${it.watched ? ICONS.eye : ICONS.eyeoff}</button>` : ""}
      <button class="btn icon ghost" id="addAl" title="Add to album">${ICONS.album}</button>
      ${inAlbumView() ? `<button class="btn icon ghost" id="rmAl" title="Remove from this album">${ICONS.rmalbum}</button>` : ""}
      ${canPlaylist ? `<button class="btn icon ghost ${plOpen ? "" : "off"}" id="plToggle" title="${plOpen ? "Hide playlist" : "Show playlist"}" aria-label="Toggle playlist">${ICONS.playlist}</button>` : ""}
      <button class="btn small" id="dl" title="Download original" aria-label="Download original">${ICONS.download} <span class="dl-txt">Download</span></button>
      <span class="vcount">${v.idx + 1} / ${v.list.length}</span>
      <button class="btn small close" id="close" title="Close (Esc)" aria-label="Close viewer"><span class="close-txt">✕ Close</span></button>
    </div>
    <div class="viewer-body">
      <div class="viewer-stage">
        ${canPrev ? `<button class="vnav l" id="vPrev" title="Previous (←)" aria-label="Previous">${ICONS.left}</button>` : ""}
        <div id="stage" style="width:100%;height:100%;display:flex;align-items:center;justify-content:center">
          <div class="spinner"></div>
        </div>
        ${canNext ? `<button class="vnav r" id="vNext" title="Next (→)" aria-label="Next">${ICONS.right}</button>` : ""}
      </div>
      ${plOpen ? buildPlaylistHTML(v, it) : ""}
    </div>
  </div>`);
  document.body.appendChild(el);

  const stage = el.querySelector("#stage");
  if (isVideo) startVideo(stage, it);
  else {
    const img = new Image();
    img.className = "viewer-img";
    img.src = thumbURL(it, 1600, false);
    img.onload = () => { stage.innerHTML = ""; stage.appendChild(img); };
    img.onerror = () => { stage.innerHTML = `<div class="muted">Preview unavailable — try downloading.</div>`; };
  }

  wireViewerGestures(el, isVideo);

  el.querySelector("#close").addEventListener("click", closeViewer);
  el.querySelector("#vPrev")?.addEventListener("click", () => stepViewer(-1));
  el.querySelector("#vNext")?.addEventListener("click", () => stepViewer(1));
  el.querySelector("#plToggle")?.addEventListener("click", () => {
    if (document.getElementById("vplaylist")) hidePlaylist();
    else showPlaylist();
  });
  wirePlaylist(el);
  if (plOpen) mountPlaylist(v);
  renderViewerAlbumBar(it);
  el.querySelector("#wToggle")?.addEventListener("click", () => setWatched(it, !it.watched, true));
  el.querySelector("#addAl").addEventListener("click", () => {
    addToAlbumModal([it.mediaKey], { inViewer: true });
  });
  el.querySelector("#rmAl")?.addEventListener("click", () => {
    confirmModal("Remove from album?", "This item will be removed from the current album. It stays in your Google Photos library.", async () => {
      try {
        const albumKey = S.route.arg;
        await api(`/api/albums/${encodeURIComponent(albumKey)}/remove`, { json: { mediaKeys: [it.mediaKey] } });
        toast("Removed from album", "ok");
        const i = S.items.findIndex(x => x.mediaKey === it.mediaKey);
        if (i >= 0) S.items.splice(i, 1);
        S.total = Math.max(0, S.total - 1);
        S.offset = Math.min(S.offset, S.items.length);
        closeViewer();
        if (S.gridEl) drawGrid(0);
        else afterMutation();
      } catch (err) { toast(err.message, "err"); }
    });
  });
  el.querySelector("#dl").addEventListener("click", () => {
    window.open(`/api/media/${encodeURIComponent(it.mediaKey)}/download?variant=original`, "_blank");
  });
  const favBtn = el.querySelector("#fav");
  favBtn.addEventListener("click", async () => {
    try {
      const nf = !it.isFavorite;
      await api(`/api/media/${encodeURIComponent(it.mediaKey)}/favorite`, { json: { favorite: nf } });
      it.isFavorite = nf;
      favBtn.style.color = nf ? "var(--star)" : "var(--text-dim)";
      toast(nf ? "Added to favorites" : "Removed from favorites", "ok");
    } catch (err) { toast(err.message, "err"); }
  });
  el.addEventListener("click", (e) => {
    const t = e.target;
    if (t === el || t.classList?.contains("viewer-stage") || t.id === "stage") closeViewer();
  });
}

function buildPlaylistHTML(v, current) {
  return `
  <aside class="playlist" id="vplaylist">
    <div class="pl-head">
      <b>Playlist</b><span>${v.vlist.length} video${v.vlist.length === 1 ? "" : "s"}</span>
      <button class="btn icon ghost small" id="plHide" title="Hide playlist">${ICONS.x}</button>
    </div>
    <div class="pl-body" id="plBody"></div>
  </aside>`;
}

const PL_MIN = 60;
const PL_BUF = 6;

function playlistRowsHTML(v, from, to) {
  let out = "";
  for (let p = from; p < to; p++) {
    const e = v.vlist[p];
    out += `
    <div class="pl-item ${e.i === v.idx ? "active" : ""}" data-i="${e.i}">
      <span class="pl-num">${p + 1}.</span>
      <span class="pl-name" title="${esc(e.x.fileName)}">${esc(e.x.fileName)}</span>
      <span class="pl-dur">${e.x.durationMs ? fmtDur(e.x.durationMs) : "—"}</span>
    </div>`;
  }
  return out;
}

function mountPlaylist(v) {
  const body = document.getElementById("plBody");
  if (!body) return;
  const n = v.vlist.length;
  if (n <= PL_MIN) {
    body.innerHTML = playlistRowsHTML(v, 0, n);
    S.plv = null;
    body.querySelector(".pl-item.active")?.scrollIntoView({ block: "nearest" });
    return;
  }
  body.innerHTML = `<div class="pl-spacer" id="plSpT" style="height:0"></div><div id="plWin"></div><div class="pl-spacer" id="plSpB" style="height:0"></div>`;
  const win = document.getElementById("plWin");
  win.innerHTML = playlistRowsHTML(v, 0, 2);
  const probe = win.firstElementChild;
  const rowH = probe && probe.offsetHeight > 10 ? probe.offsetHeight : 40;
  S.plv = { v, body, win, spT: document.getElementById("plSpT"), spB: document.getElementById("plSpB"), rowH, first: -1, last: -1 };
  plRenderWindow();
  plEnsureActive(true);
  let tick = false;
  body.addEventListener("scroll", () => {
    if (tick) return;
    tick = true;
    requestAnimationFrame(() => {
      tick = false;
      if (S.plv && S.plv.body === body && body.isConnected) plRenderWindow();
    });
  }, { passive: true });
}

function plRenderWindow() {
  const pv = S.plv;
  if (!pv || !pv.body.isConnected) return;
  const v = pv.v, n = v.vlist.length, rowH = pv.rowH;
  const vis = Math.ceil(pv.body.clientHeight / rowH) + 1;
  const firstRow = Math.max(0, Math.min(Math.floor(pv.body.scrollTop / rowH) - PL_BUF, n - 1));
  const lastRow = Math.min(n - 1, firstRow + vis + PL_BUF);
  const first = firstRow, last = lastRow + 1;
  if (first === pv.first && last === pv.last) return;
  pv.first = first; pv.last = last;
  pv.spT.style.height = (first * rowH) + "px";
  pv.spB.style.height = Math.max(0, (n - last) * rowH) + "px";
  pv.win.innerHTML = playlistRowsHTML(v, first, last);
}

function plEnsureActive(center) {
  const pv = S.plv;
  if (!pv) return;
  const pos = pv.v.vlist.findIndex(e => e.i === pv.v.idx);
  if (pos < 0) return;
  const target = pos * pv.rowH;
  const b = pv.body;
  if (center || target < b.scrollTop + b.clientHeight * 0.15 || target > b.scrollTop + b.clientHeight * 0.85) {
    b.scrollTop = Math.max(0, target - b.clientHeight / 2 + pv.rowH / 2);
  }
}

let gvResizeT = 0;
window.addEventListener("resize", () => {
  clearTimeout(gvResizeT);
  gvResizeT = setTimeout(() => {
    if (S.gv && S.gridEl?.isConnected && !S.duplicatesView && S.items.length > GV_MIN) {
      Object.assign(S.gv, gvMeasure(S.gridEl));
      S.gv.first = -1; S.gv.last = -1;
      gvRenderWindow(S.gridEl);
    }
    if (S.plv && S.plv.body.isConnected) {
      S.plv.win.innerHTML = playlistRowsHTML(S.plv.v, 0, 1);
      const probe = S.plv.win.firstElementChild;
      if (probe && probe.offsetHeight > 10) S.plv.rowH = probe.offsetHeight;
      S.plv.first = -1; S.plv.last = -1;
      plRenderWindow();
      plEnsureActive(true);
    }
  }, 160);
});

const WATCHED_FRACTION = 0.02;

async function setWatched(it, watched, manual = false) {
  if (!it || it.type !== "video" || !!it.watched === watched) return;
  const apply = (v) => {
    it.watched = v;
    for (const x of S.items) if (x.mediaKey === it.mediaKey) x.watched = v;
    const tile = S.gridEl && [...S.gridEl.querySelectorAll(".tile")].find(t => t.dataset.key === it.mediaKey);
    if (tile) tile.classList.toggle("is-watched", v);
    const btn = document.getElementById("wToggle");
    if (btn && S.viewer && S.viewer.list[S.viewer.idx] === it) {
      btn.innerHTML = v ? ICONS.eye : ICONS.eyeoff;
      btn.title = v ? "Watched: click to mark unwatched" : "Unwatched: click to mark watched";
      btn.style.color = v ? "var(--text)" : "var(--text-dim)";
    }
  };
  apply(watched);
  const video = document.querySelector("#viewer video");
  if (!watched && manual && video && video.__watch) video.__watch.suppressed = true;
  try {
    const r = await api("/api/media/watched", { json: { keys: [it.mediaKey], watched } });
    if (r && r.counts && S.state) { S.state.counts = r.counts; updateCountBadges(); }
    S.watchedDirty = true;
    if (manual) toast(watched ? "Marked as watched" : "Marked as unwatched", "ok", 1800);
  } catch (e) {
    apply(!watched);
    if (manual) toast(e.message, "err");
    throw e;
  }
}

function trackWatched(video, it) {
  const w = video.__watch = { started: false, suppressed: false, retryAt: 0 };
  const duration = () => (isFinite(video.duration) && video.duration > 0)
    ? video.duration : (it.durationMs ? it.durationMs / 1000 : 0);
  const check = () => {
    if (it.watched || w.suppressed || !w.started) return;
    const dur = duration();
    if (dur > 0 && video.currentTime >= dur * WATCHED_FRACTION && Date.now() >= w.retryAt) {
      setWatched(it, true).catch(() => { w.retryAt = Date.now() + 10000; });
    }
  };
  video.addEventListener("playing", () => { w.started = true; check(); });
  video.addEventListener("timeupdate", check);
  video.addEventListener("seeked", check);
  video.addEventListener("seeking", () => { w.suppressed = false; });
}

function mountVideoPlayer(stage, it) {
  stage.style.display = "flex";
  stage.style.flexDirection = "column";
  stage.style.alignItems = "stretch";
  stage.style.justifyContent = "stretch";
  stage.innerHTML = `
    <video playsinline></video>
    <div class="vbar">
      <button class="vbtn" id="vbBack" title="Back 5s (←)">${ICONS.back5}</button>
      <button class="vbtn" id="vbPlay" title="Play/Pause (Space)">${ICONS.pause}</button>
      <button class="vbtn" id="vbFwd" title="Forward 5s (→)">${ICONS.fwd5}</button>
      <span class="vtime" id="vbCur">0:00</span>
      <input type="range" id="vbSeek" min="0" max="1000" value="0" title="Seek (← → seek 5s)">
      <span class="vtime" id="vbDur">—</span>
      <button class="vbtn" id="vbMute" title="Mute">${ICONS.vol}</button>
      <input type="range" id="vbVol" class="vvol" min="0" max="1" step="0.01" value="1" title="Volume">
      <button class="vbtn" id="vbFs" title="Fullscreen (double-click video)">${ICONS.fs}</button>
    </div>`;
  const video = stage.querySelector("video");
  video.autoplay = true;
  if (it && it.type === "video") trackWatched(video, it);
  if (!matchMedia("(hover: none)").matches) {
    let clickTimer = null;
    video.addEventListener("click", () => {
      if (clickTimer) { clearTimeout(clickTimer); clickTimer = null; return; }
      clickTimer = setTimeout(() => {
        clickTimer = null;
        if (video.paused) video.play().catch(() => {}); else video.pause();
      }, 240);
    });
    video.addEventListener("dblclick", () => {
      if (clickTimer) { clearTimeout(clickTimer); clickTimer = null; }
      toggleVideoFS(stage);
    });
  } else {
    video.addEventListener("dblclick", () => toggleVideoFS(stage));
  }
  video.addEventListener("ended", () => {
    if (S.viewer) stepViewer(1);
  });

  const $ = id => stage.querySelector("#" + id);
  const seek = $("vbSeek"), cur = $("vbCur"), dur = $("vbDur");
  const playBtn = $("vbPlay"), muteBtn = $("vbMute"), vol = $("vbVol"), fsBtn = $("vbFs");

  const fmtT = sec => (isFinite(sec) && sec >= 0 ? fmtDur(Math.round(sec * 1000)) : "—");

  const setPlayIcon = () => { playBtn.innerHTML = video.paused ? ICONS.video : ICONS.pause; };
  const setMuteIcon = () => {
    muteBtn.innerHTML = video.muted || video.volume === 0 ? ICONS.mute : ICONS.vol;
    vol.value = video.muted ? 0 : video.volume;
  };

  video.addEventListener("play", setPlayIcon);
  video.addEventListener("pause", setPlayIcon);
  video.addEventListener("volumechange", setMuteIcon);
  setPlayIcon();
  setMuteIcon();

  let scrubbing = false;
  let held = false;
  const scrubTarget = () => (seek.value / 1000) * video.duration;
  let pending = null;
  video.__clearPending = () => { pending = null; };
  const syncTime = () => {
    const known = isFinite(video.duration) && video.duration > 0;
    if (pending) {
      const age = Date.now() - pending.at;
      if (Math.abs(video.currentTime - pending.t) < 3 && !video.seeking) pending = null;
      else if (age > 20000) pending = null;
      else if (!video.seeking && age > 1500 && pending.tries < 3) {
        pending.tries++; pending.at = Date.now();
        video.currentTime = pending.t;
      }
    }
    cur.textContent = fmtT(pending ? pending.t : video.currentTime);
    if (known) {
      dur.textContent = fmtT(video.duration);
      if (pending) { if (!scrubbing) seek.value = Math.round((pending.t / video.duration) * 1000); }
      else if (!scrubbing && !video.seeking) seek.value = Math.round((video.currentTime / video.duration) * 1000);
    }
  };
  const endScrub = () => {
    held = false;
    if (!scrubbing) return;
    scrubbing = false;
    if (isFinite(video.duration) && video.duration > 0) {
      const t = Math.min(scrubTarget(), video.duration);
      cur.textContent = fmtT(t);
      pending = { t, at: Date.now(), tries: 0 };
      video.currentTime = t;
    }
  };
  video.addEventListener("timeupdate", syncTime);
  video.addEventListener("durationchange", syncTime);
  video.addEventListener("loadedmetadata", syncTime);
  video.addEventListener("seeked", () => { if (!held) scrubbing = false; syncTime(); });

  playBtn.addEventListener("click", () => {
    if (video.paused) video.play().catch(() => {});
    else video.pause();
  });
  const seekBy = (delta) => {
    pending = null;
    if (isFinite(video.duration) && video.duration > 0) {
      video.currentTime = Math.min(Math.max(0, video.currentTime + delta), video.duration);
    } else {
      video.currentTime = Math.max(0, video.currentTime + delta);
    }
    showSeekFlash(delta > 0 ? "+5s" : "−5s");
  };
  $("vbBack").addEventListener("click", () => seekBy(-5));
  $("vbFwd").addEventListener("click", () => seekBy(5));
  seek.addEventListener("input", () => {
    if (!(isFinite(video.duration) && video.duration > 0)) return;
    scrubbing = true;
    cur.textContent = fmtT(scrubTarget());
  });
  seek.addEventListener("pointerdown", () => { held = true; scrubbing = true; });
  window.addEventListener("pointerup", () => { if (held) endScrub(); });
  window.addEventListener("pointercancel", () => { if (held) endScrub(); });
  seek.addEventListener("change", endScrub);
  seek.addEventListener("pointerup", endScrub);
  seek.addEventListener("keyup", endScrub);
  muteBtn.addEventListener("click", () => { video.muted = !video.muted; });
  vol.addEventListener("input", () => {
    video.volume = parseFloat(vol.value);
    video.muted = video.volume === 0;
  });
  fsBtn.addEventListener("click", () => toggleVideoFS(stage));
  stage.querySelectorAll(".vbar button").forEach(b =>
    b.addEventListener("click", () => b.blur()));
  return video;
}

async function startVideo(stage, it) {
  const video = mountVideoPlayer(stage, it);
  video.src = `/api/media/${encodeURIComponent(it.mediaKey)}/video`;
  video.addEventListener("error", () => { if (S.viewer) playHLS(stage, it); }, { once: true });
}

async function playHLS(stage, it) {
  if (!S.viewer) return;
  stopActiveHls();
  const video = mountVideoPlayer(stage, it);

  if (!window.Hls) {
    await new Promise((resolve, reject) => {
      const s = document.createElement("script");
      s.id = "hlsjs";
      s.src = "/hls.min.js";
      s.onload = resolve;
      s.onerror = reject;
      document.head.appendChild(s);
    }).catch(() => {
      stage.innerHTML = `<div class="muted">This video's codec isn't supported by this browser, and the HLS fallback failed to load.</div>`;
      return;
    });
  }
  if (!window.Hls || !S.viewer) return;

  if (window.Hls.isSupported()) {
    const hls = new window.Hls({ maxBufferLength: 30 });
    activeHls = hls;
    hls.loadSource(`/api/media/${encodeURIComponent(it.mediaKey)}/hls.m3u8`);
    hls.attachMedia(video);
    let mediaRetried = false, netRetries = 0;
    hls.on(window.Hls.Events.ERROR, (_, data) => {
      if (!data.fatal) return;
      if (activeHls !== hls) { try { hls.destroy(); } catch {  } return; }
      if (data.type === window.Hls.ErrorTypes.NETWORK_ERROR && netRetries < 3) {
        netRetries++;
        hls.startLoad();
        return;
      }
      if (data.type === window.Hls.ErrorTypes.MEDIA_ERROR && !mediaRetried) {
        mediaRetried = true;
        hls.recoverMediaError();
        return;
      }
      const st = document.getElementById("stage");
      if (st) st.innerHTML = `<div class="muted">Playback failed (${esc(data.details)}). Try downloading the original instead.</div>`;
      hls.destroy();
      activeHls = null;
    });
  } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
    video.src = `/api/media/${encodeURIComponent(it.mediaKey)}/hls.m3u8`;
  } else {
    stage.innerHTML = `<div class="muted">HLS not supported in this browser.</div>`;
  }
}

function renderSettings(content, titleEl) {
  titleEl.textContent = "Settings";
  const st = S.state || {};
  const el = h(`
  <div class="settings">
    <div class="panel">
      <h3>Account &amp; credentials <small>Android credential strings, same format as the desktop app</small></h3>
      <div id="accts"></div>
      <label class="field" style="margin-top:12px"><span>Add credential string</span>
        <textarea id="cred" placeholder="androidId=…&app=…&client_sig=…&Email=…&Token=…&lang=…&service=auth/photos.native"></textarea></label>
      <button class="btn primary" id="addCred">Add account</button>
    </div>

    <div class="panel">
      <h3>Library sync</h3>
      <div class="stat-row" id="libStats"></div>
      <div class="row" style="display:flex;gap:10px;align-items:center;flex-wrap:wrap">
        <button class="btn" id="syncInc">Sync changes</button>
        <button class="btn" id="syncFull">Full resync</button>
        <span class="grow" style="flex:1"></span>
        <label class="field" style="padding:0;min-width:190px"><span>Auto-sync</span>
          <select id="autoSyncSel">${autoSyncOptions(st.autosyncSec)}</select>
        </label>
        <button class="btn" id="autoSyncSave">Apply</button>
      </div>
      <p class="faint" style="margin-top:10px">Full resync re-walks your entire library (first sync does this automatically). “Sync changes” fetches only what changed — fast.</p>
    </div>

    <div class="panel">
      <h3>About</h3>
      <p class="muted">Photon ${esc(st.version || "")} · unofficial Google Photos client using the Android internal protocol.</p>
      <p class="faint">Runs anywhere — Windows, Linux, VPS. Config: <span class="mono">library-web.config</span> · Data: <span class="mono">${esc(st.dataDir || "data/")}</span></p>
    </div>
  </div>`);
  content.appendChild(el);

  drawAccounts(el.querySelector("#accts"));

  el.querySelector("#addCred").addEventListener("click", async () => {
    const cred = el.querySelector("#cred").value.trim();
    if (!cred) { toast("Paste a credential string first", "warn"); return; }
    try {
      await api("/api/account/credentials", { json: { authString: cred } });
      toast("Account added & selected", "ok");
      el.querySelector("#cred").value = "";
      await refreshState(); render();
    } catch (e) { toast(e.message, "err"); }
  });

  const stats = st.stats || {};
  el.querySelector("#libStats").innerHTML = `
    <div class="stat"><b>${stats.totalItems ?? 0}</b><span>items</span></div>
    <div class="stat"><b>${stats.videos ?? 0}</b><span>videos</span></div>
    <div class="stat"><b>${stats.photos ?? 0}</b><span>photos</span></div>
    <div class="stat"><b>${fmtSize(stats.totalSize)}</b><span>library size</span></div>
    <div class="stat"><b>${(stats.videoHours || 0).toFixed(1)} h</b><span>video runtime</span></div>`;

  el.querySelector("#syncInc").addEventListener("click", async () => {
    try { await api("/api/library/sync", { json: { full: false } }); toast("Sync started", "ok"); syncSoon(); }
    catch (e) { toast(e.message, "err"); }
  });
  el.querySelector("#syncFull").addEventListener("click", () => {
    confirmModal("Full resync?", "This re-reads your whole library from Google Photos. Useful if items look stale.", async () => {
      try { await api("/api/library/sync", { json: { full: true } }); toast("Full resync started", "ok"); syncSoon(); }
      catch (e) { toast(e.message, "err"); }
    });
  });
  el.querySelector("#autoSyncSave").addEventListener("click", async () => {
    const v = parseInt(el.querySelector("#autoSyncSel").value, 10);
    try {
      const res = await api("/api/library/autosync", { json: { intervalSec: v } });
      if (S.state) S.state.autosyncSec = res.intervalSec;
      toast(res.intervalSec < 0 ? "Auto-sync disabled" : "Auto-sync interval saved", "ok");
      const card = document.getElementById("syncCard");
      if (card) renderSyncCard(card);
    } catch (e) { toast(e.message, "err"); }
  });
}

async function drawAccounts(el) {
  let data;
  try { data = await api("/api/account"); } catch (e) { el.innerHTML = `<div class="muted">${esc(e.message)}</div>`; return; }
  el.innerHTML = "";
  const accounts = data.accounts || [];
  if (!accounts.length) {
    el.appendChild(h(`<div class="muted" style="margin-bottom:8px">No accounts yet. Capture a credential from your Android device (see the README on the project page) and paste it below.</div>`));
  }
  for (const a of accounts) {
    const row = h(`
    <div class="acct ${a.email === data.selected ? "selected" : ""}">
      <div class="avatar">${esc(a.email.slice(0, 1).toUpperCase())}</div>
      <div class="em"><b>${esc(a.email)}</b>
        <small>${a.email === data.selected ? "selected" : ""} ${a.needsTokenBinding ? '<span class="pill warn">needs token binding</span>' : ""}</small>
      </div>
      ${a.needsTokenBinding ? `<button class="btn small" data-a="bind">Bind via ADB</button>` : ""}
      ${a.email !== data.selected ? `<button class="btn small" data-a="select">Use</button>` : '<span class="pill">active</span>'}
      <button class="btn small danger" data-a="remove">${ICONS.trash}</button>
    </div>`);
    row.addEventListener("click", async (e) => {
      const btn = e.target.closest("[data-a]");
      if (!btn) return;
      const act = btn.dataset.a;
      try {
        if (act === "select") {
          const uploading = !!(S.state && S.state.upload && S.state.upload.running);
          if (uploading && !confirm("Web uploads are running. Switching accounts will cancel them. Switch anyway?")) return;
          await api("/api/account/select", { json: { email: a.email, force: uploading } });
          await onAccountChanged({ account: a.email, epoch: -1 }, { force: true });
          drawAccounts(el);
          return;
        }
        if (act === "remove") { await api(`/api/account/credentials?email=${encodeURIComponent(a.email)}`, { method: "DELETE" }); toast("Account removed", "ok"); }
        if (act === "bind") {
          btn.disabled = true;
          await api("/api/account/token-binding", { json: { email: a.email } });
          toast("Token binding alias attached", "ok");
        }
        await refreshState(); render();
      } catch (err) { toast(err.message, "err"); btn.disabled = false; }
    });
    el.appendChild(row);
  }
}

function modalEsc(e) { if (e.key === "Escape") closeModal(); }
function modal(html, mount) {
  const back = h(`<div class="modal-back"><div class="modal" role="dialog" aria-modal="true">${html}</div></div>`);
  $modalRoot.appendChild(back);
  back.addEventListener("click", (e) => { if (e.target === back) closeModal(); });
  document.addEventListener("keydown", modalEsc);
  const m = back.querySelector(".modal");
  mount && mount(m);
  const first = m.querySelector("input, select, textarea, button");
  if (first) first.focus();
  return back;
}
function closeModal() { $modalRoot.innerHTML = ""; document.removeEventListener("keydown", modalEsc); }

function confirmModal(title, text, onYes) {
  modal(`<h3>${esc(title)}</h3><p>${esc(text)}</p>
    <div class="row"><button class="btn" id="no">Cancel</button><button class="btn danger" id="yes">Confirm</button></div>`, (m) => {
    m.querySelector("#no").addEventListener("click", closeModal);
    m.querySelector("#yes").addEventListener("click", async () => { closeModal(); await onYes(); });
  });
}

function promptModal(title, value, onOk) {
  modal(`<h3>${esc(title)}</h3>
    <input type="text" id="v" value="${esc(value)}" placeholder="Name">
    <div class="row"><button class="btn" id="no">Cancel</button><button class="btn primary" id="yes">OK</button></div>`, (m) => {
    const input = m.querySelector("#v");
    input.focus(); input.select();
    const submit = async () => { const v = input.value.trim(); if (!v) return; closeModal(); await onOk(v); };
    m.querySelector("#no").addEventListener("click", closeModal);
    m.querySelector("#yes").addEventListener("click", submit);
    input.addEventListener("keydown", (e) => { if (e.key === "Enter") submit(); });
  });
}

function autoSyncLabel(sec) {
  if (sec < 0) return "Auto-sync off";
  if (!sec) sec = 120;
  return sec % 60 === 0 ? `Auto-sync every ${sec / 60} min` : `Auto-sync every ${sec} s`;
}

function autoSyncOptions(sec) {
  if (sec === 0 || sec == null) sec = 120;
  const opts = [
    [-1, "Off (manual sync only)"], [30, "Every 30 seconds"], [60, "Every minute"],
    [120, "Every 2 minutes"], [300, "Every 5 minutes"], [900, "Every 15 minutes"],
    [3600, "Every hour"],
  ];
  if (!opts.some(o => o[0] === sec)) opts.push([sec, `Every ${sec} seconds`]);
  opts.sort((a, b) => a[0] - b[0]);
  return opts.map(([v, l]) => `<option value="${v}" ${sec === v ? "selected" : ""}>${esc(l)}</option>`).join("");
}

function pollSync() {
  if (S.syncWatching) return;
  S.syncWatching = true;
  const tick = async () => {
    S.syncTimer = null;
    if (!S.me || !S.me.loggedIn) { S.syncWatching = false; return; }
    let delay = 10000;
    try {
      const card = document.getElementById("syncCard");
      const wasRunning = !!(S.state && S.state.sync && S.state.sync.running);
      const data = await api("/api/library/status");
      if (S.state) {
        S.state.sync = data.sync;
        S.state.counts = data.counts;
        S.state.stats = data.stats;
        S.state.autosyncSec = data.autosyncSec;
      }
      if (card) renderSyncCard(card);
      if (data.sync && data.sync.running) {
        delay = 1500;
      } else if (wasRunning) {
        if (data.sync.lastError) toast("Sync failed: " + data.sync.lastError, "err");
        else if (data.sync.itemsUpdated > 0 || data.sync.itemsDeleted > 0) {
          toast(`Library synced (+${data.sync.itemsUpdated} / −${data.sync.itemsDeleted})`, "ok");
          refreshAfterSync();
        } else syncAlbumsRefresh();
      }
    } catch {  }
    if (S.syncTimer === null && S.syncWatching) S.syncTimer = setTimeout(tick, delay);
  };
  S.syncTickFn = tick;
  S.syncTimer = setTimeout(tick, 400);
}

function syncSoon() {
  if (!S.syncWatching) { pollSync(); return; }
  clearTimeout(S.syncTimer);
  S.syncTimer = setTimeout(S.syncTickFn, 800);
}

function stopTimers() {
  clearTimeout(S.syncTimer); S.syncTimer = null;
  S.syncWatching = false;
  disconnectEvents();
}

function render() {
  S.watchedDirty = false;
  closeViewer();
  const contentFns = {
    library: renderLibrary,
    albums: renderAlbums,
    settings: renderSettings,
  };
  renderShell((content, title) => contentFns[S.route.view](content, title));
}

boot();
