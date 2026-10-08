import { useAuth } from "@/hooks/use-auth"
import { useDraft } from "@/hooks/use-draft"
import type { TrafficPolicy } from "@/lib/types"

/** Fields of the create form that are kept when another egg is picked (a form draft). */
export function useSharedServerFields() {
  const { user } = useAuth()
  return useDraft({
    name: "",
    memory: 4096,
    cpu: 2000,
    disk: 10240,
    // No default: every game uses other ports, so the admin enters them.
    ports: [] as number[],
    lbIP: "",
    trafficPolicy: "Local" as TrafficPolicy,
    ipv6: false,
    // Empty means the default of the panel settings.
    storageClass: "",
    pool: "",
    start: true,
    owner: user.username,
  })
}

export type SharedServerFields = ReturnType<typeof useSharedServerFields>
