/** Keys of the egg "features" field the console page implements (matched in the browser). */
export const supportedFeatures = [
  "eula",
  "java_version",
  "pid_limit",
  "steam_disk_space",
  "gsl_token",
  "hytale_oauth",
] as const

export type EggFeature = (typeof supportedFeatures)[number]
