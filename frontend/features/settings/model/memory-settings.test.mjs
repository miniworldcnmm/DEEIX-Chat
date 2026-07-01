import assert from "node:assert/strict";
import test from "node:test";

const helpers = await import("./memory-settings.ts").catch(() => ({
  MEMORY_LIMIT: 0,
  parseMemoryEnabled: () => true,
}));

test("memory is disabled unless the persisted setting is true", () => {
  assert.equal(helpers.parseMemoryEnabled({}), false);
  assert.equal(helpers.parseMemoryEnabled({ "chat.memory_enabled": "false" }), false);
  assert.equal(helpers.parseMemoryEnabled({ "chat.memory_enabled": "true" }), true);
});

test("memory list uses the backend capacity", () => {
  assert.equal(helpers.MEMORY_LIMIT, 200);
});
