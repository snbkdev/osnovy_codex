"use client";

import { useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/http-client";
import { deleteMonitor, updateMonitor, createMonitor, formatInterval, listMonitors, MONITOR_PAGE_SIZE, type Monitor } from "@/lib/monitors";

export function MonitorDashboard() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [moreBusy, setMoreBusy] = useState(false);
  const [moreError, setMoreError] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [attempt, setAttempt] = useState(0);
  const [editing, setEditing] = useState<Monitor | null>(null);
  const [deleting, setDeleting] = useState<Monitor | null>(null);
  const returnFocus = useRef<HTMLButtonElement | null>(null);
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
    listMonitors().then((data) => { if (active) { setMonitors(data); setCursor(data.length === MONITOR_PAGE_SIZE ? data[data.length - 1].id : null); } })
      .catch((err: unknown) => { if (active) setLoadError(err instanceof Error ? err.message : "Не удалось загрузить сайты."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [attempt]);
  useEffect(() => { if (open) urlInput.current?.focus(); }, [open]);

  async function loadMore() {
    if (!cursor || moreBusy || busy || open || deleting) return;
    setMoreBusy(true); setMoreError("");
    try {
      const data = await listMonitors(cursor);
      setMonitors((current) => [...current, ...data.filter((item) => !current.some((existing) => existing.id === item.id))]);
      setCursor(data.length === MONITOR_PAGE_SIZE ? data[data.length - 1].id : null);
    } catch (err) { setMoreError(err instanceof Error ? err.message : "Не удалось загрузить сайты."); }
    finally { setMoreBusy(false); }
  }
  function closeForm() {
    setEditing(null); setOpen(false); setError(""); setURL(""); setAmount("5"); setUnit(60);
    requestAnimationFrame(() => {
      if (returnFocus.current?.isConnected) returnFocus.current.focus();
      else addButton.current?.focus();
    });
  }
  async function submit(event: React.SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    setError(""); setMessage("");
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
      const saved = editing ? await updateMonitor(editing.id, url.trim(), interval) : await createMonitor(url.trim(), interval);
      setMonitors((current) => editing ? current.map((item) => item.id === saved.id ? saved : item) : [saved, ...current]);
      setMessage(editing ? "Изменения сохранены." : "Точка мониторинга создана."); closeForm();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить сайт.");
      if (err instanceof ApiError && err.status === 404) await reloadList();
    }
    finally { setBusy(false); }
  }

  async function reloadList() {
    setCursor(null); setMoreError("");
    try {
      const data = await listMonitors();
      setMonitors(data); setLoadError("");
      setCursor(data.length === MONITOR_PAGE_SIZE ? data[data.length - 1].id : null);
    } catch (err) { setLoadError(err instanceof Error ? err.message : "Не удалось обновить список сайтов."); }
  }
  function startEdit(monitor: Monitor, button: HTMLButtonElement) {
    returnFocus.current = button; setEditing(monitor); setDeleting(null); setOpen(true);
    setURL(monitor.url);
    const seconds = monitor.interval_seconds;
    const nextUnit = seconds % 3600 === 0 ? 3600 : seconds % 60 === 0 ? 60 : 1;
    setUnit(nextUnit); setAmount(String(seconds / nextUnit)); setError(""); setMessage("");
    urlInput.current?.focus();
  }
  function cancelDelete() {
    setDeleting(null); setError("");
    requestAnimationFrame(() => {
      if (returnFocus.current?.isConnected) returnFocus.current.focus();
      else addButton.current?.focus();
    });
  }
  async function confirmDelete() {
    if (!deleting || busy) return;
    setBusy(true); setError(""); setMessage("");
    try {
      await deleteMonitor(deleting.id);
      setMonitors((current) => current.filter((item) => item.id !== deleting.id));
      setDeleting(null); setMessage("Сайт удалён.");
      await reloadList(); requestAnimationFrame(() => addButton.current?.focus());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить сайт.");
      if (err instanceof ApiError && err.status === 404) await reloadList();
    } finally { setBusy(false); }
  }

  return <div className="monitor-dashboard">
    <div className="monitor-heading">
      <div><h1>Мои сайты</h1><p>Создайте точки мониторинга и задайте частоту проверки.</p></div>
      <button ref={addButton} type="button" aria-expanded={open} aria-controls="create-monitor" disabled={loading || !!loadError || busy || moreBusy} onClick={(event) => { returnFocus.current = event.currentTarget; setEditing(null); setDeleting(null); setURL(""); setAmount("5"); setUnit(60); setError(""); setOpen(true); setMessage(""); }}>Добавить сайт</button>
    </div>
    {open && <section id="create-monitor" aria-labelledby="create-monitor-title" onKeyDown={(event) => { if (event.key === "Escape" && !busy) closeForm(); }}>
      <h2 id="create-monitor-title">{editing ? "Редактировать сайт" : "Добавить сайт"}</h2>
      <form onSubmit={submit}>
        <label htmlFor="monitor-url">URL сайта<input ref={urlInput} id="monitor-url" name="url" type="url" required maxLength={2048} placeholder="https://example.com" autoComplete="url" value={url} disabled={busy} onChange={(event) => setURL(event.target.value)} /></label>
        <div className="monitor-interval-fields">
          <label htmlFor="monitor-interval">Проверять каждые<input id="monitor-interval" name="interval" type="number" required min={Math.ceil(5 / unit)} max={Math.floor(604800 / unit)} step="1" value={amount} disabled={busy} onChange={(event) => setAmount(event.target.value)} aria-describedby="monitor-interval-hint" /></label>
          <label htmlFor="monitor-unit">Единица времени<select id="monitor-unit" value={unit} disabled={busy} onChange={(event) => setUnit(Number(event.target.value))}><option value={1}>Секунды</option><option value={60}>Минуты</option><option value={3600}>Часы</option></select></label>
        </div>
        <p id="monitor-interval-hint" className="field-hint">От 5 секунд до 7 дней. Например, два раза за 10 минут — это каждые 5 минут.</p>
        <div className="monitor-form-actions"><button type="submit" disabled={busy}>{busy ? (editing ? "Сохраняем…" : "Создаём…") : (editing ? "Сохранить" : "Создать точку")}</button><button type="button" className="secondary-button" disabled={busy} onClick={closeForm}>Отмена</button></div>
        {error && <p role="alert">{error}</p>}
      </form>
    </section>}
    {deleting && <section aria-label="Подтверждение удаления" onKeyDown={(event) => { if (event.key === "Escape" && !busy) cancelDelete(); }}>
      <h2>Удалить сайт?</h2><p>{deleting.url}</p><p>Точка мониторинга будет удалена без возможности восстановления.</p>
      <div className="monitor-form-actions"><button autoFocus type="button" className="secondary-button" disabled={busy} onClick={cancelDelete}>Отмена</button><button type="button" disabled={busy} onClick={() => { void confirmDelete(); }}>{busy ? "Удаляем…" : "Удалить"}</button></div>
      {error && <p role="alert">{error}</p>}
    </section>}
    {message && <p role="status">{message}</p>}
    {loading ? <p role="status">Загружаем сайты…</p> : loadError ? <section><p role="alert">{loadError}</p><button type="button" onClick={() => { setLoading(true); setLoadError(""); setAttempt((value) => value + 1); }}>Повторить загрузку</button></section> : <section aria-label="Точки мониторинга">
      {monitors.length === 0 ? <p>Сайтов пока нет. Нажмите «Добавить сайт», чтобы создать первую точку.</p> : <ul className="monitor-list">{monitors.map((monitor) => <li key={monitor.id}>
        <a href={monitor.url} target="_blank" rel="noopener noreferrer">{monitor.url}</a>
        <p>Интервал: {formatInterval(monitor.interval_seconds)}</p>
        <div className="monitor-form-actions">
          <button type="button" className="secondary-button" disabled={busy || moreBusy} aria-label={`Редактировать ${monitor.url}`} onClick={(event) => startEdit(monitor, event.currentTarget)}>Редактировать</button>
          <button type="button" className="secondary-button" disabled={busy || moreBusy} aria-label={`Удалить ${monitor.url}`} onClick={(event) => { returnFocus.current = event.currentTarget; setOpen(false); setEditing(null); setDeleting(monitor); setError(""); setMessage(""); }}>Удалить</button>
        </div>
        <span className="monitor-status">Ожидает запуска мониторинга</span>
      </li>)}</ul>}
      {cursor && <button type="button" disabled={moreBusy || busy || open || !!deleting} onClick={() => { void loadMore(); }}>{moreBusy ? "Загружаем…" : "Загрузить ещё"}</button>}
      {moreError && <p role="alert">{moreError}</p>}
    </section>}
  </div>;
}
