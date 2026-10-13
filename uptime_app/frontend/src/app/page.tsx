import { AuthNav } from "@/components/auth-nav";
import { ApiStatus } from "@/components/api-status";

export default function Home() {
 return <main>
  <p className="eyebrow">UPTIME APP</p>
  <h1>Мониторинг доступности сервисов</h1>
  <p>Основа приложения готова. Здесь появятся ваши сервисы и история проверок.</p>
  <AuthNav />
  <ApiStatus />
 </main>;
}
