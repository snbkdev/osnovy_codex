"use client";

import Link from "next/link";
import { useEffect, useSyncExternalStore } from "react";
import { useRouter } from "next/navigation";
import { DashboardHeader } from "@/components/dashboard-header";
import { ProfileForm } from "@/components/profile-form";
import { getAuthState, getServerAuthState, refreshSession, subscribeAuth } from "@/lib/auth";

export default function ProfilePage() {
  const state = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
  const router = useRouter();
  useEffect(() => {
    if (!state.loading && !state.user && !state.error) router.replace("/login");
  }, [state.loading, state.user, state.error, router]);
  if (!state.user) return <main>
    {state.error ? <><p role="alert">{state.error}</p><button onClick={() => { void refreshSession(); }}>Повторить</button><p><Link href="/login">Перейти ко входу</Link></p></> : <p role="status">Проверяем сессию…</p>}
  </main>;
  return <div className="dashboard">
    <DashboardHeader />
    <main className="dashboard-content">
      <Link href="/">← На главную</Link>
      <h1>Мой профиль</h1>
      <ProfileForm key={state.user.id} />
    </main>
  </div>;
}
