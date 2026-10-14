"use client";

import { useSyncExternalStore } from "react";
import { AuthNav } from "@/components/auth-nav";
import { DashboardHeader } from "@/components/dashboard-header";
import { getAuthState, getServerAuthState, subscribeAuth } from "@/lib/auth";

export default function Home() {
 const { user } = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
 return <div className={user ? "dashboard" : undefined}>
  {user && <DashboardHeader />}
  <main className={user ? "dashboard-content" : undefined} aria-label="Содержимое дэшборда">
   {!user && <AuthNav />}
  </main>
 </div>;
}
