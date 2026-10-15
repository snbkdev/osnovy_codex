"use client";

import { useEffect, useRef, useState } from "react";
import { createMonitor, formatInterval, listMonitors, type Monitor } from "@/lib/monitors";

export function MonitorDashboard() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [attempt, setAttempt] = useState(0);
  const [open, setOpen] = useState(false);
  const [url, setURL] = useState("");
  const [amount, setAmount] = useState("5");
  const [unit, setUnit] = useState(60);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const addButton = useRef<HTMLButtonElement>(null);
  const urlInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let active = true;
    listMonitors().then((data) => { if (active) setMonitors(data); })
      .catch((err: unknown) => { if (active) setLoadError(err instanceof Error ? err.message : "Не удалось загрузить сайты."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [attempt]);
  useEffect(() => { if (open) urlInput.current?.focus(); }, [open]);

  function closeForm() {
    setOpen(false); setError(""); setURL(""); setAmount("5"); setUnit(60);
    addButton.current?.focus();
  }
  async function submit(event: React.SubmitEvent<HTMLFormElement>) {
    event.preventDefault(); setError(""); setMessage("");
    const interval = Number(amount) * unit;
    if (!Number.isInteger(Number(amount)) || !Number.isInteger(interval) || interval < 5 || interval > 604800) {
      setError("Укажите целый интервал от 5 секунд до 7 дней."); return;
    }
    let parsed: URL;
    try { parsed = new URL(url.trim()); }
    catch { setError("Укажите полный URL, например https://example.com."); return; }
    if (!["http:", "https:"].includes(parsed.protocol) || parsed.username || parsed.password) {
      setError("Используйте HTTP или HTTPS URL без логина и пароля."); return;
    }
    setBusy(true);
    try {
      const created = await createMonitor(url.trim(), interval);
      setMonitors((current) => [created, ...current]);
      setMessage("Точка мониторинга создана."); closeForm();
    } catch (err) { setError(err instanceof Error ? err.message : "Не удалось добавить сайт."); }
    finally { setBusy(false); }
  }

  return <div className="monitor-dashboard">
    <div className="monitor-heading">
      <div><h1>Мои сайты</h1><p>Создайте точки мониторинга и задайте частоту проверки.</p></div>
      <button ref={addButton} type="button" aria-expanded={open} aria-controls="create-monitor" disabled={loading || !!loadError || busy} onClick={() => { setOpen(true); setMessage(""); }}>Добавить сайт</button>
    </div>
    {open && <section id="create-monitor" aria-labelledby="create-monitor-title" onKeyDown={(event) => { if (event.key === "Escape" && !busy) closeForm(); }}>
      <h2 id="create-monitor-title">Добавить сайт</h2>
      <form onSubmit={submit}>
        <label htmlFor="monitor-url">URL сайта<input ref={urlInput} id="monitor-url" name="url" type="url" required maxLength={2048} placeholder="https://example.com" autoComplete="url" value={url} disabled={busy} onChange={(event) => setURL(event.target.value)} /></label>
        <div className="monitor-interval-fields">
          <label htmlFor="monitor-interval">Проверять каждые<input id="monitor-interval" name="interval" type="number" required min={Math.ceil(5 / unit)} max={Math.floor(604800 / unit)} step="1" value={amount} disabled={busy} onChange={(event) => setAmount(event.target.value)} aria-describedby="monitor-interval-hint" /></label>
          <label htmlFor="monitor-unit">Единица времени<select id="monitor-unit" value={unit} disabled={busy} onChange={(event) => setUnit(Number(event.target.value))}><option value={1}>Секунды</option><option value={60}>Минуты</option><option value={3600}>Часы</option></select></label>
        </div>
        <p id="monitor-interval-hint" className="field-hint">От 5 секунд до 7 дней. Например, два раза за 10 минут — это каждые 5 минут.</p>
        <div className="monitor-form-actions"><button type="submit" disabled={busy}>{busy ? "Создаём…" : "Создать точку"}</button><button type="button" className="secondary-button" disabled={busy} onClick={closeForm}>Отмена</button></div>
        {error && <p role="alert">{error}</p>}
      </form>
    </section>}
    {message && <p role="status">{message}</p>}
    {loading ? <p role="status">Загружаем сайты…</p> : loadError ? <section><p role="alert">{loadError}</p><button type="button" onClick={() => { setLoading(true); setLoadError(""); setAttempt((value) => value + 1); }}>Повторить загрузку</button></section> : <section aria-label="Точки мониторинга">
      {monitors.length === 0 ? <p>Сайтов пока нет. Нажмите «Добавить сайт», чтобы создать первую точку.</p> : <ul className="monitor-list">{monitors.map((monitor) => <li key={monitor.id}>
        <a href={monitor.url} target="_blank" rel="noopener noreferrer">{monitor.url}</a>
        <p>Интервал: {formatInterval(monitor.interval_seconds)}</p>
        <span className="monitor-status">Ожидает запуска мониторинга</span>
      </li>)}</ul>}
    </section>}
  </div>;
}
