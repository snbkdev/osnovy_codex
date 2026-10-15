export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message); }
}
export async function request<T>(path: string, body?: unknown, bearer?: string, extraHeaders: Record<string, string> = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path.startsWith("/api/v1/") ? path : `/api/v1/auth/${path}`, {
      method: body === undefined ? "GET" : "POST", credentials: "same-origin", cache: "no-store",
      headers: { ...extraHeaders, ...(body === undefined || body instanceof FormData ? {} : { "Content-Type": "application/json" }), ...(bearer ? { Authorization: `Bearer ${bearer}` } : {}) },
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body), signal: AbortSignal.timeout(20000),
    });
  } catch { throw new ApiError(503, "service_unavailable", "Не удалось связаться с сервером. Попробуйте позже."); }
  const data = await response.json().catch(() => null);
  if (response.ok && data === null) throw new ApiError(503, "invalid_response", "Не удалось прочитать ответ сервера. Попробуйте позже.");
  if (!response.ok) throw new ApiError(response.status, data?.error?.code || "unknown_error", data?.error?.message || "Ошибка запроса.");
  return data as T;
}
