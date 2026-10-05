// The legal texts an administrator can publish (settings → footer of every page).

import type { LegalTexts } from "@/lib/types"

export type LegalDoc = keyof LegalTexts

/** Title of each legal text, in the order of the footer links. */
export const legalTitles: Record<LegalDoc, string> = {
  legalNotice: "Legal notice",
  privacyPolicy: "Privacy policy",
}

export const legalDocs = Object.keys(legalTitles) as LegalDoc[]
