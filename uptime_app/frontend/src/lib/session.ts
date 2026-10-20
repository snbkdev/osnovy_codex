import { ApiError, request } from "./http-client";
import type { User, TokenResponse } from "./auth-types";

type State = { user: User | null; loading: boolean; error: string | null };
let state: State = { user: null, loading: true, error: null };
let accessToken: string | null = null;
let refreshFlight: Promise<boolean> | null = null;
let initialized = false;
let generation = 0;
let channel: BroadcastChannel | null = null;
const listeners = new Set<() => void>();
const serverState: State = { user: null, loading: true, error: null };
export const getAuthState = () => state;
export const getServerAuthState = () => serverState;
export function subscribeAuth(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener); }; }
function update(next: State) { state = next; listeners.forEach((fn) => fn()); }
function clear(error: string | null = null) { accessToken = null; update({ user: null, loading: false, error }); }
function accept(data: TokenResponse) { accessToken = data.access_token; update({ user: data.user, loading: false, error: null }); }
async function sessionLock<T>(work: () => Promise<T>): Promise<T> {
  if (typeof navigator === "undefined" || !navigator.locks) throw new ApiError(503, "locks_unavailable", "Для безопасного обновления сессии нужен браузер с Web Locks и HTTPS (либо localhost).");
  return navigator.locks.request("uptime-auth-session", work);
}
export function refreshSession(): Promise<boolean> {
  if (refreshFlight) return refreshFlight;
  const started = generation;
  refreshFlight = sessionLock(async () => {
    if (started !== generation) return false;
    try {
      const bytes = crypto.getRandomValues(new Uint8Array(32));
      const attempt = btoa(String.fromCharCode(...bytes)).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
      const headers = { "X-Uptime-Refresh-Request": attempt };
      let data: TokenResponse;
      try { data = await request<TokenResponse>("refresh", {}, undefined, headers); }
      catch (err) {
        if (!(err instanceof ApiError) || ![503, 504].includes(err.status)) throw err;
        // Only refresh is retried, with the same key while holding the cross-tab lock.
        data = await request<TokenResponse>("refresh", {}, undefined, headers);
      }
      if (started !== generation) return false;
      accept(data); return true;
    } catch (err) {
      if (started === generation) clear(err instanceof ApiError && err.status === 401 ? null : err instanceof Error ? err.message : "Ошибка восстановления сессии.");
      return false;
    }
  }).catch((err: unknown) => {
    if (started === generation) clear(err instanceof Error ? err.message : "Ошибка восстановления сессии.");
    return false;
  }).finally(() => { refreshFlight = null; });
  return refreshFlight;
}
export function initializeAuth() {
  if (initialized) return;
  initialized = true;
  if (typeof BroadcastChannel !== "undefined") {
    channel = new BroadcastChannel("uptime-auth");
    channel.onmessage = (event) => {
      if (event.data === "logout") { generation++; clear(); }
      if (event.data === "login") {
        generation++; clear();
        // Wait for a stale in-flight refresh before starting the new generation.
        if (refreshFlight) void refreshFlight.then(() => refreshSession());
        else void refreshSession();
      }
    };
  }
  void refreshSession();
}
export async function login(email: string, password: string) {
  await sessionLock(async () => {
    const data = await request<TokenResponse>("login", { email, password });
    generation++; accept(data); channel?.postMessage("login");
  });
}
export async function logout() {
  await sessionLock(async () => {
    await request("logout", {}); generation++; clear(); channel?.postMessage("logout");
  });
}
export async function authRequest<T>(path: string, body?: unknown, method?: "GET" | "POST" | "PUT" | "DELETE"): Promise<T> {
  if (!accessToken && !(await refreshSession())) throw new ApiError(401, "unauthorized", "Войдите в аккаунт.");
  const usedToken = accessToken;
  try { return await request<T>(path, body, usedToken || undefined, {}, method); }
  catch (err) {
    if (!(err instanceof ApiError) || err.status !== 401) throw err;
    // Another request may already have replaced this access token.
    if (accessToken === usedToken && !(await refreshSession())) throw err;
    return request<T>(path, body, accessToken || undefined, {}, method);
  }
}
export const publicAuthRequest = <T = { message: string }>(path: string, body: unknown) => request<T>(path, body);

export async function updateCurrentUser(work: () => Promise<User>): Promise<User> {
  const started = generation;
  const user = await work();
  if (started === generation && state.user?.id === user.id) update({ ...state, user });
  return user;
}
