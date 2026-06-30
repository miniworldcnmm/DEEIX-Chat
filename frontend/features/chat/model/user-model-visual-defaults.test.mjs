import assert from "node:assert/strict";
import test from "node:test";

const helpers = await import("./user-model-visual-defaults.ts").catch(() => ({
  resolveUserModelVisualFields: () => [],
  userModelOptionPayloadFromFields: () => ({}),
}));

const {
  resolveUserModelVisualFields,
  userModelOptionPayloadFromFields,
} = helpers;

test("creates thinking, effort, and temperature fields without administrator controls", () => {
  const fields = resolveUserModelVisualFields({
    protocol: "openai_responses",
    explicitOptions: {},
    platformDefaultOptions: {},
    userSettings: {
      "chat.default_thinking_enabled": "true",
      "chat.default_temperature": "0.8",
      "chat.default_reasoning_effort": "high",
    },
    modelOption: {},
  });

  assert.deepEqual(
    fields.map(({ id, path, value, active }) => ({ id, path, value, active })),
    [
      { id: "thinking_enabled", path: ["enable_thinking"], value: true, active: false },
      { id: "temperature", path: ["temperature"], value: 0.8, active: false },
      { id: "reasoning_effort", path: ["reasoning", "effort"], value: "high", active: false },
    ],
  );
});

test("uses explicit values before model and global defaults", () => {
  const fields = resolveUserModelVisualFields({
    protocol: "openai_chat_completions",
    explicitOptions: {
      enable_thinking: false,
      temperature: 0.2,
      reasoning_effort: "low",
    },
    platformDefaultOptions: { temperature: 1.2, reasoning_effort: "medium" },
    userSettings: {
      "chat.default_thinking_enabled": "true",
      "chat.default_temperature": "0.8",
      "chat.default_reasoning_effort": "high",
    },
    modelOption: {
      thinkingEnabled: true,
      temperature: 0.5,
      reasoningEffort: "xhigh",
    },
  });

  assert.deepEqual(fields.map(({ value, active }) => ({ value, active })), [
    { value: false, active: true },
    { value: 0.2, active: true },
    { value: "low", active: true },
  ]);
});

test("uses model defaults before global and platform defaults", () => {
  const fields = resolveUserModelVisualFields({
    protocol: "gemini_generate_content",
    explicitOptions: {},
    platformDefaultOptions: {
      generationConfig: {
        temperature: 1.2,
        thinkingConfig: { thinkingLevel: "low" },
      },
    },
    userSettings: {
      "chat.default_thinking_enabled": "true",
      "chat.default_temperature": "0.8",
      "chat.default_reasoning_effort": "medium",
    },
    modelOption: {
      thinkingEnabled: false,
      temperature: 0.4,
      reasoningEffort: "high",
    },
  });

  assert.deepEqual(
    fields.map(({ id, path, value }) => ({ id, path, value })),
    [
      { id: "thinking_enabled", path: ["enable_thinking"], value: false },
      { id: "temperature", path: ["generationConfig", "temperature"], value: 0.4 },
      {
        id: "reasoning_effort",
        path: ["generationConfig", "thinkingConfig", "thinkingLevel"],
        value: "high",
      },
    ],
  );
});

test("omits reasoning effort for protocols without a supported path", () => {
  const fields = resolveUserModelVisualFields({
    protocol: "anthropic_messages",
    explicitOptions: {},
    platformDefaultOptions: {},
    userSettings: {},
    modelOption: {},
  });

  assert.deepEqual(fields.map(({ id }) => id), ["thinking_enabled", "temperature"]);
});

test("extracts a complete single-model payload from resolved fields", () => {
  const fields = resolveUserModelVisualFields({
    protocol: "gemini_generate_content",
    explicitOptions: {
      enable_thinking: true,
      generationConfig: {
        temperature: 0.6,
        thinkingConfig: { thinkingLevel: "custom" },
      },
    },
    platformDefaultOptions: {},
    userSettings: {},
    modelOption: {},
  });

  assert.deepEqual(userModelOptionPayloadFromFields(fields), {
    thinkingEnabled: true,
    temperature: 0.6,
    reasoningEffort: "custom",
  });
});
