"use client";

import Link from "next/link";
import { useSyncExternalStore } from "react";
import { AuthNav } from "@/components/auth-nav";
import { getAuthState, getServerAuthState, subscribeAuth } from "@/lib/auth";

export default function Home() {
 const { user } = useSyncExternalStore(subscribeAuth, getAuthState, getServerAuthState);
 return <div className={user ? "dashboard" : undefined}>
  {user && <header className="dashboard-header">
   <Link className="dashboard-brand" href="/" aria-label="Uptime App — главная">UPTIME APP</Link>
   <AuthNav />
  </header>}
  <main className={user ? "dashboard-content" : undefined} aria-label="Содержимое дэшборда">
   {!user && <AuthNav />}
  </main>
 </div>;
}
