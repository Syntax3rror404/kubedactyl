import { toast } from "sonner"

// Toast wording used everywhere: success "<Thing> <past participle>" ("Backup deleted"),
// failure "Could not <action>" with the message of the server as description.

/** onError of a mutation: `useDeleteEgg(name, { onError: failed("delete the egg") })`. */
export const failed = (action: string) => (err: Error) =>
  toast.error(`Could not ${action}`, { description: err.message })
