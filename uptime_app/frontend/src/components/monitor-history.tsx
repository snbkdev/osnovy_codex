"use client";

import Link from "next/link";
import { useEffect, useState, useSyncExternalStore } from "react";
import { useRouter } from "next/navigation";
import { DashboardHeader } from "@/components/dashboard-header";
import { getAuthState, getServerAuthState, refreshSession, subscribeAuth } from "@/lib/auth";
import { getMonitorHistory, type HistoryBucket, type MonitorHistory } from "@/lib/monitors";

const dateTime = (value: number | string) => new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });

function AvailabilityChart({ history }: { history: MonitorHistory }) {
  const [selected, setSelected] = useState<number | null>(null);
  const step = history.bucket_seconds * 1000;
  const from = new Date(history.from).getTime();
  const to = new Date(history.to).getTime();
  const first = Math.floor(from / step) * step;
  const byTime = new Map(history.buckets.map((bucket) => [new Date(bucket.start).getTime(), bucket]));
  const buckets: HistoryBucket[] = [];
  for (let start = first; start <= to; start += step) {
    buckets.push(byTime.get(start) ?? { start: new Date(start).toISOString(), success: 0, failure: 0 });
  }
  const total = history.buckets.reduce((sum, bucket) => sum + bucket.success + bucket.failure, 0);
  const success = history.buckets.reduce((sum, bucket) => sum + bucket.success, 0);
  function describe(bucket: HistoryBucket) {
    const start = new Date(bucket.start).getTime();
    const period = `${dateTime(Math.max(start, from))} — ${dateTime(Math.min(start + step, to))}`;
    return `${period}: ${bucket.success + bucket.failure ? `успешных ${bucket.success}, ошибок ${bucket.failure}` : "нет проверок"}`;
  }
  const detail = selected !== null ? buckets.find((bucket) => new Date(bucket.start).getTime() === selected) : undefined;
  return <section className="history-card" aria-labelledby="availability-title">
    <h2 id="availability-title">Доступность за последние 24 часа</h2>
    <p className="history-summary">{total ? `Успешных проверок: ${success} из ${total} (${(success / total * 100).toFixed(1)}%). Ошибок: ${total - success}.` : "Проверок за этот период пока нет. История появится после ближайшей автоматической проверки."}</p>
    <ul className="history-legend" aria-label="Обозначения графика">
      <li><span className="history-key history-success" />Работал — HTTP 200</li>
      <li><span className="history-key history-failure" />Не работал — ошибка</li>
      <li><span className="history-key history-unknown" />Нет проверок</li>
    </ul>
    <p id="history-hint" className="history-hint">Каждый столбец — 5 минут. Зелёная и красная части показывают долю успешных и неуспешных проверок. Нажмите на столбец, чтобы увидеть время и количество проверок.</p>
    <div className="history-scroll" role="region" aria-label="График истории проверок, прокрутка по горизонтали" tabIndex={0}>
      <div className="history-plot">
        <div className="history-bars" role="group" aria-label="Проверки по времени" aria-describedby="history-hint">
          {buckets.map((bucket) => {
            const count = bucket.success + bucket.failure;
            const start = new Date(bucket.start).getTime();
            return <button key={bucket.start} type="button" className={`history-bar${count ? "" : " history-unknown"}`} title={describe(bucket)} aria-label={describe(bucket)} aria-pressed={selected === start} onClick={() => setSelected(start)}>
              {count > 0 && <><span className="history-failure" style={{ height: `${bucket.failure / count * 100}%` }} /><span className="history-success" style={{ height: `${bucket.success / count * 100}%` }} /></>}
            </button>;
          })}
        </div>
        <div className="history-axis" aria-hidden="true"><span>{dateTime(from)}</span><span>{dateTime((from + to) / 2)}</span><span>{dateTime(to)}</span></div>
      </div>
    </div>
    <p className="history-detail" role="status">{detail ? describe(detail) : "Выберите интервал на графике для подробностей."}</p>
    <p className="history-hint">Время указано в вашем часовом поясе. График показывает результаты проверок, а не точную длительность простоя между ними. Показана история текущего URL; данные обновляются каждые 10 секунд.</p>
  </section>;
}

function HistoryContent({ id }: { id: string }) {
  const [history, setHistory] = useState<MonitorHistory | null>(null);
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    let timer: ReturnType<typeof setTimeout>;
    async function load() {
      try {
        const data = await getMonitorHistory(id);
        if (active) { setHistory(data); setError(""); }
      } catch (err) {
        if (active) setError(err instanceof Error ? err.message : "Не удалось загрузить историю.");
      } finally {
        if (active) timer = setTimeout(() => { void load(); }, 10000);
      }
    }
    void load();
    return () => { active = false; clearTimeout(timer); };
  }, [id, attempt]);
  return <>
    <h1>История сайта</h1>
    {history && <p className="history-url"><a href={history.monitor.url} target="_blank" rel="noopener noreferrer">{history.monitor.url}</a></p>}
    {error && <div role="alert"><p>{error} {history ? "На графике — последние загруженные данные." : ""}</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Повторить загрузку</button></div>}
    {!history && !error && <p role="status">Загружаем историю…</p>}
    {history && <AvailabilityChart history={history} />}
  </>;
}

export function MonitorHistoryPage({ id }: { id: string }) {
  const state = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
  const router = useRouter();
  useEffect(() => {
    if (!state.loading && !state.user && !state.error) router.replace("/login");
  }, [state.loading, state.user, state.error, router]);
  if (!state.user) return <main>
    {state.error ? <><p role="alert">{state.error}</p><button onClick={() => { void refreshSession(); }}>Повторить</button><p><Link href="/login">Перейти ко входу</Link></p></> : <p role="status">Проверяем сессию…</p>}
  </main>;
  return <div className="dashboard"><DashboardHeader /><main className="dashboard-content">
    <Link href="/">← Мои сайты</Link>
    <HistoryContent key={`${state.user.id}:${id}`} id={id} />
  </main></div>;
}
