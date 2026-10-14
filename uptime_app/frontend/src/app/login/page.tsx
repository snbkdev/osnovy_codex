"use client";

import { useEffect, useSyncExternalStore } from "react";
import { useRouter } from "next/navigation";
import { AuthForm } from "@/components/auth-form";
import { getAuthState, getServerAuthState, subscribeAuth } from "@/lib/auth";

export default function Page() {
  const router = useRouter();
  const { user, loading } = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
  useEffect(() => {
    if (!loading && user) router.replace("/");
  }, [loading, user, router]);
  if (loading || user) return <main><p role="status">{loading ? "Проверяем сессию…" : "Открываем дэшборд…"}</p></main>;
  return <AuthForm mode="login" />;
}
