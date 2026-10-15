/* eslint-disable @next/next/no-img-element -- Same-origin local avatar files. */
import type { User } from "@/lib/auth";

export function Avatar({ user }: { user: User }) {
  return <span className="dashboard-avatar" aria-hidden="true">
    {user.avatar_url ? <img src={user.avatar_url} alt="" width={48} height={48} /> : (user.full_name || user.email)[0]?.toUpperCase()}
  </span>;
}
