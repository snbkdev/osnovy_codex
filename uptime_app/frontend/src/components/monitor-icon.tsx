"use client";

import Image from "next/image";
import { useState } from "react";

export function MonitorIcon({ url }: { url: string }) {
  // Remount when the URL changes so a previous fallback does not persist.
  return <SiteIcon key={url} url={url} />;
}

function SiteIcon({ url }: { url: string }) {
  const [attempt, setAttempt] = useState(0);
  const origin = new URL(url).origin;
  const sources = [`${origin}/apple-touch-icon.png`, `${origin}/favicon.ico`, "/icon.svg"];

  return <Image
    key={attempt}
    className="monitor-icon"
    src={sources[attempt]}
    alt=""
    width={32}
    height={32}
    unoptimized
    referrerPolicy="no-referrer"
    onError={attempt < sources.length - 1 ? () => setAttempt(attempt + 1) : undefined}
  />;
}
