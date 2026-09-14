export function showAssistantMessageToolbar(
  message: {
    role?: string;
    content?: string;
    tool_calls?: unknown[] | null;
    tool_call?: unknown;
  },
  isStreaming: boolean,
): boolean {
  return (
    !isStreaming &&
    message.role === "assistant" &&
    !!message.content &&
    message.content.trim() !== "" &&
    (!message.tool_calls || message.tool_calls.length === 0) &&
    !message.tool_call
  );
}
