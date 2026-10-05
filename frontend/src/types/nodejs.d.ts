// Some registry components (Magic UI particles) type timers as NodeJS.Timeout; the browser build has no
// Node types, so the name is declared here instead of editing the library file.
declare namespace NodeJS {
  type Timeout = ReturnType<typeof setTimeout>
}
