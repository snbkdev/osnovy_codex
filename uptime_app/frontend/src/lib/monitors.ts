import { authRequest } from "./session";

export type Monitor = { id: string; url: string; interval_seconds: number; created_at: string };
export const MONITOR_PAGE_SIZE = 20;
export const listMonitors = (before?: string) => authRequest<Monitor[]>(`/api/v1/monitors${before ? `?before=${encodeURIComponent(before)}` : ""}`);
export const createMonitor = (url: string, intervalSeconds: number) => authRequest<Monitor>("/api/v1/monitors", { url, interval_seconds: intervalSeconds });

export function formatInterval(seconds: number) {
  if (seconds % 3600 === 0) return `${seconds / 3600} ч`;
  if (seconds % 60 === 0) return `${seconds / 60} мин`;
  return `${seconds} сек`;
}
