import assert from "node:assert/strict";
import test from "node:test";

const generationStopHelpers = await import("./generation-stop.ts").catch((error) => {
  if (error?.code !== "ERR_MODULE_NOT_FOUND") {
    throw error;
  }
  return {};
});
const messageSubmitHelpers = await import("./message-submit.ts");

test("cancels a started generation and refreshes persisted messages", async () => {
  assert.equal(typeof generationStopHelpers.cancelGenerationAndReload, "function");

  const calls = [];
  const canceled = await generationStopHelpers.cancelGenerationAndReload({
    accessToken: "token_existing",
    runID: "run_started",
    resolveAccessToken: async () => {
      calls.push("resolve-token");
      return "token_resolved";
    },
    cancelGeneration: async (accessToken, runID) => {
      calls.push(`cancel:${accessToken}:${runID}`);
      return { canceled: true };
    },
    reload: () => {
      calls.push("reload");
    },
  });

  assert.equal(canceled, true);
  assert.deepEqual(calls, ["cancel:token_existing:run_started", "reload"]);
});

test("resolves a missing access token before canceling", async () => {
  assert.equal(typeof generationStopHelpers.cancelGenerationAndReload, "function");

  const calls = [];
  const canceled = await generationStopHelpers.cancelGenerationAndReload({
    accessToken: null,
    runID: "run_resumed",
    resolveAccessToken: async () => {
      calls.push("resolve-token");
      return "token_resolved";
    },
    cancelGeneration: async (accessToken, runID) => {
      calls.push(`cancel:${accessToken}:${runID}`);
      return { canceled: true };
    },
    reload: () => {
      calls.push("reload");
    },
  });

  assert.equal(canceled, true);
  assert.deepEqual(calls, ["resolve-token", "cancel:token_resolved:run_resumed", "reload"]);
});

test("refreshes messages when canceling fails", async () => {
  assert.equal(typeof generationStopHelpers.cancelGenerationAndReload, "function");

  const calls = [];
  const canceled = await generationStopHelpers.cancelGenerationAndReload({
    accessToken: "token_existing",
    runID: "run_failed",
    resolveAccessToken: async () => "token_resolved",
    cancelGeneration: async () => {
      calls.push("cancel");
      throw new Error("network unavailable");
    },
    reload: () => {
      calls.push("reload");
    },
  });

  assert.equal(canceled, false);
  assert.deepEqual(calls, ["cancel", "reload"]);
});

test("only exposes assistant retry for persisted idle messages", () => {
  assert.equal(typeof messageSubmitHelpers.canRetryPersistedAssistantMessage, "function");

  const base = {
    publicID: "message_persisted",
    busy: false,
    isPending: false,
    isStreaming: false,
    readOnly: false,
  };

  assert.equal(messageSubmitHelpers.canRetryPersistedAssistantMessage(base), true);
  assert.equal(
    messageSubmitHelpers.canRetryPersistedAssistantMessage({
      ...base,
      publicID: "local-exchange-1-assistant",
    }),
    false,
  );
  assert.equal(messageSubmitHelpers.canRetryPersistedAssistantMessage({ ...base, busy: true }), false);
  assert.equal(messageSubmitHelpers.canRetryPersistedAssistantMessage({ ...base, isStreaming: true }), false);
  assert.equal(messageSubmitHelpers.canRetryPersistedAssistantMessage({ ...base, readOnly: true }), false);
});

test("restores a local draft only when the generation request was not dispatched", () => {
  assert.equal(typeof generationStopHelpers.resolveAbortedSubmissionDisposition, "function");
  assert.equal(
    generationStopHelpers.resolveAbortedSubmissionDisposition({ requestDispatched: false }),
    "restore_draft",
  );
  assert.equal(
    generationStopHelpers.resolveAbortedSubmissionDisposition({ requestDispatched: true }),
    "await_server_reconciliation",
  );
});
