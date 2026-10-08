// Eggs: game server templates (Pterodactyl/Pelican).

import type { V1Alpha1Egg } from "@/lib/types/api.gen"
import type { Stored } from "@/lib/types/common"

export type {
  EgglibraryEgg as LibraryEgg,
  EgglibraryRepository as LibraryRepository,
  HttpapiLibraryEgg as LibraryEggContent,
  HttpapiLibraryList as LibraryList,
  HttpapiLibraryRepositoryCheck as LibraryRepositoryCheck,
  V1Alpha1ConfigFile as ConfigFile,
  V1Alpha1ConfigReplace as ConfigReplace,
  V1Alpha1EggSpec as EggSpec,
  V1Alpha1EggVariable as EggVariable,
} from "@/lib/types/api.gen"

export type Egg = Stored<V1Alpha1Egg>
export type EggList = { items: Egg[] }

/** Formats of GET /eggs/{egg}/export (query parameter). */
export type EggExportFormat = "yaml" | "json" | "ptdl"
