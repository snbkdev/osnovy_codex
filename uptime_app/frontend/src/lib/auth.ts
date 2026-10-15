export type User = { id: string; email: string; email_verified: boolean; full_name: string; avatar_url: string; date_of_birth: string | null };
export type TokenResponse = { access_token: string; token_type: "Bearer"; expires_in: number; user: User };
export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message); }
}
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
async function request<T>(path: string, body?: unknown, bearer?: string): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path.startsWith("/api/v1/") ? path : `/api/v1/auth/${path}`, {
      method: body === undefined ? "GET" : "POST", credentials: "same-origin", cache: "no-store",
      headers: { ...(body === undefined || body instanceof FormData ? {} : { "Content-Type": "application/json" }), ...(bearer ? { Authorization: `Bearer ${bearer}` } : {}) },
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body), signal: AbortSignal.timeout(20000),
    });
  } catch { throw new ApiError(503, "service_unavailable", "Не удалось связаться с сервером. Попробуйте позже."); }
  const data = await response.json().catch(() => null);
  if (!response.ok) throw new ApiError(response.status, data?.error?.code || "unknown_error", data?.error?.message || "Ошибка запроса.");
  return data as T;
}
export function refreshSession(): Promise<boolean> {
  if (refreshFlight) return refreshFlight;
  const started = generation;
  refreshFlight = sessionLock(async () => {
    if (started !== generation) return false;
    try {
      const data = await request<TokenResponse>("refresh", {});
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
export async function authRequest<T>(path: string, body?: unknown): Promise<T> {
  if (!accessToken && !(await refreshSession())) throw new ApiError(401, "unauthorized", "Войдите в аккаунт.");
  const usedToken = accessToken;
  try { return await request<T>(path, body, usedToken || undefined); }
  catch (err) {
    if (!(err instanceof ApiError) || err.status !== 401) throw err;
    // Another request may already have replaced this access token.
    if (accessToken === usedToken && !(await refreshSession())) throw err;
    return request<T>(path, body, accessToken || undefined);
  }
}
export const publicAuthRequest = <T = { message: string }>(path: string, body: unknown) => request<T>(path, body);

export async function saveProfile(fullName: string, dateOfBirth: string) {
  const started = generation;
  const user = await authRequest<User>("profile", { full_name: fullName, date_of_birth: dateOfBirth || null });
  if (started === generation && state.user?.id === user.id) update({ ...state, user });
  return user;
}

export async function saveAvatar(file: Blob) {
  const started = generation;
  const body = new FormData();
  body.append("avatar", file, "avatar.png");
  const user = await authRequest<User>("avatar", body);
  if (started === generation && state.user?.id === user.id) update({ ...state, user });
  return user;
}
