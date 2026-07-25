import assert from "node:assert/strict";
import test from "node:test";
import { createBridgeApprovalHandler, parseBridgeCallback } from "./approval-handler.js";

function context(payload, overrides = {}) {
  const edits = [];
  const replies = [];
  return {
    edits,
    replies,
    value: {
      callback: {
        payload,
        chatId: "881753004",
      },
      senderId: "881753004",
      conversationId: "881753004",
      isGroup: false,
      auth: { isAuthorizedSender: true },
      respond: {
        editMessage: async (message) => edits.push(message),
        reply: async (message) => replies.push(message),
      },
      ...overrides,
    },
  };
}

const request = {
  requestId: "9d1a817efe6b065c6eb5754f",
  compareCode: "610f49",
  durationSeconds: 1800,
  hostname: "STANPC",
};

test("parses only bounded Bridge callbacks", () => {
  assert.deepEqual(
    parseBridgeCallback("approve:9d1a817efe6b065c6eb5754f:610f49"),
    {
      action: "approve",
      requestId: "9d1a817efe6b065c6eb5754f",
      compareCode: "610f49",
    },
  );
  assert.equal(parseBridgeCallback("approve:anything:610f49"), null);
  assert.equal(parseBridgeCallback("approve:9d1a817efe6b065c6eb5754f:610f49:extra"), null);
});

test("approves the exact pending request and clears buttons", async () => {
  const calls = [];
  const ctx = context("approve:9d1a817efe6b065c6eb5754f:610f49");
  const handler = createBridgeApprovalHandler({
    allowedApproverIds: ["881753004"],
    allowedConversationIds: ["881753004"],
    operatorPath: "bridge-operator",
    runOperator: async (args) => {
      calls.push(args);
      if (args[0] === "pending") return [request];
      return {
        status: "approved",
        requestId: request.requestId,
        minutes: 30,
        expiresAt: "2026-07-25T09:05:00Z",
      };
    },
  });

  assert.deepEqual(await handler(ctx.value), { handled: true });
  assert.deepEqual(calls, [
    ["pending"],
    ["approve", request.requestId, "30"],
  ]);
  assert.deepEqual(ctx.edits[0].buttons, []);
  assert.match(ctx.edits[0].text, /SESSIONE APPROVATA/);
});

test("rejects unauthorized conversations without touching the broker", async () => {
  let called = false;
  const ctx = context("approve:9d1a817efe6b065c6eb5754f:610f49", {
    conversationId: "other-chat",
  });
  const handler = createBridgeApprovalHandler({
    allowedApproverIds: ["881753004"],
    allowedConversationIds: ["881753004"],
    operatorPath: "bridge-operator",
    runOperator: async () => {
      called = true;
    },
  });

  assert.deepEqual(await handler(ctx.value), { handled: true });
  assert.equal(called, false);
  assert.equal(ctx.replies.length, 1);
});

test("marks stale requests expired and clears buttons", async () => {
  const ctx = context("approve:9d1a817efe6b065c6eb5754f:610f49");
  const handler = createBridgeApprovalHandler({
    allowedApproverIds: ["881753004"],
    allowedConversationIds: ["881753004"],
    operatorPath: "bridge-operator",
    runOperator: async () => [],
  });

  assert.deepEqual(await handler(ctx.value), { handled: true });
  assert.deepEqual(ctx.edits[0].buttons, []);
  assert.match(ctx.edits[0].text, /SCADUTA/);
});

test("rejects the exact pending request and grants no capability", async () => {
  const calls = [];
  const ctx = context("reject:9d1a817efe6b065c6eb5754f:610f49");
  const handler = createBridgeApprovalHandler({
    allowedApproverIds: ["881753004"],
    allowedConversationIds: ["881753004"],
    operatorPath: "bridge-operator",
    runOperator: async (args) => {
      calls.push(args);
      if (args[0] === "pending") return [request];
      return { status: "rejected" };
    },
  });

  assert.deepEqual(await handler(ctx.value), { handled: true });
  assert.deepEqual(calls, [
    ["pending"],
    ["reject", request.requestId],
  ]);
  assert.deepEqual(ctx.edits[0].buttons, []);
  assert.match(ctx.edits[0].text, /SESSIONE RIFIUTATA/);
});

test("fails closed when the comparison code differs", async () => {
  const calls = [];
  const ctx = context("approve:9d1a817efe6b065c6eb5754f:ffffff");
  const handler = createBridgeApprovalHandler({
    allowedApproverIds: ["881753004"],
    allowedConversationIds: ["881753004"],
    operatorPath: "bridge-operator",
    runOperator: async (args) => {
      calls.push(args);
      return [request];
    },
  });

  assert.deepEqual(await handler(ctx.value), { handled: true });
  assert.deepEqual(calls, [["pending"]]);
  assert.match(ctx.edits[0].text, /NON COINCIDENTE/);
});
