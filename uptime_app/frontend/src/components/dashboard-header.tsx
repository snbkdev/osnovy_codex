import Link from "next/link";
import { AuthNav } from "@/components/auth-nav";

export function DashboardHeader() {
  return <header className="dashboard-header">
    <Link className="dashboard-brand" href="/" aria-label="Uptime App — главная">UPTIME APP</Link>
    <AuthNav />
  </header>;
}
