export type UserModelOptionDefaults = {
  thinkingEnabled?: boolean;
  temperature?: number;
  reasoningEffort?: string;
};

export type UserModelVisualFieldID = "thinking_enabled" | "temperature" | "reasoning_effort";

export type UserModelVisualField = {
  id: UserModelVisualFieldID;
  path: string[];
  value: boolean | number | string;
  active: boolean;
  kind: "boolean" | "number" | "select";
  options?: string[];
  min?: number;
  max?: number;
  step?: number;
};

type ResolveUserModelVisualFieldsInput = {
  protocol: string;
  explicitOptions: Record<string, unknown>;
  platformDefaultOptions: Record<string, unknown>;
  userSettings: Record<string, string>;
  modelOption: UserModelOptionDefaults;
};

const REASONING_EFFORT_OPTIONS = ["minimal", "low", "medium", "high", "xhigh", "max"];

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function readPath(source: Record<string, unknown>, path: string[]): unknown {
  let current: unknown = source;
  for (const segment of path) {
    if (!isRecord(current) || !Object.prototype.hasOwnProperty.call(current, segment)) {
      return undefined;
    }
    current = current[segment];
  }
  return current;
}

function temperaturePath(protocol: string): string[] {
  return protocol === "gemini_generate_content"
    ? ["generationConfig", "temperature"]
    : ["temperature"];
}

function reasoningEffortPath(protocol: string): string[] | null {
  switch (protocol) {
    case "openai_chat_completions":
    case "openrouter_chat_completions":
      return ["reasoning_effort"];
    case "openai_responses":
    case "openrouter_responses":
    case "xai_responses":
      return ["reasoning", "effort"];
    case "gemini_generate_content":
      return ["generationConfig", "thinkingConfig", "thinkingLevel"];
    default:
      return null;
  }
}

function readBoolean(source: Record<string, unknown>, paths: string[][]): boolean | undefined {
  for (const path of paths) {
    const value = readPath(source, path);
    if (typeof value === "boolean") {
      return value;
    }
    if (path.join(".") === "thinking.type" && typeof value === "string") {
      const normalized = value.trim().toLowerCase();
      if (normalized === "enabled" || normalized === "adaptive") {
        return true;
      }
      if (normalized === "disabled") {
        return false;
      }
    }
  }
  return undefined;
}

function readNumber(source: Record<string, unknown>, paths: string[][]): number | undefined {
  for (const path of paths) {
    const value = readPath(source, path);
    if (typeof value === "number" && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

function readString(source: Record<string, unknown>, paths: string[][]): string | undefined {
  for (const path of paths) {
    const value = readPath(source, path);
    if (typeof value === "string" && value.trim() !== "") {
      return value.trim();
    }
  }
  return undefined;
}

function parseTemperature(raw: string | undefined): number | undefined {
  if (raw === undefined || raw.trim() === "") {
    return undefined;
  }
  const value = Number(raw);
  return Number.isFinite(value) && value >= 0 && value <= 2 ? value : undefined;
}

function uniqueOptions(value: string): string[] {
  return Array.from(new Set([...REASONING_EFFORT_OPTIONS, value].filter(Boolean)));
}

export function resolveUserModelVisualFields({
  protocol,
  explicitOptions,
  platformDefaultOptions,
  userSettings,
  modelOption,
}: ResolveUserModelVisualFieldsInput): UserModelVisualField[] {
  const thinkingPaths = [["enable_thinking"], ["thinking", "type"]];
  const explicitThinking = readBoolean(explicitOptions, thinkingPaths);
  const platformThinking = readBoolean(platformDefaultOptions, thinkingPaths);
  const globalThinking = userSettings["chat.default_thinking_enabled"] !== "false";

  const currentTemperaturePath = temperaturePath(protocol);
  const explicitTemperature = readNumber(explicitOptions, [currentTemperaturePath, ["temperature"]]);
  const platformTemperature = readNumber(platformDefaultOptions, [currentTemperaturePath, ["temperature"]]);
  const globalTemperature = parseTemperature(userSettings["chat.default_temperature"]);

  const currentEffortPath = reasoningEffortPath(protocol);
  const effortPaths = currentEffortPath
    ? [
        currentEffortPath,
        ["reasoning_effort"],
        ["effort"],
        ["reasoning", "effort"],
        ["generationConfig", "thinkingConfig", "thinkingLevel"],
      ]
    : [];
  const explicitEffort = readString(explicitOptions, effortPaths);
  const platformEffort = readString(platformDefaultOptions, effortPaths);
  const globalEffort = userSettings["chat.default_reasoning_effort"]?.trim() ?? "";

  const fields: UserModelVisualField[] = [
    {
      id: "thinking_enabled",
      path: ["enable_thinking"],
      value: explicitThinking ?? modelOption.thinkingEnabled ?? globalThinking ?? platformThinking ?? true,
      active: explicitThinking !== undefined,
      kind: "boolean",
    },
    {
      id: "temperature",
      path: currentTemperaturePath,
      value: explicitTemperature ?? modelOption.temperature ?? globalTemperature ?? platformTemperature ?? 1,
      active: explicitTemperature !== undefined,
      kind: "number",
      min: 0,
      max: 2,
      step: 0.1,
    },
  ];

  if (currentEffortPath) {
    const value = (explicitEffort ?? modelOption.reasoningEffort?.trim() ?? globalEffort) || platformEffort || "";
    fields.push({
      id: "reasoning_effort",
      path: currentEffortPath,
      value,
      active: explicitEffort !== undefined,
      kind: "select",
      options: uniqueOptions(value),
    });
  }

  return fields;
}

export function userModelOptionPayloadFromFields(fields: UserModelVisualField[]): UserModelOptionDefaults {
  const payload: UserModelOptionDefaults = {};
  for (const field of fields) {
    if (field.id === "thinking_enabled" && typeof field.value === "boolean") {
      payload.thinkingEnabled = field.value;
    } else if (field.id === "temperature" && typeof field.value === "number") {
      payload.temperature = field.value;
    } else if (field.id === "reasoning_effort" && typeof field.value === "string" && field.value.trim() !== "") {
      payload.reasoningEffort = field.value.trim();
    }
  }
  return payload;
}
