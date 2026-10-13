import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/components/auth-provider";

export const metadata: Metadata = {
 title: "Uptime App",
 description: "Мониторинг доступности сервисов",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
 return <html lang="ru"><body><AuthProvider>{children}</AuthProvider></body></html>;
}
