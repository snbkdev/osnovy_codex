import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const frontend = fileURLToPath(new URL("../", import.meta.url));
const appOutput = path.join(frontend, ".next/server/app");

// Run against a fresh production build to verify Next.js metadata generation.
test("favicon is served as SVG and preserves the source asset", async () => {
  const metadata = JSON.parse(await readFile(path.join(appOutput, "icon.svg.meta"), "utf8"));
  assert.equal(metadata.status, 200);
  assert.equal(metadata.headers["content-type"], "image/svg+xml");
  const source = await readFile(path.join(frontend, "src/app/icon.svg"), "utf8");
  const served = await readFile(path.join(appOutput, "icon.svg.body"), "utf8");
  assert.equal(served, source);
  assert.match(source, /<svg\b[^>]*xmlns="http:\/\/www\.w3\.org\/2000\/svg"/);
  assert.match(source, /\bviewBox="0 0 32 32"/);
});

test("every generated page includes the shared favicon exactly once", async (t) => {
  const files = (await readdir(appOutput, { recursive: true })).filter((file) => file.endsWith(".html"));
  assert.ok(files.includes("index.html"), "The production build must include the home page");
  assert.ok(files.includes("login.html"), "The production build must include the login page");
  for (const file of files) {
    await t.test(file, async () => {
      const html = await readFile(path.join(appOutput, file), "utf8");
      const head = html.match(/<head\b[^>]*>([\s\S]*?)<\/head>/)?.[1];
      assert.ok(head, "Page must have a head element");
      const icons = [...head.matchAll(/<link\b[^>]*\brel="icon"[^>]*>/g)];
      assert.equal(icons.length, 1, "Expected a single favicon link");
      const attributes = Object.fromEntries([...icons[0][0].matchAll(/([\w-]+)="([^"]*)"/g)].map((match) => [match[1], match[2]]));
      const iconURL = new URL(attributes.href, "https://uptime.example");
      assert.equal(iconURL.origin, "https://uptime.example");
      assert.equal(iconURL.pathname, "/icon.svg");
      assert.equal(attributes.type, "image/svg+xml");
      assert.equal(attributes.sizes, "any");
    });
  }
});
