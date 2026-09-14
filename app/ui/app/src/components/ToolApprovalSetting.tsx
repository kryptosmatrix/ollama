import { ShieldCheckIcon } from "@heroicons/react/20/solid";
import { Switch } from "@/components/ui/switch";
import { Field, Label, Description } from "@/components/ui/fieldset";

export const AUTO_APPROVE_LABEL = "Approve all tool calls automatically";

/**
 * The one switch that stops the app asking before a tool runs.
 *
 * Off, which is the default, every tool call from a connected server waits
 * for the user to allow it in the chat. On, the call runs at once and no
 * prompt is shown. The description states whichever is currently true,
 * because the person reading it is deciding whether the tools they connected
 * may act without them.
 */
export function ToolApprovalSetting({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="overflow-hidden rounded-xl bg-white dark:bg-neutral-800">
      <div className="space-y-4 p-4">
        <Field>
          <div className="flex items-center justify-between">
            <div className="flex items-start space-x-3">
              <ShieldCheckIcon className="mt-1 h-5 w-5 flex-shrink-0 text-black dark:text-neutral-100" />
              <div>
                <Label>{AUTO_APPROVE_LABEL}</Label>
                <Description>
                  {checked
                    ? "Tool calls run at once without asking, from the next call on. Connected tools can read, write and run things on your behalf while this is on."
                    : "Tool calls that need approval wait for you in the chat, unless you have already allowed them for that chat."}
                </Description>
              </div>
            </div>
            <Switch
              checked={checked}
              onChange={onChange}
              aria-label={AUTO_APPROVE_LABEL}
            />
          </div>
        </Field>
      </div>
    </div>
  );
}
