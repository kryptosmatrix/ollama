import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BuildingStorefrontIcon } from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import {
  addMCPServer,
  approveMCPServer,
  listMCPServers,
  removeMCPServer,
  setMCPServerEnabled,
} from "@/api";
import type { MCPServer } from "@/gotypes";
import {
  parsePastedServers,
  presentMCPServer,
  type AddMCPServerInput,
} from "@/utils/mcpServers";
import MCPRegistryBrowse from "./MCPRegistryBrowse";

export default function MCPServers() {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [paste, setPaste] = useState("");
  const [showPaste, setShowPaste] = useState(false);
  const [showBrowse, setShowBrowse] = useState(false);

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
      <header className="flex items-center gap-3">
        <BuildingStorefrontIcon className="h-6 w-6 stroke-current text-neutral-500" />
        <h1 className="text-lg font-medium text-neutral-900 dark:text-neutral-100">
          MCP Servers
        </h1>
      </header>

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

      {showPaste && (
        <div className="flex flex-col gap-2">
          <Text>
            Paste a configuration block. Servers are added switched on but not
            approved — you will be asked to read each command line first.
          </Text>
          <textarea
            className="h-40 w-full rounded-lg border border-neutral-300 bg-white p-2 font-mono text-xs dark:border-neutral-700 dark:bg-neutral-900"
            value={paste}
            onChange={(event) => setPaste(event.target.value)}
            placeholder={
              '{\n  "mcpServers": {\n    "files": {\n      "command": "uvx",\n      "args": ["mcp-server-files"]\n    }\n  }\n}'
            }
          />
          <div>
            <Button
              disabled={paste.trim() === "" || addPasted.isPending}
              onClick={() => addPasted.mutate(paste)}
            >
              Add
            </Button>
          </div>
        </div>
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
}

export function MCPServerRow({
  server,
  onApprove,
  onToggle,
  onRemove,
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

      <div className="mt-3 flex flex-wrap gap-2">
        {presentation.needsApproval && (
          <Button onClick={onApprove}>Approve and run</Button>
        )}
        <Button onClick={onToggle}>
          {server.enabled ? "Switch off" : "Switch on"}
        </Button>
        <Button onClick={onRemove}>Remove</Button>
      </div>
    </li>
  );
}
