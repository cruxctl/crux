import { client } from "./client";
import type { MCPServer, MCPTool, MCPCallResult } from "./types";

export async function listMCPServers(): Promise<MCPServer[]> {
  return client.fetch<MCPServer[]>("/v1/mcp/servers");
}

export async function getMCPServer(id: string): Promise<MCPServer | null> {
  try {
    return await client.fetch<MCPServer>(`/v1/mcp/servers/${id}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}

export async function listMCPTools(): Promise<MCPTool[]> {
  return client.fetch<MCPTool[]>("/v1/mcp/tools");
}

export async function callMCPTool(
  toolID: string,
  params?: Record<string, unknown>
): Promise<MCPCallResult> {
  return client.fetch<MCPCallResult>("/v1/mcp/calls", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tool_id: toolID, params }),
  });
}
