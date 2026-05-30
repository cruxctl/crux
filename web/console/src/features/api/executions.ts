import { client } from "./client";
import type { Execution } from "./types";

export async function listExecutions(): Promise<Execution[]> {
  return client.fetch<Execution[]>("/v1/executions");
}

export async function getExecution(id: string): Promise<Execution | null> {
  try {
    return await client.fetch<Execution>(`/v1/executions/${id}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}
