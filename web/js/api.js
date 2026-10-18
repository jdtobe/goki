// Thin client for the goki JSON API.

export class ApiError extends Error {
  constructor(status, message, body) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function request(method, url, body) {
  const init = { method, headers: { Accept: "application/json" } };
  if (method !== "GET") init.headers["X-Goki"] = "1";
  if (body instanceof FormData) {
    init.body = body;
  } else if (body !== undefined) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(body);
  }
  const res = await fetch(url, init);
  if (res.status === 204) return null;
  let data = null;
  const text = await res.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { error: text };
    }
  }
  if (!res.ok) throw new ApiError(res.status, (data && data.error) || res.statusText, data);
  return data;
}

const q = (params) => new URLSearchParams(params).toString();

export const api = {
  config: () => request("GET", "/-/api/config"),
  tree: () => request("GET", "/-/api/tree"),
  /** Resolves a route; a missing page yields {kind:"missing"} instead of throwing. */
  page: async (path) => {
    try {
      return await request("GET", "/-/api/page?" + q({ path }));
    } catch (e) {
      if (e.status === 404 && e.body && e.body.kind === "missing") return e.body;
      throw e;
    }
  },
  save: (file, content, { create = false, baseHash = "" } = {}) =>
    request("PUT", "/-/api/page", { file, content, create, baseHash }),
  move: (from, to) => request("POST", "/-/api/move", { from, to }),
  remove: (path, recursive = false) => request("DELETE", "/-/api/entry?" + q({ path, recursive })),
  mkdir: (path) => request("POST", "/-/api/folder", { path }),
  search: (query, limit = 30) => request("GET", "/-/api/search?" + q({ q: query, limit })),
  upload: (dir, file) => {
    const fd = new FormData();
    fd.append("file", file, file.name || "pasted.png");
    return request("POST", "/-/api/upload?" + q({ dir }), fd);
  },
};

/** Subscribes to server-sent change events. Returns an unsubscribe function. */
export function subscribe(onChange, onStatus) {
  let es;
  let closed = false;
  const connect = () => {
    es = new EventSource("/-/api/events");
    es.addEventListener("hello", () => onStatus && onStatus(true));
    es.addEventListener("change", (e) => {
      try {
        onChange(JSON.parse(e.data));
      } catch {}
    });
    es.onerror = () => {
      onStatus && onStatus(false);
      if (es.readyState === EventSource.CLOSED && !closed) setTimeout(connect, 3000);
    };
  };
  connect();
  return () => {
    closed = true;
    es && es.close();
  };
}
