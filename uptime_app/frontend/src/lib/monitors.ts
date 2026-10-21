import { authRequest } from "./session";

export type Monitor = { id: string; url: string; interval_seconds: number; created_at: string; status: "pending" | "success" | "error"; http_status: number; check_error: string; checked_at: string | null };
export const MONITOR_PAGE_SIZE = 20;
export const listMonitors = (before?: string) => authRequest<Monitor[]>(`/api/v1/monitors${before ? `?before=${encodeURIComponent(before)}` : ""}`);
export const createMonitor = (url: string, intervalSeconds: number) => authRequest<Monitor>("/api/v1/monitors", { url, interval_seconds: intervalSeconds });

export function formatInterval(seconds: number) {
  if (seconds % 3600 === 0) return `${seconds / 3600} ч`;
  if (seconds % 60 === 0) return `${seconds / 60} мин`;
  return `${seconds} сек`;
}

export const updateMonitor = (id: string, url: string, intervalSeconds: number) => authRequest<Monitor>(`/api/v1/monitors/${encodeURIComponent(id)}`, { url, interval_seconds: intervalSeconds }, "PUT");
export const deleteMonitor = (id: string) => authRequest<void>(`/api/v1/monitors/${encodeURIComponent(id)}`, undefined, "DELETE");

export function monitorStatus(monitor: Monitor): string {
  if (monitor.status === "pending") return "Ожидает первой проверки";
  if (monitor.status === "success") return "Успешно — HTTP 200";
  if (monitor.http_status) return `Ошибка — HTTP ${monitor.http_status}`;
  if (monitor.check_error === "timeout") return "Сайт не ответил за 10 секунд";
  if (monitor.check_error === "blocked_address") return "Проверка недоступна: требуется публичный IP-адрес";
  return "Не удалось подключиться к сайту";
}

export type HistoryBucket = { start: string; success: number; failure: number };
export type MonitorHistory = { monitor: Monitor; from: string; to: string; bucket_seconds: number; buckets: HistoryBucket[] };
export const getMonitorHistory = (id: string) => authRequest<MonitorHistory>(`/api/v1/monitors/${encodeURIComponent(id)}/history`);
