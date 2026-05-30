import { client } from "./client";
import type { Agent } from "./types";

export async function listAgents(): Promise<Agent[]> {
  return client.fetch<Agent[]>("/v1/agents");
}

export async function getAgent(name: string): Promise<Agent | null> {
  try {
    return await client.fetch<Agent>(`/v1/agents/${name}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}

export async function upsertAgent(agent: Agent): Promise<Agent> {
  return client.fetch<Agent>("/v1/agents", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(agent),
  });
}

export async function deleteAgent(name: string): Promise<void> {
  await client.fetch(`/v1/agents/${name}`, { method: "DELETE" });
}

export async function refreshAgents(): Promise<void> {
  await client.fetch("/v1/agents/refresh", { method: "POST" });
}
