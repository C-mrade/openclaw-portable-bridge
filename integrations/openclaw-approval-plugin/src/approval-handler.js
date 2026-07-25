import { execFile as execFileCallback } from "node:child_process";
import { promisify } from "node:util";

const execFile = promisify(execFileCallback);
const REQUEST_ID_RE = /^[a-f0-9]{24}$/;
const COMPARE_CODE_RE = /^[a-f0-9]{6}$/;

export function parseBridgeCallback(payload) {
  const [action, requestId, compareCode, ...extra] = String(payload).split(":");
  if (
    extra.length > 0 ||
    !["approve", "reject"].includes(action) ||
    !REQUEST_ID_RE.test(requestId ?? "") ||
    !COMPARE_CODE_RE.test(compareCode ?? "")
  ) {
    return null;
  }
  return { action, requestId, compareCode };
}

function formatExpiry(value) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return null;
  return new Intl.DateTimeFormat("it-IT", {
    dateStyle: "short",
    timeStyle: "medium",
    timeZone: "Europe/Rome",
  }).format(date);
}

export function createBridgeApprovalHandler(options) {
  const runOperator =
    options.runOperator ??
    (async (args) => {
      const result = await execFile(options.operatorPath, args, {
        encoding: "utf8",
        timeout: 10_000,
        maxBuffer: 4 * 1024 * 1024,
        windowsHide: true,
      });
      return JSON.parse(result.stdout);
    });

  return async (ctx) => {
    const callback = parseBridgeCallback(ctx.callback.payload);
    if (!callback) return { handled: false };

    const senderId = String(ctx.senderId ?? "");
    const conversationId = String(ctx.conversationId ?? "");
    const callbackChatId = String(ctx.callback.chatId ?? "");
    const authorized =
      ctx.auth?.isAuthorizedSender === true &&
      ctx.isGroup === false &&
      options.allowedApproverIds.includes(senderId) &&
      options.allowedConversationIds.includes(conversationId) &&
      options.allowedConversationIds.includes(callbackChatId);

    if (!authorized) {
      options.logger?.warn?.(
        `portable-bridge-approval: denied callback sender=${senderId || "unknown"} conversation=${conversationId || "unknown"}`,
      );
      await ctx.respond.reply({
        text: "⛔ Approval Portable Bridge non autorizzato in questa conversazione.",
      });
      return { handled: true };
    }

    const pending = await runOperator(["pending"]);
    if (!Array.isArray(pending)) throw new Error("bridge-operator returned an invalid pending response");
    const request = pending.find((item) => item?.requestId === callback.requestId);

    if (!request) {
      await ctx.respond.editMessage({
        text: "⌛ SESSIONE SCADUTA\n\nLa richiesta non è più pendente. Nessuna approvazione eseguita.",
        buttons: [],
      });
      return { handled: true };
    }

    if (request.compareCode !== callback.compareCode) {
      options.logger?.warn?.(
        `portable-bridge-approval: comparison mismatch request=${callback.requestId}`,
      );
      await ctx.respond.editMessage({
        text: "⛔ CODICE NON COINCIDENTE\n\nLa richiesta non è stata approvata.",
        buttons: [],
      });
      return { handled: true };
    }

    const durationSeconds = Number(request.durationSeconds);
    const minutes = Math.ceil(durationSeconds / 60);
    if (!Number.isInteger(minutes) || minutes < 1 || minutes > 60) {
      throw new Error("pending request has an invalid duration");
    }

    if (callback.action === "reject") {
      await runOperator(["reject", callback.requestId]);
      await ctx.respond.editMessage({
        text: [
          `❌ SESSIONE RIFIUTATA — ${request.hostname ?? "guest"}`,
          "",
          `Codice verificato: ${callback.compareCode}`,
          "Nessuna capacità concessa.",
        ].join("\n"),
        buttons: [],
      });
      return { handled: true };
    }

    const result = await runOperator(["approve", callback.requestId, String(minutes)]);
    const expiry = formatExpiry(result?.expiresAt);
    await ctx.respond.editMessage({
      text: [
        `✅ SESSIONE APPROVATA — ${request.hostname ?? "guest"}`,
        "",
        `Codice verificato: ${callback.compareCode}`,
        `Durata: ${Number(result?.minutes) || minutes} minuti`,
        ...(expiry ? [`Scadenza: ${expiry}`] : []),
        "Capacità concesse: esclusivamente quelle richieste dal guest.",
      ].join("\n"),
      buttons: [],
    });
    return { handled: true };
  };
}
