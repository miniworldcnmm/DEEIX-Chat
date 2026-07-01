export const MEMORY_LIMIT = 200;

export function parseMemoryEnabled(settings: Record<string, string>): boolean {
  return settings["chat.memory_enabled"] === "true";
}
