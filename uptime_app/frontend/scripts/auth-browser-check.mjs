import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { chromium } from "playwright";
import ts from "typescript";

const base = process.env.AUTH_TEST_URL || "http://localhost:3000";
const mailpit = process.env.MAILPIT_URL || "http://localhost:8025";
for (const address of [base, mailpit]) {
  assert(["localhost", "127.0.0.1", "[::1]"].includes(new URL(address).hostname), "Run only against local test services");
}
const browser = await chromium.launch({
  headless: true,
  ...(process.env.BROWSER_EXECUTABLE ? { executablePath: process.env.BROWSER_EXECUTABLE } : {}),
});
const context = await browser.newContext();
const page = await context.newPage();
page.setDefaultTimeout(10000);
const email = `browser-${Date.now()}@example.com`;
const password = "  Browser password  ";
const nextPassword = "  New browser password  ";
const errors = [];
context.on("page", (tab) => tab.on("pageerror", (error) => errors.push(error.message)));
page.on("pageerror", (error) => errors.push(error.message));

async function poll(work) {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    const value = await work();
    if (value) return value;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error("Timed out waiting for expected state");
}
async function visible(tab, text) { await tab.getByText(text, { exact: false }).first().waitFor(); }
async function fill(tab, route, pass = password) {
  await tab.goto(`${base}/${route}`);
  await tab.getByLabel("Email", { exact: true }).fill(email);
  if (route !== "forgot-password") await tab.getByLabel("Пароль", { exact: true }).fill(pass);
}
async function letterLink(path, previous) {
  return poll(async () => {
    const list = await fetch(`${mailpit}/api/v1/messages`).then((res) => res.json());
    for (const message of list.messages || []) {
      if (!message.To?.some((to) => to.Address === email)) continue;
      const detail = await fetch(`${mailpit}/api/v1/message/${message.ID}`).then((res) => res.json());
      const link = detail.Text?.match(new RegExp(`${base}/${path}#token=[A-Za-z0-9_-]+`))?.[0];
      if (link && link !== previous) return link;
    }
    return null;
  });
}
async function api(tab, path, body) {
  return tab.evaluate(async ({ path, body }) => {
    const res = await fetch(`/api/v1/auth/${path}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
    return { status: res.status, data: await res.json() };
  }, { path, body });
}
async function tokenPage(link, button, pass) {
  let posts = 0;
  const listener = (req) => { if (req.method() === "POST" && /\/(verify-email|reset-password)$/.test(req.url())) posts++; };
  page.on("request", listener);
  await page.goto(link);
  await poll(async () => !(await page.evaluate(() => location.hash)));
  assert.equal(posts, 0, "Email links must require explicit user action");
  if (pass) await page.getByLabel("Пароль", { exact: true }).fill(pass);
  await page.getByRole("button", { name: button, exact: true }).click();
  await poll(() => posts === 1);
  page.off("request", listener);
}

try {
  await fill(page, "register");
  await page.getByRole("button", { name: "Зарегистрироваться", exact: true }).click();
  await visible(page, "Проверьте почту");
  const firstLink = await letterLink("verify-email");
  await fill(page, "login");
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await visible(page, "Подтвердите email перед входом");
  await page.getByRole("button", { name: "Отправить подтверждение ещё раз" }).click();
  await visible(page, "мы отправили новое письмо");
  const verification = await letterLink("verify-email", firstLink);
  await tokenPage(firstLink, "Подтвердить email");
  await visible(page, "Ссылка могла истечь или уже использована");
  await tokenPage(verification, "Подтвердить email");
  await visible(page, "Теперь войдите в аккаунт");
  await tokenPage(verification, "Подтвердить email");
  await visible(page, "Ссылка могла истечь или уже использована");
  for (const path of ["verify-email", "reset-password"]) {
    await page.goto(`${base}/${path}`);
    await visible(page, "В ссылке отсутствует токен");
    await tokenPage(`${base}/${path}#token=invalid`, path === "verify-email" ? "Подтвердить email" : "Сохранить пароль", path === "reset-password" ? nextPassword : undefined);
    await visible(page, "Ссылка могла истечь или уже использована");
    await page.route(`**/api/v1/auth/${path}`, (route) => route.fulfill({
      status: 400, json: { error: { code: "invalid_input", message: "Проверьте данные. Ссылка могла истечь или уже использована." } },
    }));
    await tokenPage(`${base}/${path}#token=expired`, path === "verify-email" ? "Подтвердить email" : "Сохранить пароль", path === "reset-password" ? nextPassword : undefined);
    await visible(page, "Ссылка могла истечь или уже использована");
    await page.unroute(`**/api/v1/auth/${path}`);
  }
  console.log("PASS registration, SMTP delivery, resend, explicit confirmation, missing/invalid/reused links");

  await fill(page, "login");
  const loginResponse = page.waitForResponse((res) => res.url().endsWith("/auth/login") && res.status() === 200);
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  const loginData = await (await loginResponse).json();
  await visible(page, email);
  const cookie = (await context.cookies(`${base}/api/v1/auth`)).find((item) => item.name === "uptime_refresh");
  assert(cookie?.httpOnly && cookie.sameSite === "Strict" && cookie.path === "/api/v1/auth");
  assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0);
  await page.reload();
  await visible(page, email);
  const tab = await context.newPage();
  await tab.goto(base);
  await visible(tab, email);
  await Promise.all([page.reload(), tab.reload()]);
  await Promise.all([visible(page, email), visible(tab, email)]);
  assert.equal(await page.locator("main").innerText(), "");
  assert.equal(await page.getByRole("button", { name: "Проверить API" }).count(), 0);
  assert(await page.getByRole("button", { name: "Выйти", exact: true }).isVisible());
  console.log("PASS login, cookie flags, memory-only tokens, reload, simultaneous tabs, empty dashboard");

  // Exercise the actual client module with the real browser's cookies and Web Locks.
  const source = await readFile(new URL("../src/lib/auth.ts", import.meta.url), "utf8");
  const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  for (const target of [page, tab]) {
    await target.evaluate((code) => {
      const exports = {};
      new Function("exports", code)(exports);
      window.authCheck = exports;
    }, compiled);
  }
  let refreshCount = 0;
  const countRefresh = (req) => { if (req.url().endsWith("/auth/refresh")) refreshCount++; };
  page.on("request", countRefresh);
  assert((await page.evaluate(() => Promise.all(Array.from({ length: 8 }, () => window.authCheck.refreshSession())))).every(Boolean));
  assert.equal(refreshCount, 1, "Concurrent refresh calls should share one request");
  assert((await Promise.all([page.evaluate(() => window.authCheck.refreshSession()), tab.evaluate(() => window.authCheck.refreshSession())])).every(Boolean));
  let meCalls = 0;
  await page.route("**/api/v1/auth/me", async (route) => {
    assert(route.request().headers().authorization?.startsWith("Bearer "));
    meCalls++;
    if (meCalls === 1) await route.fulfill({ status: 401, json: { error: { code: "unauthorized", message: "Expired" } } });
    else await route.continue();
  });
  assert.equal((await page.evaluate(() => window.authCheck.authRequest("me"))).email, email);
  assert.equal(meCalls, 2);
  await page.unroute("**/api/v1/auth/me");
  meCalls = 0;
  await page.route("**/api/v1/auth/me", async (route) => {
    meCalls++;
    await route.fulfill({ status: 401, json: { error: { code: "unauthorized", message: "Expired" } } });
  });
  assert.equal(await page.evaluate(() => window.authCheck.authRequest("me").then(() => 0, (error) => error.status)), 401);
  assert.equal(meCalls, 2, "A protected request may only retry once");
  await page.unroute("**/api/v1/auth/me");
  page.off("request", countRefresh);
  console.log("PASS refresh deduplication, cross-tab locks, Bearer header, single retry");

  await fill(page, "forgot-password");
  await page.getByRole("button", { name: "Отправить письмо" }).click();
  await visible(page, "вы получите письмо");
  const reset = await letterLink("reset-password");
  await tokenPage(reset, "Сохранить пароль", nextPassword);
  await visible(page, "Теперь войдите в аккаунт");
  assert.equal((await api(tab, "refresh", {})).status, 401);
  const oldAccess = await tab.evaluate(async (token) => (await fetch("/api/v1/auth/me", { headers: { Authorization: `Bearer ${token}` } })).status, loginData.access_token);
  assert.equal(oldAccess, 401);
  await tokenPage(reset, "Сохранить пароль", nextPassword);
  await visible(page, "Ссылка могла истечь или уже использована");
  await fill(page, "login");
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await visible(page, "Неверный email или пароль");
  await page.getByLabel("Пароль", { exact: true }).fill(nextPassword);
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await visible(page, email);
  await tab.goto(base);
  await visible(tab, email);
  await page.locator(".dashboard-profile summary").click();
  await page.getByRole("heading", { name: "Профиль пользователя", exact: true }).waitFor();
  await visible(page, "Email подтверждён");
  await page.keyboard.press("Escape");
  assert.equal(await page.locator(".dashboard-profile").getAttribute("open"), null);
  await page.locator(".dashboard-profile summary").click();
  await page.getByRole("button", { name: "Выйти", exact: true }).click();
  await page.waitForURL(`${base}/login`);
  await page.getByRole("heading", { name: "Вход", exact: true }).waitFor();
  await tab.getByRole("link", { name: "Войти", exact: true }).waitFor();
  await page.goto(base);
  await page.reload();
  await page.getByRole("link", { name: "Войти", exact: true }).waitFor();
  assert.equal((await context.cookies(`${base}/api/v1/auth`)).filter((item) => item.name === "uptime_refresh").length, 0);
  console.log("PASS password reset, old password/access/refresh rejection, new login, cross-tab logout");

  await page.goto(`${base}/register`);
  await page.getByLabel("Email", { exact: true }).focus();
  await page.keyboard.press("Tab");
  assert.equal(await page.evaluate(() => document.activeElement.id), "password");
  await page.route("**/api/v1/auth/register", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 500));
    await route.fulfill({ status: 503, json: { error: { code: "service_unavailable", message: "Сервис временно недоступен." } } });
  });
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Пароль", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Зарегистрироваться", exact: true }).click();
  assert(await page.getByRole("button", { name: "Подождите…" }).isDisabled());
  await visible(page, "Сервис временно недоступен");
  await page.unroute("**/api/v1/auth/register");
  await page.route("**/api/v1/auth/refresh", (route) => route.fulfill({
    status: 503, json: { error: { code: "service_unavailable", message: "Сервис временно недоступен." } },
  }));
  await page.goto(base);
  await visible(page, "Сервис временно недоступен");
  await page.unroute("**/api/v1/auth/refresh");
  await page.getByRole("button", { name: "Повторить восстановление сессии" }).click();
  await poll(async () => (await page.getByRole("button", { name: "Повторить восстановление сессии" }).count()) === 0);
  assert.deepEqual(errors, []);
  console.log("PASS accessible labels, keyboard focus, loading/error states; no browser exceptions");
} finally {
  await browser.close();
}
