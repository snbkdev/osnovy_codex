import { createConnection } from "node:net";
import { socketPath } from "./dev-socket.mjs";

const socket = createConnection(socketPath);
socket.setTimeout(10000, () => socket.destroy(new Error("Timed out stopping development servers.")));
socket.on("connect", () => socket.write("stop\n"));
socket.on("data", (message) => process.stdout.write(message));
socket.on("error", (error) => {
  if (error.code === "ENOENT" || error.code === "ECONNREFUSED") {
    console.log("No active make dev for this checkout.");
  } else {
    console.error(error.message);
    process.exitCode = 1;
  }
});
