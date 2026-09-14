import { createFileRoute } from "@tanstack/react-router";
import MCPServers from "@/components/MCPServers";

export const Route = createFileRoute("/mcp")({
  component: MCPServers,
});
