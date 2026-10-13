"use client";

import { useEffect } from "react";
import { initializeAuth } from "@/lib/auth";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => { initializeAuth(); }, []);
  return children;
}
