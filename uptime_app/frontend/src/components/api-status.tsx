"use client";

import { useState } from "react";
import { checkHealth } from "@/lib/api";

export function ApiStatus() {
 const [status, setStatus] = useState("Соединение ещё не проверено");
 const [loading, setLoading] = useState(false);
 async function check() {
  setLoading(true);
  try { setStatus(await checkHealth() ? "API доступен" : "Неожиданный ответ API"); }
  catch { setStatus("API недоступен. Убедитесь, что backend запущен."); }
  finally { setLoading(false); }
 }
 return <section aria-label="Статус API">
  <p role="status">{status}</p>
  <button onClick={check} disabled={loading}>{loading ? "Проверяем…" : "Проверить API"}</button>
 </section>;
}
