import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { discoverMCPServers, probeMCPServers } from "@/api";
import type { MCPDiscoveredServer } from "@/gotypes";
import type { AddMCPServerInput } from "@/utils/mcpServers";
import { addRequestForDiscovered } from "@/utils/mcpDiscovery";

interface MCPLocalDiscoveryProps {
  /** Called with the request that adds the server, once the user has agreed. */
  onAdd: (request: AddMCPServerInput) => Promise<void>;
}

/**
 * Servers already on this machine.
 *
 * The two kinds are kept in separate lists rather than merged. "Another
 * application is configured with this" and "something is answering on this
 * port right now" are different claims, and a user deciding whether to trust a
 * server needs to know which one they are looking at.
 */
export default function MCPLocalDiscovery({ onAdd }: MCPLocalDiscoveryProps) {
  const [error, setError] = useState<string | null>(null);
  const [showSearched, setShowSearched] = useState(false);

  const configured = useQuery({
    queryKey: ["mcpDiscovery"],
    queryFn: discoverMCPServers,
  });

  // The probe contacts things, so it never runs on its own. Something has to
  // ask for it.
  const probe = useMutation({
    mutationFn: probeMCPServers,
    onError: (err: unknown) =>
      setError(err instanceof Error ? err.message : String(err)),
  });

  const add = async (server: MCPDiscoveredServer) => {
    setError(null);
    try {
      await onAdd(addRequestForDiscovered(server));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <Text>
        Servers already on this machine. Adding one copies its configuration
        here; it arrives switched on but not approved, and Ollama will not run
        it until you have read what it launches.
      </Text>

      {error && (
        <p className="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/40 dark:text-red-300">
          {error}
        </p>
      )}

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
          Configured in other applications
        </h2>

        {configured.isLoading && <Text>Looking…</Text>}
        {configured.isError && (
          <p className="text-sm text-red-700 dark:text-red-400">
            {configured.error instanceof Error
              ? configured.error.message
              : "Could not search this machine."}
          </p>
        )}
        {configured.data?.servers.length === 0 && (
          <Text>
            No other MCP client on this machine has servers configured.
          </Text>
        )}

        <ul className="flex flex-col gap-2">
          {configured.data?.servers.map((server) => (
            <DiscoveredRow
              key={`${server.sources.join(",")}:${server.name}`}
              server={server}
              onAdd={() => add(server)}
            />
          ))}
        </ul>

        {/* What was searched, on request. A list of results says nothing about
            where it did not look. */}
        {configured.data && configured.data.searched.length > 0 && (
          <div>
            <button
              type="button"
              className="text-xs text-neutral-500 underline dark:text-neutral-400"
              onClick={() => setShowSearched((open) => !open)}
            >
              {showSearched ? "Hide" : "Show"} the files that were searched
            </button>
            {showSearched && (
              <ul className="mt-1 flex flex-col gap-0.5">
                {configured.data.searched.map((path) => (
                  <li
                    key={path}
                    className="break-all font-mono text-xs text-neutral-500 dark:text-neutral-400"
                  >
                    {path}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </section>

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
          Answering on this machine
        </h2>
        <Text>
          This contacts every port with something listening on it, on this
          machine only, with the MCP handshake — so what it finds is a server
          that answered, not a port that happened to be open. It takes a few
          seconds.
        </Text>

        <div>
          <Button disabled={probe.isPending} onClick={() => probe.mutate()}>
            {probe.isPending ? "Checking…" : "Check what is listening"}
          </Button>
        </div>

        {probe.data?.error && (
          <p className="text-xs text-amber-700 dark:text-amber-400">
            {probe.data.error}
          </p>
        )}
        {probe.data && !probe.data.error && probe.data.servers.length === 0 && (
          <Text>Nothing on this machine answered an MCP handshake.</Text>
        )}

        <ul className="flex flex-col gap-2">
          {probe.data?.servers.map((server) => (
            <DiscoveredRow
              key={server.runs}
              server={server}
              onAdd={() => add(server)}
            />
          ))}
        </ul>
      </section>
    </div>
  );
}

export function DiscoveredRow({
  server,
  onAdd,
}: {
  server: MCPDiscoveredServer;
  onAdd: () => void;
}) {
  return (
    <li className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800">
      <div className="flex flex-wrap items-baseline gap-2">
        <span className="font-medium text-neutral-900 dark:text-neutral-100">
          {server.name}
        </span>
        {/* Every application configured with it, not just the first. One
            server registered in two places is one server, and which two is
            what tells the user they are looking at their own setup rather
            than a duplicate. */}
        <span className="text-xs text-neutral-500 dark:text-neutral-400">
          {server.sources.join(", ")}
        </span>
      </div>

      {/* What it runs, verbatim. This is what would be added. */}
      <p className="mt-1 break-all font-mono text-xs text-neutral-600 dark:text-neutral-400">
        {server.runs}
      </p>

      {server.paths?.map((path) => (
        <p
          key={path}
          className="mt-1 break-all text-xs text-neutral-500 dark:text-neutral-400"
        >
          {path}
        </p>
      ))}

      {/* What discovery changed or left behind, beside the button rather than
          after the click. */}
      {server.notes?.map((note) => (
        <p
          key={note}
          className="mt-1 text-xs text-amber-700 dark:text-amber-400"
        >
          Note: {note}
        </p>
      ))}

      {server.problem ? (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-400">
          Cannot be added as it stands: {server.problem}
        </p>
      ) : (
        <div className="mt-2">
          <Button onClick={onAdd}>Add…</Button>
        </div>
      )}
    </li>
  );
}
