import type { ChatEvent } from "@/gotypes";

/**
 * A tool call that is waiting on the user. The chat's streaming response is
 * blocked while this exists, so it must be answered or the call is refused
 * when the server's timeout expires.
 */
export interface PendingApproval {
  approvalId: string;
  toolName: string;
  scope: string;
  args: Record<string, unknown>;
}

/**
 * Reads a pending approval out of a chat event, or returns null when the event
 * is anything else.
 *
 * An event missing its identifier is ignored rather than shown: a prompt with
 * nothing to answer with would strand the chat, because clicking it could not
 * reach the waiting call.
 */
export function pendingApprovalFromEvent(
  event: Pick<
    ChatEvent,
    "eventName" | "toolName" | "approvalId" | "approvalScope" | "approvalArgs"
  >,
): PendingApproval | null {
  if (event.eventName !== "tool_approval") {
    return null;
  }
  if (!event.approvalId) {
    return null;
  }
  const toolName = event.toolName ?? "";
  return {
    approvalId: event.approvalId,
    toolName,
    scope: event.approvalScope || toolName,
    args: (event.approvalArgs as Record<string, unknown>) ?? {},
  };
}

/** One argument, rendered for a person to read before they agree to it. */
export interface ApprovalArgument {
  name: string;
  value: string;
}

/** How much of a single argument value is shown before it is cut short. */
export const MAX_ARGUMENT_LENGTH = 500;

/**
 * Renders a tool call's arguments in a stable order for display.
 *
 * The user is being asked to agree to a specific call, so they have to be able
 * to see what it would do. Values are stringified and capped: a server that
 * sends an enormous argument must not be able to push the buttons off the
 * screen, and the cut is marked so nothing appears to be the whole value when
 * it is not.
 */
export function approvalArguments(
  args: Record<string, unknown>,
): ApprovalArgument[] {
  return Object.keys(args ?? {})
    .sort()
    .map((name) => ({ name, value: formatArgumentValue(args[name]) }));
}

function formatArgumentValue(value: unknown): string {
  let text: string;
  if (typeof value === "string") {
    text = value;
  } else {
    try {
      text = JSON.stringify(value) ?? String(value);
    } catch {
      text = String(value);
    }
  }

  // Control characters would let a value forge the layout around it.
  // eslint-disable-next-line no-control-regex -- matching control characters is the point: they are what is being removed.
  text = text.replace(/[\u0000-\u001f\u007f]/g, "");

  if (text.length > MAX_ARGUMENT_LENGTH) {
    return `${text.slice(0, MAX_ARGUMENT_LENGTH)}… (truncated)`;
  }
  return text;
}

/** The answer being sent back for a pending approval. */
export interface ApprovalDecision {
  allow: boolean;
  remember?: boolean;
  rememberAll?: boolean;
}

/**
 * Builds the body posted back to the server.
 *
 * A refusal never carries a remember flag. The server ignores them on a
 * refusal too, but sending them would be describing an agreement that was not
 * given, and the wire should not say something the user did not.
 */
export function approvalRequestBody(
  approvalId: string,
  decision: ApprovalDecision,
): {
  approvalId: string;
  allow: boolean;
  remember: boolean;
  rememberAll: boolean;
} {
  if (!decision.allow) {
    return {
      approvalId,
      allow: false,
      remember: false,
      rememberAll: false,
    };
  }
  return {
    approvalId,
    allow: true,
    remember: Boolean(decision.remember) || Boolean(decision.rememberAll),
    rememberAll: Boolean(decision.rememberAll),
  };
}
