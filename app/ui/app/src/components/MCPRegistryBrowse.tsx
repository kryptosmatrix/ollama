import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { browseMCPRegistry, resolveMCPRegistryEntry } from "@/api";
import type { MCPRegistryEntry } from "@/gotypes";
import type { AddMCPServerInput } from "@/utils/mcpServers";
import { installRequest, presentRegistryEntry } from "@/utils/mcpRegistry";

interface MCPRegistryBrowseProps {
  /** Called with the request that adds the entry, once the user has agreed. */
  onInstall: (request: AddMCPServerInput) => Promise<void>;
}

export default function MCPRegistryBrowse({
  onInstall,
}: MCPRegistryBrowseProps) {
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [pending, setPending] = useState<MCPRegistryEntry | null>(null);
  const [error, setError] = useState<string | null>(null);

  const results = useQuery({
    queryKey: ["mcpRegistry", search],
    queryFn: () => browseMCPRegistry(search),
  });

  // Ask the server again at the moment of the decision. The list may be
  // minutes old, and what the user reads before agreeing must be current.
  const beginInstall = async (entry: MCPRegistryEntry) => {
    setError(null);
    try {
      setPending(await resolveMCPRegistryEntry(entry.name));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const confirmInstall = async () => {
    if (!pending) return;
    setError(null);
    try {
      await onInstall(installRequest(pending));
      setPending(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <RegistrySearchForm
        query={query}
        onQueryChange={setQuery}
        onSubmit={() => setSearch(query)}
      />

      {/* Not decoration. The registry is open-publish and this must be said. */}
      <Text>
        Anyone can publish to the MCP registry. A listing is its publisher's
        claim, not something Ollama has checked.
      </Text>

      {error && (
        <p className="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/40 dark:text-red-300">
          {error}
        </p>
      )}

      {results.isLoading && <Text>Searching…</Text>}
      {results.isError && (
        <p className="text-sm text-red-700 dark:text-red-400">
          {results.error instanceof Error
            ? results.error.message
            : "Could not reach the registry."}
        </p>
      )}
      {results.data?.entries.length === 0 && <Text>Nothing found.</Text>}

      {pending && (
        <RegistryInstallConfirmation
          entry={pending}
          onConfirm={confirmInstall}
          onCancel={() => setPending(null)}
        />
      )}

      <ul className="flex flex-col gap-2">
        {results.data?.entries.map((entry) => (
          <RegistryResult
            key={entry.name}
            entry={entry}
            onInstall={() => beginInstall(entry)}
          />
        ))}
      </ul>
    </div>
  );
}

/**
 * The search field. Its own component so its colours can be asserted without
 * standing up a query client for the whole browse pane.
 */
export function RegistrySearchForm({
  query,
  onQueryChange,
  onSubmit,
}: {
  query: string;
  onQueryChange: (value: string) => void;
  onSubmit: () => void;
}) {
  return (
    <form
      className="flex gap-2"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      {/* The text colour is not decoration: without it the field inherits
          black and what you typed is unreadable against the dark field. */}
      <input
        className="flex-1 rounded-lg border border-neutral-300 bg-white px-2 py-1 text-sm text-neutral-900 placeholder:text-neutral-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500"
        value={query}
        onChange={(event) => onQueryChange(event.target.value)}
        placeholder="Search the MCP registry"
        aria-label="Search the MCP registry"
      />
      <Button type="submit">Search</Button>
    </form>
  );
}

export function RegistryResult({
  entry,
  onInstall,
}: {
  entry: MCPRegistryEntry;
  onInstall: () => void;
}) {
  const shown = presentRegistryEntry(entry);

  return (
    <li className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800">
      <div className="flex flex-wrap items-baseline gap-2">
        <span className="font-medium text-neutral-900 dark:text-neutral-100">
          {shown.heading}
        </span>
        {/* The publisher namespace is the only provenance on offer. */}
        <span className="text-xs text-neutral-500 dark:text-neutral-400">
          by {shown.publisher}
        </span>
      </div>

      {entry.description && (
        <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
          {entry.description}
        </p>
      )}

      {shown.repository && (
        <p className="mt-1 break-all text-xs text-neutral-500 dark:text-neutral-400">
          {shown.repository}
        </p>
      )}

      {shown.installable ? (
        <>
          <p className="mt-2 break-all font-mono text-xs text-neutral-600 dark:text-neutral-400">
            {shown.runs}
          </p>
          <div className="mt-2">
            <Button onClick={onInstall}>Add…</Button>
          </div>
        </>
      ) : (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-400">
          {shown.reason}
        </p>
      )}
    </li>
  );
}

export function RegistryInstallConfirmation({
  entry,
  onConfirm,
  onCancel,
}: {
  entry: MCPRegistryEntry;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const shown = presentRegistryEntry(entry);

  return (
    <div
      className="rounded-xl border border-amber-300 bg-amber-50 p-3 text-sm dark:border-amber-800/60 dark:bg-amber-950/40"
      role="alertdialog"
      aria-label={`Add ${entry.name}`}
    >
      <p className="text-neutral-900 dark:text-neutral-100">
        Add <span className="font-medium">{entry.name}</span> as{" "}
        <span className="font-mono">{entry.suggestedName}</span>?
      </p>

      {/* The whole point: the command line, verbatim, before anything is written. */}
      <p className="mt-2 break-all font-mono text-xs text-neutral-800 dark:text-neutral-200">
        {shown.runs}
      </p>

      {shown.variables.length > 0 && (
        <p className="mt-2 text-xs text-neutral-700 dark:text-neutral-300">
          You will need to set: {shown.variables.join(", ")}
        </p>
      )}

      <p className="mt-2 text-xs text-neutral-700 dark:text-neutral-300">
        It will be added switched on but not approved. Ollama will not run it
        until you approve it below.
      </p>

      <div className="mt-3 flex gap-2">
        <Button onClick={onConfirm}>Add</Button>
        <Button onClick={onCancel}>Cancel</Button>
      </div>
    </div>
  );
}
