// Constants Vite puts into the build (vite.config.ts "define").
declare const __FRONTEND_VERSIONS__: Record<string, string>

// Some registry components (Magic UI particles) type timers as NodeJS.Timeout; the browser build has no
// Node types, so the name is declared here instead of editing the library file.
declare namespace NodeJS {
  type Timeout = ReturnType<typeof setTimeout>
}
