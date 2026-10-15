"use client";

/* eslint-disable @next/next/no-img-element -- Local file previews and same-origin avatars. */
import { useEffect, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { saveAvatar, type User } from "@/lib/auth";

export function AvatarEditor({ user, onSaved, disabled, onBusyChange, onSelected }: { user: User; onSaved: (user: User) => void; disabled: boolean; onBusyChange: (busy: boolean) => void; onSelected: (file: Blob | null) => void }) {
  const [source, setSource] = useState("");
  const [crop, setCrop] = useState({ x: 50, y: 50, zoom: 1 });
  const [selected, setSelected] = useState<Blob | null>(null);
  const [preview, setPreview] = useState("");
  const [busy, setBusy] = useState(false);
  const [dimensions, setDimensions] = useState({ width: 256, height: 256 });
  const [ready, setReady] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const image = useRef<HTMLImageElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const selectionVersion = useRef(0);
  useEffect(() => () => { if (source) URL.revokeObjectURL(source); }, [source]);
  useEffect(() => () => { if (preview) URL.revokeObjectURL(preview); }, [preview]);

  function chooseFile(file?: File) {
    selectionVersion.current++;
    setError(""); setMessage("");
    if (!file) return;
    if (!["image/jpeg", "image/png"].includes(file.type) || file.size > 5 * 1024 * 1024) {
      setError("Выберите JPEG или PNG размером до 5 МБ."); return;
    }
    setReady(false); setCrop({ x: 50, y: 50, zoom: 1 });
    setSource(URL.createObjectURL(file));
  }
  async function selectCrop() {
    const img = image.current;
    if (!img || !ready) return;
    const version = selectionVersion.current;
    const side = Math.min(img.naturalWidth, img.naturalHeight) / crop.zoom;
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = 256;
    const context = canvas.getContext("2d");
    if (!context) { setError("Не удалось обработать изображение."); return; }
    context.drawImage(img, (img.naturalWidth - side) * crop.x / 100, (img.naturalHeight - side) * crop.y / 100, side, side, 0, 0, 256, 256);
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/png"));
    if (version !== selectionVersion.current) return;
    if (!blob) { setError("Не удалось обработать изображение."); return; }
    setSelected(blob); onSelected(blob); setPreview(URL.createObjectURL(blob)); setSource("");
  }
  async function save() {
    if (!selected) return;
    setBusy(true); onBusyChange(true); setError(""); setMessage("");
    try {
      const data = await saveAvatar(selected);
      setSelected(null); setPreview(""); onSaved(data);
    } catch (err) { setError(err instanceof Error ? err.message : "Не удалось сохранить аватар."); }
    finally { setBusy(false); onBusyChange(false); }
  }
  const scale = 100 * crop.zoom / Math.min(dimensions.width, dimensions.height);
  return <fieldset className="avatar-editor" disabled={busy || disabled}>
    <legend>Аватар</legend>
    {preview ? <img className="avatar-preview" src={preview} alt="Предпросмотр нового аватара" width={96} height={96} /> : <Avatar user={user} />}
    <input ref={fileInput} id="avatar-file" className="avatar-file" type="file" accept="image/jpeg,image/png" aria-label="Загрузить аватар" onChange={(event) => { chooseFile(event.target.files?.[0]); event.target.value = ""; }} />
    <button type="button" onClick={() => fileInput.current?.click()}>Загрузить аватар</button>
    <p className="field-hint">JPEG или PNG, до 5 МБ и 4096×4096 пикселей. Настройте область и нажмите «Выбрать», затем «Сохранить аватар» или «Сохранить изменения».</p>
    {source && <div className="avatar-crop-controls">
      <div className="avatar-crop" aria-label="Область аватара">
        <img ref={image} src={source} alt="Выбранное изображение в рамке обрезки" onLoad={(event) => {
          const el = event.currentTarget;
          if (el.naturalWidth > 4096 || el.naturalHeight > 4096 || el.naturalWidth * el.naturalHeight > 16000000) { setSource(""); setError("Изображение слишком большое: максимум 4096×4096 и 16 млн пикселей."); }
          else { setDimensions({ width: el.naturalWidth, height: el.naturalHeight }); setReady(true); }
        }} onError={() => { setSource(""); setError("Не удалось прочитать изображение. Выберите корректный JPEG или PNG."); }} style={{ width: `${dimensions.width * scale}%`, height: `${dimensions.height * scale}%`, left: `${-(dimensions.width * scale - 100) * crop.x / 100}%`, top: `${-(dimensions.height * scale - 100) * crop.y / 100}%` }} />
      </div>
      <label>Масштаб<input type="range" min="1" max="3" step="0.01" value={crop.zoom} onChange={(event) => setCrop({ ...crop, zoom: Number(event.target.value) })} /></label>
      <label>Положение по горизонтали<input type="range" min="0" max="100" value={crop.x} onChange={(event) => setCrop({ ...crop, x: Number(event.target.value) })} /></label>
      <label>Положение по вертикали<input type="range" min="0" max="100" value={crop.y} onChange={(event) => setCrop({ ...crop, y: Number(event.target.value) })} /></label>
      <button type="button" disabled={!ready} onClick={() => { void selectCrop(); }}>Выбрать</button>
      <button type="button" onClick={() => { selectionVersion.current++; setSource(""); }}>Отмена</button>
    </div>}
    {selected && !source && <><button type="button" onClick={() => { void save(); }}>{busy ? "Сохраняем аватар…" : "Сохранить аватар"}</button><button type="button" onClick={() => { setSelected(null); onSelected(null); setPreview(""); }}>Отменить новый аватар</button></>}
    {selected && <p role="status">Новый аватар ещё не сохранён.</p>}
    {error && <p role="alert">{error}</p>}
    {message && <p role="status">{message}</p>}
  </fieldset>;
}
