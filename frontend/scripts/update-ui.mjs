// Re-installs every library component listed in ui-components.json from its source (shadcn registry,
// Magic UI registry, shadcn docs), exactly as delivered, with the dependency versions the registry items name.
// Run with `make ui-update`, then check `npx tsc -b` and the app; the CLI lists the files it updated.
import { execFileSync } from "node:child_process"
import { readFileSync, writeFileSync } from "node:fs"
import { join } from "node:path"

const root = new URL("..", import.meta.url).pathname
const manifest = JSON.parse(readFileSync(join(root, "ui-components.json"), "utf8"))

const run = (args) =>
  execFileSync("npx", ["-y", "shadcn@latest", ...args], { cwd: root, stdio: ["ignore", "inherit", "inherit"] })

console.log("== shadcn registry")
run(["add", ...manifest.shadcn, "--overwrite", "-y"])

console.log("== magic ui registry")
run(["add", ...manifest.magicui.map((n) => `@magicui/${n}`), "--overwrite", "-y"])

console.log("== shadcn docs")
for (const [url, files] of Object.entries(manifest.docs)) {
  const doc = await (await fetch(url)).text()
  for (const [, title, code] of doc.matchAll(/```tsx title="([^"]+)"[^\n]*\n([\s\S]*?)```/g)) {
    if (files.includes(title)) {
      writeFileSync(join(root, "src", title), code)
      console.log(`wrote src/${title}`)
    }
  }
}
