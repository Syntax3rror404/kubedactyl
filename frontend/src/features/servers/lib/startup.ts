import type { EggSpec } from "@/lib/types"

/** How a server starts: its own command (admins; empty for none) or the name of one of the egg's commands. */
export type StartupChoice = { startup: string; startupName: string }

/** The egg command a choice picks: the named one, else the default (first). */
export function pickedCommand(egg: EggSpec | undefined, choice: StartupChoice) {
  const commands = egg?.startupCommands ?? []
  return commands.find((c) => c.name === choice.startupName) ?? commands[0]
}

/** The (unsubstituted) command a server runs with the choice: its own, else the picked egg command. */
export function startupCommandOf(egg: EggSpec | undefined, choice: StartupChoice): string {
  return choice.startup || pickedCommand(egg, choice)?.command || ""
}
