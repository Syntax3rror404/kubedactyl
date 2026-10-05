import { readFileSync } from "node:fs"
import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

import { thirdPartyLicenses } from "./scripts/licenses-plugin.ts"

// Versions of the main frontend libraries, shown to administrators on the Settings page.
const frontendVersions = Object.fromEntries(
  [
    ["React", "react"],
    ["Vite", "vite"],
    ["Tailwind CSS", "tailwindcss"],
    ["TanStack Query", "@tanstack/react-query"],
    ["React Router", "react-router"],
  ].map(([name, pkg]) => [name, JSON.parse(readFileSync(`node_modules/${pkg}/package.json`, "utf8")).version]),
)

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss(), thirdPartyLicenses()],
  define: { __FRONTEND_VERSIONS__: JSON.stringify(frontendVersions) },
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    // In development, forward API calls, the console websocket and the Swagger UI to the Go backend
    proxy: {
      "/api": { target: "http://localhost:8080", ws: true },
      "/swagger": "http://localhost:8080",
    },
  },
  build: {
    // The build goes straight into the Go package that embeds it via go:embed
    outDir: "../backend/web/dist",
    emptyOutDir: true,
  },
})
