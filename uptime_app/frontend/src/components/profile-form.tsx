"use client";

import { useEffect, useState } from "react";
import { AvatarEditor } from "@/components/avatar-editor";
import { authRequest, saveAvatar, saveProfile, type User } from "@/lib/auth";

export function ProfileForm() {
  const [name, setName] = useState("");
  const [birthday, setBirthday] = useState("");
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pendingAvatar, setPendingAvatar] = useState<Blob | null>(null);
  const [avatarRevision, setAvatarRevision] = useState(0);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    authRequest<User>("me").then((data) => {
      if (!active) return;
      setUser(data); setName(data.full_name); setBirthday(data.date_of_birth || "");
    }).catch((err: unknown) => {
      if (active) setError(err instanceof Error ? err.message : "Не удалось загрузить профиль.");
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [attempt]);
  function avatarSaved(data: User) {
    setUser(data); setPendingAvatar(null); setAvatarRevision((value) => value + 1);
    setError(""); setMessage("Аватар сохранён.");
  }
  async function submit(event: React.SubmitEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(""); setMessage("");
    try {
      if (!/\p{L}/u.test(name)) throw new Error("ФИО должно содержать хотя бы одну букву, а не только пробелы или символы.");
      if (Array.from(name.trim()).length > 200) throw new Error("ФИО должно содержать не более 200 символов.");
      let data = await saveProfile(name, birthday);
      setUser(data); setName(data.full_name); setBirthday(data.date_of_birth || "");
      if (pendingAvatar) {
        data = await saveAvatar(pendingAvatar);
        avatarSaved(data);
      }
      setMessage("Профиль сохранён.");
    } catch (err) { setError(err instanceof Error ? err.message : "Не удалось сохранить профиль."); }
    finally { setBusy(false); }
  }
  if (loading) return <p role="status">Загружаем профиль…</p>;
  if (!user) return <section><p role="alert">{error}</p><button onClick={() => { setError(""); setLoading(true); setAttempt((value) => value + 1); }}>Повторить загрузку</button></section>;
  return <section className="profile-card" aria-label="Редактирование профиля">
    <AvatarEditor key={avatarRevision} user={user} onSaved={avatarSaved} disabled={busy} onBusyChange={setBusy} onSelected={setPendingAvatar} />
    <form onSubmit={submit}>
      <label htmlFor="profile-email">Email<input id="profile-email" type="email" value={user.email} readOnly aria-describedby="profile-email-hint" /></label>
      <p id="profile-email-hint" className="field-hint">Email используется для входа и не меняется в этой форме.</p>
      <label htmlFor="full-name">ФИО<input id="full-name" name="full_name" autoComplete="name" required value={name} onChange={(event) => { setName(event.target.value); setMessage(""); }} disabled={busy} /></label>
      <label htmlFor="date-of-birth">Дата рождения<input id="date-of-birth" name="date_of_birth" type="date" autoComplete="bday" min="0001-01-01" max={new Date().toISOString().slice(0, 10)} value={birthday} onChange={(event) => { setBirthday(event.target.value); setMessage(""); }} disabled={busy} /></label>
      <button type="submit" disabled={busy}>{busy ? "Сохраняем…" : "Сохранить изменения"}</button>
    </form>
    {message && <p role="status">{message}</p>}
    {error && <p role="alert">{error}</p>}
  </section>;
}
