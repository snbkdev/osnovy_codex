"use client";

import Link from "next/link";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useRouter } from "next/navigation";
import { getAuthState, getServerAuthState, logout, refreshSession, subscribeAuth } from "@/lib/auth";

export function AuthNav() {
  const router = useRouter();
  const menu = useRef<HTMLDetailsElement>(null);
  const state = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    function closeOutside(event: PointerEvent) {
      if (menu.current && event.target instanceof Node && !menu.current.contains(event.target)) menu.current.open = false;
    }
    document.addEventListener("pointerdown", closeOutside);
    return () => document.removeEventListener("pointerdown", closeOutside);
  }, []);
  async function exit() {
    setBusy(true); setError("");
    try { await logout(); router.replace("/login"); } catch (err) { setError(err instanceof Error ? err.message : "Не удалось выйти."); }
    finally { setBusy(false); }
  }
  if (state.user) return <div className="dashboard-account">
   <div className="dashboard-account-controls">
    <details className="dashboard-profile" ref={menu} onKeyDown={(event) => {
    if (event.key === "Escape" && menu.current) {
      menu.current.open = false;
      menu.current.querySelector("summary")?.focus();
    }
  }}>
    <summary aria-label={`Профиль пользователя ${state.user.email}`}>
      <span className="dashboard-avatar" aria-hidden="true">{(state.user.full_name || state.user.email)[0].toUpperCase()}</span>
      <span className="dashboard-profile-label">{state.user.full_name || "Профиль"}<span>{state.user.email}</span></span>
      <span aria-hidden="true">⌄</span>
    </summary>
    <div className="dashboard-profile-panel">
      <h2>Профиль пользователя</h2>
      <p className="dashboard-profile-email">{state.user.email}</p>
      <p className="dashboard-profile-status">{state.user.email_verified ? "Email подтверждён" : "Email не подтверждён"}</p>
      <Link href="/profile" onClick={() => { if (menu.current) menu.current.open = false; }}>Редактировать профиль</Link>
    </div>
    </details>
    <button className="dashboard-logout" disabled={busy} onClick={exit}>{busy ? "Выходим…" : "Выйти"}</button>
   </div>
   {error && <p role="alert">{error}</p>}
  </div>;
  return <section aria-label="Аккаунт">
    {state.loading ? <p role="status">Восстанавливаем сессию…</p> : <nav><Link href="/login">Войти</Link><Link href="/register">Зарегистрироваться</Link></nav>}
    {(error || state.error) && <p role="alert">{error || state.error}</p>}
    {state.error && <button onClick={() => { void refreshSession(); }}>Повторить восстановление сессии</button>}
  </section>;
}
