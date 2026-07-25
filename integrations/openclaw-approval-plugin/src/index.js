import { definePluginEntry } from "openclaw/plugin-sdk/plugin-entry";
import { createBridgeApprovalHandler } from "./approval-handler.js";

function normalizeIdList(value) {
  return Array.isArray(value)
    ? value.map((item) => String(item).trim()).filter(Boolean)
    : [];
}

export default definePluginEntry({
  id: "portable-bridge-approval",
  name: "Portable Bridge Approval",
  description: "Resolve Portable Bridge approvals before the agent queue.",
  register(api) {
    const config = api.pluginConfig ?? {};
    const allowedApproverIds = normalizeIdList(config.allowedApproverIds);
    const allowedConversationIds = normalizeIdList(config.allowedConversationIds);
    const operatorPath = String(config.operatorPath ?? "bridge-operator").trim();

    if (allowedApproverIds.length === 0 || allowedConversationIds.length === 0) {
      api.logger.warn?.(
        "portable-bridge-approval: no explicit approver/conversation allowlist; every callback will fail closed",
      );
    }

    api.registerInteractiveHandler({
      channel: "telegram",
      namespace: "bridge",
      handler: createBridgeApprovalHandler({
        allowedApproverIds,
        allowedConversationIds,
        operatorPath,
        logger: api.logger,
      }),
    });
  },
});
