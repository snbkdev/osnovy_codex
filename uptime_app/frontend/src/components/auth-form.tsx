"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, login, publicAuthRequest } from "@/lib/auth";

type Mode = "register" | "login" | "verify-email" | "forgot-password" | "reset-password";
const titles: Record<Mode, string> = {
  register: "Регистрация", login: "Вход", "verify-email": "Подтверждение email",
  "forgot-password": "Восстановление пароля", "reset-password": "Новый пароль",
};
export function AuthForm({ mode }: { mode: Mode }) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [unverified, setUnverified] = useState(false);
  const needsToken = mode === "verify-email" || mode === "reset-password";
  const needsEmail = mode === "register" || mode === "login" || mode === "forgot-password";
  const needsPassword = mode === "register" || mode === "login" || mode === "reset-password";
  useEffect(() => {
    if (!needsToken) return;
    function readToken() {
      const fragment = window.location.hash;
      if (!fragment) {
        // Preserve the first read when Strict Mode repeats the initial effect.
        setToken((current) => current ?? "");
        return;
      }
      const value = new URLSearchParams(fragment.slice(1)).get("token") || "";
      window.history.replaceState(window.history.state, "", window.location.pathname + window.location.search);
      setToken(value);
      setMessage("");
      setError("");
      setPassword("");
    }
    // Tokens also arrive by opening another email link on the same page.
    readToken();
    window.addEventListener("hashchange", readToken);
    return () => window.removeEventListener("hashchange", readToken);
  }, [needsToken, mode]);
  async function submit(event: React.SubmitEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(""); setUnverified(false);
    try {
      if (needsPassword && (Array.from(password).length < 12 || Array.from(password).length > 128)) throw new Error("Пароль должен содержать от 12 до 128 символов.");
      if (mode === "login") { await login(email, password); router.push("/"); return; }
      await publicAuthRequest(mode, needsToken ? { token, ...(needsPassword ? { password } : {}) } : { email, ...(needsPassword ? { password } : {}) });
      setPassword("");
      setMessage(mode === "register" ? "Проверьте почту: если адрес подходит для регистрации, мы отправили ссылку подтверждения." : mode === "forgot-password" ? "Если адрес подходит для восстановления, вы получите письмо со ссылкой." : "Готово. Теперь войдите в аккаунт.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось выполнить запрос.");
      setUnverified(err instanceof ApiError && err.code === "email_not_verified");
    } finally { setBusy(false); }
  }
  async function resend() {
    setBusy(true); setError("");
    try { await publicAuthRequest("resend-verification", { email }); setMessage("Если адрес ожидает подтверждения, мы отправили новое письмо. Проверьте почту."); }
    catch (err) { setError(err instanceof Error ? err.message : "Не удалось отправить письмо."); }
    finally { setBusy(false); }
  }
  return <main>
    <Link href="/">← На главную</Link>
    <h1>{titles[mode]}</h1>
    <section>
      {message ? <><p role="status">{message}</p><Link href="/login">Перейти ко входу</Link></> : <>
        {needsToken && token === "" ? <p role="alert">В ссылке отсутствует токен. Откройте полную ссылку из письма или запросите новое письмо.</p> : <form onSubmit={submit}>
          {needsEmail && <label htmlFor="email">Email<input id="email" name="email" type="email" autoComplete="email" required maxLength={254} value={email} onChange={(e) => setEmail(e.target.value)} disabled={busy} /></label>}
          {needsPassword && <><label htmlFor="password">Пароль<input id="password" name="password" type="password" autoComplete={mode === "login" ? "current-password" : "new-password"} required aria-describedby="password-hint" value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} /></label><p id="password-hint" className="field-hint">От 12 до 128 символов. Пробелы сохраняются.</p></>}
          {mode === "verify-email" && <p>Нажмите кнопку, чтобы подтвердить свой email.</p>}
          <button disabled={busy || (needsToken && !token)} type="submit">{busy ? "Подождите…" : mode === "verify-email" ? "Подтвердить email" : mode === "login" ? "Войти" : mode === "reset-password" ? "Сохранить пароль" : mode === "register" ? "Зарегистрироваться" : "Отправить письмо"}</button>
        </form>}
      </>}
      {error && <p role="alert">{error}</p>}
      {unverified && <button disabled={busy} onClick={resend}>Отправить подтверждение ещё раз</button>}
      <nav>
        {mode === "login" && <><Link href="/forgot-password">Забыли пароль?</Link><Link href="/register">Создать аккаунт</Link></>}
        {mode === "register" && <Link href="/login">Уже есть аккаунт? Войти</Link>}
        {needsToken && <Link href={mode === "reset-password" ? "/forgot-password" : "/login"}>Запросить новое письмо</Link>}
      </nav>
    </section>
  </main>;
}
