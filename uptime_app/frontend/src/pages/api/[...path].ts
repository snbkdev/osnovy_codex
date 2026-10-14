import type { NextApiRequest, NextApiResponse } from "next";
import http from "node:http";
import https from "node:https";

export const config = { api: { bodyParser: false, responseLimit: false } };

// Read the actual peer socket. Never accept a browser-supplied forwarded IP.
export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  await new Promise<void>((resolve) => {
    res.once("finish", resolve);
    res.once("close", resolve);
    const target = new URL(process.env.API_URL || "http://127.0.0.1:8080");
    const path = req.url || "/api/";
    const headers = { ...req.headers };
    for (const name of ["connection", "host", "forwarded", "x-forwarded-for", "x-real-ip", "x-uptime-client-ip", "x-uptime-proxy-secret"]) delete headers[name];
    headers["x-uptime-client-ip"] = req.socket.remoteAddress?.replace(/^::ffff:/, "") || "";
    headers["x-uptime-proxy-secret"] = process.env.PROXY_SECRET || "";
    const upstream = (target.protocol === "https:" ? https : http).request({
      hostname: target.hostname, port: target.port, path, method: req.method, headers, timeout: 15000,
    }, (response) => {
      res.statusCode = response.statusCode || 502;
      for (const [key, value] of Object.entries(response.headers)) {
        if (value !== undefined && !["connection", "transfer-encoding"].includes(key)) res.setHeader(key, value);
      }
      response.on("error", () => res.destroy());
      response.pipe(res);
    });
    upstream.on("timeout", () => upstream.destroy(new Error("API timeout")));
    upstream.on("error", () => {
      if (res.headersSent) { res.destroy(); return; }
      res.setHeader("Cache-Control", "no-store");
      res.status(503).json({ error: { code: "service_unavailable", message: "API временно недоступен." } });
    });
    req.on("aborted", () => upstream.destroy());
    req.pipe(upstream);
  });
}
