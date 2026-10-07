import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  DownloadIcon,
  FileArchiveIcon,
  PackageOpenIcon,
  type LucideIcon,
} from "lucide-react"

import type { ServerJob } from "@/lib/types"

/** How the UI names each kind of background job: in toasts, while it runs, and its icon. */
export const jobKinds = {
  backup: { name: "Backup", verb: "Creating backup", icon: ArchiveIcon },
  restore: { name: "Restore", verb: "Restoring", icon: ArchiveRestoreIcon },
  pull: { name: "Download", verb: "Downloading", icon: DownloadIcon },
  compress: { name: "Compression", verb: "Compressing", icon: FileArchiveIcon },
  decompress: { name: "Extraction", verb: "Extracting", icon: PackageOpenIcon },
} satisfies Record<ServerJob["kind"], { name: string; verb: string; icon: LucideIcon }>
