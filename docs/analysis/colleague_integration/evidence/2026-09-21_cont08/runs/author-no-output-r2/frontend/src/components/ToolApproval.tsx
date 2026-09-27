import { ShieldExclamationIcon } from "@heroicons/react/20/solid";
import { Button } from "@/components/ui/button";
import {
  approvalArguments,
  type ApprovalDecision,
  type PendingApproval,
} from "@/utils/toolApproval";

interface ToolApprovalProps {
  approval: PendingApproval;
  onAnswer: (decision: ApprovalDecision) => void;
  error?: string | null;
  isAnswering?: boolean;
}

/**
 * Asks the user to approve one tool call.
 *
 * The chat is blocked while this is on screen, so it says what is being asked
 * for rather than merely that something is. The arguments are shown because
 * the user is agreeing to this call, not to the tool in the abstract, and they
 * come from a server Ollama does not control — they are displayed as plain
 * text and never interpreted.
 */
export function ToolApproval({
  approval,
  onAnswer,
  error,
  isAnswering,
}: ToolApprovalProps) {
  const args = approvalArguments(approval.args);

  return (
    <div
      className="mb-2 rounded-xl border border-amber-300 bg-amber-50 p-3 text-sm dark:border-amber-800/60 dark:bg-amber-950/40"
      role="alertdialog"
      aria-label={`Approve ${approval.toolName}`}
    >
      <div className="flex items-start gap-2">
        <ShieldExclamationIcon className="mt-0.5 h-5 w-5 flex-shrink-0 text-amber-600 dark:text-amber-400" />
        <div className="min-w-0 flex-1">
          <p className="text-neutral-900 dark:text-neutral-100">
            Allow <span className="font-medium">{approval.toolName}</span> to
            run?
          </p>

          {args.length > 0 && (
            <dl className="mt-2 space-y-1">
              {args.map((argument) => (
                <div key={argument.name} className="flex gap-2 overflow-hidden">
                  <dt className="flex-shrink-0 font-mono text-xs text-neutral-500 dark:text-neutral-400">
                    {argument.name}
                  </dt>
                  <dd className="min-w-0 break-all font-mono text-xs text-neutral-700 dark:text-neutral-300">
                    {argument.value}
                  </dd>
                </div>
              ))}
            </dl>
          )}

          {error && (
            <p className="mt-2 text-xs text-red-600 dark:text-red-400">
              {error}
            </p>
          )}

          <div className="mt-3 flex flex-wrap gap-2">
            <Button
              disabled={isAnswering}
              onClick={() => onAnswer({ allow: true })}
            >
              Allow once
            </Button>
            <Button
              disabled={isAnswering}
              onClick={() => onAnswer({ allow: true, remember: true })}
            >
              Always allow {approval.scope}
            </Button>
            <Button
              disabled={isAnswering}
              onClick={() => onAnswer({ allow: false })}
            >
              Decline
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ToolApproval;
