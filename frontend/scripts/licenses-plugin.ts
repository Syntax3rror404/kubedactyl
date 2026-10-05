import fs from "node:fs"
import path from "node:path"
import type { Plugin, Rolldown } from "vite"

// Writes the license texts of the npm packages listed in THIRD_PARTY_NOTICES.md ("npm packages"
// table: name, version, license, tracked by hand) to "licenses-npm.md" in the build output.
// backend/tools/licenses merges it with the Go modules into third-party-licenses.md (shown on the
// licenses page and shipped in the image). The build fails when a package in the bundle is not
// listed, or when a listed package is not installed in that version or under that license.

const licenseFile = /^(licen[cs]e|copying|notice|copyright)([.-].*)?$/i

interface Pkg {
  name: string
  version: string
  license: string
  author: string
  repository: string
  dir: string
}

const tableHeading = "### npm packages"

const row = (p: Pick<Pkg, "name" | "version" | "license">) => `| ${p.name} | ${p.version} | ${p.license} |`

/** Rows of the npm table in THIRD_PARTY_NOTICES.md. */
function listedPackages(root: string): Pick<Pkg, "name" | "version" | "license">[] {
  const rows = []
  let inTable = false
  for (const line of fs.readFileSync(path.join(root, "../THIRD_PARTY_NOTICES.md"), "utf8").split("\n")) {
    if (line.startsWith("#")) {
      inTable = line.startsWith(tableHeading)
      continue
    }
    const cells = line
      .trim()
      .replace(/^\||\|$/g, "")
      .split("|")
      .map((c) => c.trim())
    if (inTable && cells.length === 3 && cells[0] !== "Package" && !cells[0].startsWith("---"))
      rows.push({ name: cells[0], version: cells[1], license: cells[2] })
  }
  return rows
}

// Standard texts for packages that declare MIT/ISC but ship no license file.
const mit = (holder: string) => `MIT License

Copyright (c) ${holder}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`

const isc = (holder: string) => `ISC License

Copyright (c) ${holder}

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.`

/** License text for a package without license file, from the license declared in package.json. */
function declaredText(p: Pkg): string {
  const note = `(The package ships no license file; standard text of the license declared in its package.json, author "${p.author || "unknown"}"${p.repository ? `, ${p.repository}` : ""}.)`
  const holder = p.author ? `${p.author} and contributors` : `the ${p.name} authors`
  const texts = [/\bMIT\b/.test(p.license) && mit(holder), /\bISC\b/.test(p.license) && isc(holder)].filter(Boolean)
  if (texts.length === 0)
    throw new Error(`${p.name}@${p.version}: no license file and no MIT/ISC license declared ("${p.license}")`)
  return [note, ...texts].join("\n\n")
}

/** Package directory of a file inside node_modules (the innermost one for nested packages). */
function packageDir(file: string): string | null {
  const parts = file.split(/[\\/]/)
  const i = parts.lastIndexOf("node_modules")
  if (i < 0 || i + 1 >= parts.length) return null
  const n = parts[i + 1].startsWith("@") ? 3 : 2
  return parts.slice(0, i + n).join("/")
}

/** Bare package imports in the main stylesheet (Tailwind resolves those, not Vite's module graph). */
function cssPackages(root: string): string[] {
  const css = fs.readFileSync(path.join(root, "src/index.css"), "utf8")
  return [...css.matchAll(/@import\s+"([^"./][^"]*)"/g)].map((m) => {
    const spec = m[1].split("/")
    return path.join(root, "node_modules", ...(spec[0].startsWith("@") ? spec.slice(0, 2) : spec.slice(0, 1)))
  })
}

function readPkg(dir: string): Pkg | null {
  try {
    const j = JSON.parse(fs.readFileSync(path.join(dir, "package.json"), "utf8"))
    if (!j.name || !j.version) return null
    const license =
      typeof j.license === "string"
        ? j.license
        : (j.licenses ?? []).map((l: { type?: string }) => l.type).join(" OR ") || "unknown"
    const author = typeof j.author === "string" ? j.author : (j.author?.name ?? "")
    const repository = typeof j.repository === "string" ? j.repository : (j.repository?.url ?? "")
    return { name: j.name, version: j.version, license, author: author.replace(/\s*<[^>]*>/, ""), repository, dir }
  } catch {
    return null
  }
}

/** The packages whose code or assets are in the bundle, by "name@version". */
function bundledPackages(root: string, moduleIds: Iterable<string>, bundle: Rolldown.OutputBundle): Map<string, Pkg> {
  const dirs = new Set<string>(cssPackages(root))
  for (const id of moduleIds) {
    const dir = packageDir(id.replace(/^\0/, "").split("?")[0])
    if (dir) dirs.add(dir)
  }
  // Assets such as fonts, referenced from CSS.
  for (const out of Object.values(bundle)) {
    if (out.type !== "asset") continue
    for (const file of out.originalFileNames ?? []) {
      const dir = packageDir(path.resolve(root, file))
      if (dir) dirs.add(dir)
    }
  }
  const pkgs = new Map<string, Pkg>()
  for (const dir of dirs) {
    const pkg = readPkg(dir)
    // Sub-directories with their own package.json (e.g. "esm") point back to the real package.
    if (pkg) pkgs.set(`${pkg.name}@${pkg.version}`, pkg)
  }
  return pkgs
}

/** The listed packages as installed, and the rows that are missing, outdated or wrong. */
function checkListed(root: string, bundled: Map<string, Pkg>) {
  const listed = listedPackages(root)
  const problems: string[] = []
  const pkgs: Pkg[] = []
  for (const l of listed) {
    const p = bundled.get(`${l.name}@${l.version}`) ?? readPkg(path.join(root, "node_modules", l.name))
    if (!p || p.version !== l.version) problems.push(`  not installed in this version: ${row(l)}`)
    else if (p.license !== l.license) problems.push(`  ${l.name} declares "${p.license}": ${row(l)}`)
    else pkgs.push(p)
  }
  const names = new Set(listed.map((l) => `${l.name}@${l.version}`))
  for (const [key, p] of bundled) if (!names.has(key)) problems.push(`  missing (add with its license): ${row(p)}`)
  return { pkgs, problems: problems.sort() }
}

/** Fences text with more backticks than any run inside it (license texts keep their line breaks). */
function codeBlock(text: string): string {
  const longest = Math.max(0, ...(text.match(/`+/g) ?? []).map((run) => run.length))
  const fence = "`".repeat(Math.max(3, longest + 1))
  return `${fence}\n${text.trim()}\n${fence}`
}

function section(p: Pkg): string {
  const files = fs
    .readdirSync(p.dir)
    .filter((f) => licenseFile.test(f))
    .sort()
  const texts = files.map((f) => fs.readFileSync(path.join(p.dir, f), "utf8").trim())
  if (texts.length === 0) texts.push(declaredText(p))
  return `### ${p.name} ${p.version} (${p.license})\n\n${codeBlock(texts.join("\n\n"))}`
}

export function thirdPartyLicenses(): Plugin {
  let root = ""
  return {
    name: "kubedactyl-third-party-licenses",
    apply: "build",
    configResolved(config) {
      root = config.root
    },
    generateBundle(_, bundle) {
      const { pkgs, problems } = checkListed(root, bundledPackages(root, this.getModuleIds(), bundle))
      if (problems.length > 0)
        this.error(`THIRD_PARTY_NOTICES.md (${tableHeading}) does not match the build:\n${problems.join("\n")}`)
      this.emitFile({
        type: "asset",
        fileName: "licenses-npm.md",
        source: `## npm packages in the web interface (${pkgs.length})\n\n${pkgs.map(section).join("\n\n")}\n`,
      })
    },
  }
}
