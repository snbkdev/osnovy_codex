"use client";

import Link from "next/link";
import { useState, useSyncExternalStore } from "react";
import { getAuthState, getServerAuthState, logout, refreshSession, subscribeAuth } from "@/lib/auth";

export function AuthNav() {
  const state = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function exit() {
    setBusy(true); setError("");
    try { await logout(); } catch (err) { setError(err instanceof Error ? err.message : "Не удалось выйти."); }
    finally { setBusy(false); }
  }
  return <section aria-label="Аккаунт">
    {state.loading ? <p role="status">Восстанавливаем сессию…</p> : state.user ? <>
      <p>Вы вошли как <strong>{state.user.email}</strong></p>
      <button disabled={busy} onClick={exit}>{busy ? "Выходим…" : "Выйти"}</button>
    </> : <nav><Link href="/login">Войти</Link><Link href="/register">Зарегистрироваться</Link></nav>}
    {(error || state.error) && <p role="alert">{error || state.error}</p>}
    {state.error && <button onClick={() => { void refreshSession(); }}>Повторить восстановление сессии</button>}
  </section>;
}
