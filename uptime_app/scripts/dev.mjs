import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { chmod, lstat, unlink } from "node:fs/promises";
import { createConnection, createServer } from "node:net";
import { socketPath } from "./dev-socket.mjs";

const root = new URL("../", import.meta.url);
const children = new Set();
let stopping = false;
let killTimer;
const clients = new Set();
const server = createServer((socket) => {
  socket.setTimeout(1000, () => socket.destroy());
  socket.once("data", (message) => {
    if (message.toString() !== "stop\n") { socket.destroy(); return; }
    socket.setTimeout(0);
    clients.add(socket);
    socket.on("close", () => clients.delete(socket));
    stop(0);
  });
});

function finished() {
  if (!stopping || children.size) return;
  clearTimeout(killTimer);
  for (const socket of clients) socket.end("Backend and frontend stopped.\n");
}

function stop(code) {
  if (stopping) return;
  stopping = true;
  process.exitCode = code;
  server.close();
  for (const child of children) {
    try { process.kill(-child.pid, "SIGTERM"); }
    catch (error) { if (error.code !== "ESRCH") console.error(error.message); }
  }
  killTimer = setTimeout(() => {
    for (const child of children) {
      try { process.kill(-child.pid, "SIGKILL"); }
      catch (error) { if (error.code !== "ESRCH") console.error(error.message); }
    }
  }, 5000);
  finished();
}

function start(name, command, args, directory) {
  console.log(`Starting ${name}…`);
  const child = spawn(command, args, {
    cwd: fileURLToPath(new URL(directory, root)),
    stdio: "inherit",
    detached: true,
  });
  children.add(child);
  child.on("error", (error) => {
    console.error(`${name}: ${error.message}`);
    children.delete(child);
    stop(1);
  });
  child.on("close", () => { children.delete(child); finished(); });
  child.on("exit", (code, signal) => {
    if (!stopping) {
      console.error(`${name} stopped (${signal || code}).`);
      stop(code || 1);
    }
  });
}

// Probe before removing a socket left by an interrupted previous run.
const active = await new Promise((resolve, reject) => {
  const probe = createConnection(socketPath);
  probe.on("connect", () => { probe.destroy(); resolve(true); });
  probe.on("error", (error) => {
    if (["ENOENT", "ECONNREFUSED"].includes(error.code)) resolve(false);
    else reject(error);
  });
});
if (active) {
  console.error("Development servers are already running. Use make stop first.");
  process.exit(1);
}
try {
  const info = await lstat(socketPath);
  if (!info.isSocket() || info.uid !== process.getuid()) throw new Error("Unexpected development socket owner or file type.");
  await unlink(socketPath);
} catch (error) { if (error.code !== "ENOENT") throw error; }
await new Promise((resolve, reject) => {
  server.once("error", reject);
  server.listen(socketPath, resolve);
});
await chmod(socketPath, 0o600);
process.on("SIGINT", () => stop(130));
process.on("SIGTERM", () => stop(143));
start("backend", "bash", ["../scripts/backend-env.sh", "go", "run", "./cmd/api"], "backend/");
start("frontend", "npm", ["run", "dev"], "frontend/");
