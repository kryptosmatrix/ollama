import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BuildingStorefrontIcon } from "@heroicons/react/24/outline";
import { ArrowLeftIcon } from "@heroicons/react/20/solid";
import { useNavigate } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import {
  addMCPServer,
  approveMCPServer,
  listMCPServers,
  removeMCPServer,
  setMCPServerEnabled,
  signInMCPServer,
  signOutMCPServer,
} from "@/api";
import type { MCPServer } from "@/gotypes";
import {
  parsePastedServers,
  presentMCPServer,
  type AddMCPServerInput,
} from "@/utils/mcpServers";
import MCPRegistryBrowse from "./MCPRegistryBrowse";
import MCPLocalDiscovery from "./MCPLocalDiscovery";

/**
 * The page header, with the way back.
 *
 * The arrow goes to "/", not to a chat id: the index route already resolves
 * which home the user last had open. Working that out again here would be a
 * second implementation of it, and the two would drift.
 */
export function MCPPageHeader({ onBack }: { onBack: () => void }) {
  return (
    <header className="flex items-center gap-3">
      <button
        type="button"
        onClick={onBack}
        aria-label="Back to home"
        title="Back"
        className="-ml-1.5 rounded-full p-1.5 hover:bg-neutral-100 dark:hover:bg-neutral-800"
      >
        <ArrowLeftIcon className="h-5 w-5 text-neutral-700 dark:text-white" />
      </button>
      <BuildingStorefrontIcon className="h-6 w-6 stroke-current text-neutral-500" />
      <h1 className="text-lg font-medium text-neutral-900 dark:text-neutral-100">
        MCP Servers
      </h1>
    </header>
  );
}

/**
 * The paste-a-configuration block.
 *
 * Its own component so the field can be rendered — and its colours asserted —
 * without standing up a router and a query client for the whole page.
 */
export function PasteConfiguration({
  value,
  onChange,
  onAdd,
  pending,
}: {
  value: string;
  onChange: (value: string) => void;
  onAdd: () => void;
  pending: boolean;
}) {
  return (
    <div className="flex flex-col gap-2">
      <Text>
        Paste a configuration block. Servers are added switched on but not
        approved — you will be asked to read each command line first.
      </Text>
      {/* The text colour is not decoration: without it the field inherits
          black and what you typed is unreadable against the dark field. */}
      <textarea
        className="h-40 w-full rounded-lg border border-neutral-300 bg-white p-2 font-mono text-xs text-neutral-900 placeholder:text-neutral-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-label="MCP server configuration"
        placeholder={
          '{\n  "mcpServers": {\n    "files": {\n      "command": "uvx",\n      "args": ["mcp-server-files"]\n    }\n  }\n}'
        }
      />
      <div>
        <Button disabled={value.trim() === "" || pending} onClick={onAdd}>
          Add
        </Button>
      </div>
    </div>
  );
}

export default function MCPServers() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [paste, setPaste] = useState("");
  const [showPaste, setShowPaste] = useState(false);
  const [showBrowse, setShowBrowse] = useState(false);
  const [showLocal, setShowLocal] = useState(false);

  const servers = useQuery({
    queryKey: ["mcpServers"],
    queryFn: listMCPServers,
  });

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: ["mcpServers"] });

  const run = (action: () => Promise<void>) => {
    setError(null);
    action()
      .then(refresh)
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : String(err)),
      );
  };

  const addPasted = useMutation({
    mutationFn: async (text: string) => {
      for (const server of parsePastedServers(text)) {
        await addMCPServer(server);
      }
    },
    onSuccess: () => {
      setPaste("");
      setShowPaste(false);
      setError(null);
      refresh();
    },
    onError: (err: unknown) =>
      setError(err instanceof Error ? err.message : String(err)),
  });

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 p-6">
      <MCPPageHeader onBack={() => navigate({ to: "/" })} />

      <Text>
        MCP servers give models extra tools. Ollama will not run a server until
        you have read what it launches and approved it.
      </Text>

      {error && (
        <p className="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/40 dark:text-red-300">
          {error}
        </p>
      )}

      <div className="flex gap-2">
        <Button onClick={() => setShowBrowse((open) => !open)}>
          Browse the registry
        </Button>
        <Button onClick={() => setShowPaste((open) => !open)}>
          Add from configuration
        </Button>
        <Button onClick={() => setShowLocal((open) => !open)}>
          Find on this Mac
        </Button>
      </div>

      {showBrowse && (
        <MCPRegistryBrowse
          onInstall={async (request: AddMCPServerInput) => {
            // The ordinary add path: it lands unapproved and the user then
            // approves it below, where the command line is checked against
            // what is on disk.
            await addMCPServer(request);
            setShowBrowse(false);
            await refresh();
          }}
        />
      )}

      {showLocal && (
        <MCPLocalDiscovery
          onAdd={async (request: AddMCPServerInput) => {
            // The ordinary add path, the same one a pasted or registry server
            // takes: it lands unapproved and is approved below, where the
            // command line is checked against what is on disk.
            await addMCPServer(request);
            setShowLocal(false);
            await refresh();
          }}
        />
      )}

      {showPaste && (
        <PasteConfiguration
          value={paste}
          onChange={setPaste}
          onAdd={() => addPasted.mutate(paste)}
          pending={addPasted.isPending}
        />
      )}

      {servers.isLoading && <Text>Loading…</Text>}

      {servers.data?.length === 0 && (
        <Text>
          No MCP servers configured yet. Paste a configuration above, or add one
          with <span className="font-mono">ollama mcp add</span>.
        </Text>
      )}

      <ul className="flex flex-col gap-3">
        {servers.data?.map((server) => (
          <MCPServerRow
            key={server.name}
            server={server}
            onApprove={() =>
              run(() => approveMCPServer(server.name, server.runs))
            }
            onToggle={() =>
              run(() => setMCPServerEnabled(server.name, !server.enabled))
            }
            onRemove={() => run(() => removeMCPServer(server.name))}
            onSignIn={() => run(() => signInMCPServer(server.name))}
            onSignOut={() =>
              run(async () => {
                // A sign-out whose revocation failed still deleted the token,
                // so the request succeeded. What failed is worth saying out
                // loud, because withdrawing the token is then the user's to do
                // in that service's own account settings.
                const result = await signOutMCPServer(server.name);
                if (result.error) throw new Error(result.error);
              })
            }
          />
        ))}
      </ul>
    </div>
  );
}

interface MCPServerRowProps {
  server: MCPServer;
  onApprove: () => void;
  onToggle: () => void;
  onRemove: () => void;
  onSignIn: () => void;
  onSignOut: () => void;
}

export function MCPServerRow({
  server,
  onApprove,
  onToggle,
  onRemove,
  onSignIn,
  onSignOut,
}: MCPServerRowProps) {
  const presentation = presentMCPServer(server);

  return (
    <li className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800">
      <div className="flex flex-wrap items-baseline gap-2">
        <span className="font-medium text-neutral-900 dark:text-neutral-100">
          {server.name}
        </span>
        <span
          className={
            presentation.attention
              ? "text-xs text-amber-700 dark:text-amber-400"
              : "text-xs text-neutral-500 dark:text-neutral-400"
          }
        >
          {presentation.label}
        </span>
      </div>

      {/* What it runs, verbatim. This is what the user is agreeing to. */}
      <p className="mt-1 break-all font-mono text-xs text-neutral-600 dark:text-neutral-400">
        {server.runs}
      </p>

      {presentation.detail && (
        <p className="mt-1 text-xs text-neutral-600 dark:text-neutral-400">
          {presentation.detail}
        </p>
      )}

      {/* What this configuration costs, beside the approve button rather than
          buried somewhere else. Amber, not red: nothing here is broken. */}
      {server.warnings?.map((warning) => (
        <p
          key={warning}
          className="mt-1 text-xs text-amber-700 dark:text-amber-400"
        >
          Note: {warning}
        </p>
      ))}

      {server.skipped?.map((skipped) => (
        <p
          key={skipped.name}
          className="mt-1 text-xs text-amber-700 dark:text-amber-400"
        >
          Tool {skipped.name} was not offered: {skipped.reason}
        </p>
      ))}

      {server.tools && server.tools.length > 0 && (
        <details className="mt-2">
          <summary className="cursor-pointer text-xs text-neutral-500 dark:text-neutral-400">
            {server.tools.length} tools
          </summary>
          <dl className="mt-1 space-y-1">
            {server.tools.map((tool) => (
              <div key={tool.name} className="flex gap-2">
                <dt className="flex-shrink-0 font-mono text-xs text-neutral-500 dark:text-neutral-400">
                  {tool.name}
                </dt>
                {/* The description comes from the server, not from Ollama. */}
                <dd className="min-w-0 text-xs text-neutral-600 dark:text-neutral-400">
                  {tool.description}
                </dd>
              </div>
            ))}
          </dl>
        </details>
      )}

      {/* Where the credential ends up, said before one is created rather than
          after. The file store is weaker than the operating system keychain,
          and a user signing in to a third-party service is entitled to know. */}
      {server.canSignIn && !server.signedIn && server.tokenStore && (
        <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
          Your token will be kept in {server.tokenStore}
        </p>
      )}

      <div className="mt-3 flex flex-wrap gap-2">
        {presentation.needsApproval && (
          <Button onClick={onApprove}>Approve and run</Button>
        )}
        {server.canSignIn &&
          !server.signedIn &&
          !server.signingIn &&
          server.approved && <Button onClick={onSignIn}>Sign in</Button>}
        {server.canSignIn && server.signedIn && (
          <Button onClick={onSignOut}>Sign out</Button>
        )}
        <Button onClick={onToggle}>
          {server.enabled ? "Switch off" : "Switch on"}
        </Button>
        <Button onClick={onRemove}>Remove</Button>
      </div>
    </li>
  );
}
