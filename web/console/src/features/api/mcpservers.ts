import { client } from "./client";
import type { MCPServer } from "./types";

export async function listMCPServers(
  namespace = "default"
): Promise<MCPServer[]> {
  return client.fetch<MCPServer[]>(
    `/v1/namespaces/${namespace}/mcpservers`
  );
}

export async function getMCPServer(
  namespace: string,
  name: string
): Promise<MCPServer | null> {
  try {
    return await client.fetch<MCPServer>(
      `/v1/namespaces/${namespace}/mcpservers/${name}`
    );
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}
