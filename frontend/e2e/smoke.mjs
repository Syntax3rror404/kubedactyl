// Browser smoke test against a running panel (default http://localhost:8080).
//
//   E2E_USER=admin E2E_PASSWORD=… npm run e2e                  # loads every page, fails on JS errors / external requests
//   E2E_USER=… E2E_PASSWORD=… E2E_SERVER=<name> npm run e2e    # additionally: start, command, file ops, stop
//
// Uses an installed Chromium based browser (BROWSER=/path/to/binary); on macOS Chrome or Edge is found automatically.
import { existsSync } from "node:fs"
import puppeteer from "puppeteer-core"

const base = process.env.E2E_URL ?? "http://localhost:8080"
const username = process.env.E2E_USER
const password = process.env.E2E_PASSWORD
if (!username || !password) throw new Error("set E2E_USER and E2E_PASSWORD")
const server = process.env.E2E_SERVER
const browserPath =
  process.env.BROWSER ??
  [
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
    "/usr/bin/chromium",
    "/usr/bin/google-chrome",
  ].find(existsSync)
if (!browserPath) throw new Error("no browser found, set BROWSER")

const browser = await puppeteer.launch({ executablePath: browserPath, headless: true })
const page = await browser.newPage()
await page.setViewport({ width: 1440, height: 1000 })
const errors = []
const external = []
page.on("pageerror", (e) => errors.push(e.message))
let signedIn = false
page.on("console", (m) => {
  // Before the login the session check (/api/auth/me) answers 401 on purpose.
  if (m.type() === "error" && !(signedIn === false && m.text().includes("status of 401"))) errors.push(m.text())
})
page.on(
  "request",
  (r) => !/^(https?:\/\/localhost|wss?:\/\/localhost|data:|blob:)/.test(r.url()) && external.push(r.url()),
)

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
let token = ""
const api = async (path) =>
  (await fetch(base + "/api" + path, { headers: { Authorization: `Bearer ${token}` } })).json()
const step = (label) => console.log("ok ", label)
const clickText = async (selector, text) => {
  for (const el of await page.$$(selector))
    if ((await el.evaluate((n) => n.innerText.trim())) === text) return el.click()
  throw new Error(`${selector} "${text}" not found`)
}
const waitPhase = async (want, seconds = 180) => {
  for (let i = 0; i < seconds; i++) {
    if ((await api(`/servers/${server}`)).status?.phase === want) return
    await sleep(1000)
  }
  throw new Error(`server did not reach ${want}`)
}

try {
  // Sign in through the login page, like a user would.
  await page.goto(base + "/login", { waitUntil: "networkidle2" })
  await page.type("#username", username)
  await page.type("#password", password)
  const [loginResponse] = await Promise.all([
    page.waitForResponse((r) => r.url().endsWith("/api/auth/login")),
    page.click("button[type=submit]"),
  ])
  if (!loginResponse.ok()) throw new Error(`login failed: ${loginResponse.status()}`)
  token = (await loginResponse.json()).token
  signedIn = true
  await page.waitForSelector("h1")
  step(`signed in as ${username}`)

  const eggs = (await api("/eggs")).items
  const servers = (await api("/servers")).items
  const me = await api("/auth/me")
  const pages = ["/", "/account"]
  if (me.role === "admin")
    pages.push(
      "/eggs",
      "/eggs/new",
      "/users",
      "/cluster",
      "/settings",
      "/servers/new",
      ...eggs.slice(0, 1).flatMap((e) => [`/eggs/${e.metadata.name}`, `/eggs/${e.metadata.name}/edit`]),
    )
  for (const s of servers.slice(0, 1))
    for (const tab of ["", "/files", "/startup", "/schedules", "/backups", "/settings"])
      pages.push(`/servers/${s.metadata.name}${tab}`)
  for (const theme of ["light", "dark"]) {
    await page.evaluateOnNewDocument((t) => localStorage.setItem("app-theme", t), theme)
    for (const path of pages) {
      await page.goto(base + path, { waitUntil: "networkidle2" })
      await sleep(500)
    }
    step(`${pages.length} pages loaded (${theme})`)
  }

  if (server) {
    await page.goto(`${base}/servers/${server}`, { waitUntil: "networkidle2" })
    if ((await api(`/servers/${server}`)).status?.phase !== "Running") {
      await clickText("button", "Start")
      await waitPhase("Running")
    }
    step("server running")
    await page.type('input[placeholder="Type a command…"]', "list")
    await page.keyboard.press("Enter")
    await sleep(2500)
    step(`command echoed: ${/list/.test(await page.$eval(".xterm-rows", (e) => e.innerText))}`)

    await page.goto(`${base}/servers/${server}/files`, { waitUntil: "networkidle2" })
    // The file container starts on demand; the page shows a waiting view until then.
    await page.waitForSelector("table", { timeout: 120_000 })
    step("file container started on demand")
    await clickText("button", "Folder")
    await page.waitForSelector("[role=dialog] input")
    await page.type("[role=dialog] input", "e2e-test")
    await clickText("[role=dialog] button", "Save")
    await sleep(2000)
    const rows = async () => page.$$eval("tbody tr", (r) => r.map((x) => x.innerText))
    if (!(await rows()).some((t) => t.includes("e2e-test"))) throw new Error("folder not created")
    for (const row of await page.$$("tbody tr")) {
      if ((await row.evaluate((n) => n.innerText)).includes("e2e-test")) {
        await (await row.$("button[aria-haspopup=menu]")).click()
        break
      }
    }
    await clickText("[role=menuitem]", "Delete")
    await page.waitForSelector("[role=alertdialog]")
    await clickText("[role=alertdialog] button", "Delete")
    await sleep(2000)
    if ((await rows()).some((t) => t.includes("e2e-test"))) throw new Error("folder not deleted")
    step("file manager create + delete")

    await page.goto(`${base}/servers/${server}`, { waitUntil: "networkidle2" })
    await clickText("button", "Stop")
    await waitPhase("Offline")
    step("server stopped")
  }

  if (errors.length || external.length)
    throw new Error(`errors: ${JSON.stringify(errors)} external: ${JSON.stringify(external)}`)
  console.log("PASS")
} finally {
  await browser.close()
}
