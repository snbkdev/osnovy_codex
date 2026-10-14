import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";

// Keep the Unix socket path short enough for macOS and unique to this checkout.
const checkout = fileURLToPath(new URL("../", import.meta.url));
const hash = createHash("sha256").update(checkout).digest("hex").slice(0, 16);
export const socketPath = `/tmp/uptime-dev-${process.getuid()}-${hash}.sock`;
