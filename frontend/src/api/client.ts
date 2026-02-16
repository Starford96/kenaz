import axios from "axios";

/** Base API client. Auth token injected via interceptor if configured. */
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE ?? "/api",
  headers: { "Content-Type": "application/json" },
});

// Optional Bearer token from env (for token auth mode).
const token = import.meta.env.VITE_AUTH_TOKEN as string | undefined;
if (token) {
  api.interceptors.request.use((cfg) => {
    cfg.headers.Authorization = `Bearer ${token}`;
    return cfg;
  });
}

export default api;
