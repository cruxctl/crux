import { client } from "./client";
import type { AOSEvent } from "./types";

export async function listAOSEvents(
  filter?: Record<string, string>
): Promise<AOSEvent[]> {
  const params = filter ? new URLSearchParams(filter).toString() : "";
  return client.fetch<AOSEvent[]>(`/v1/aos/events${params ? "?" + params : ""}`);
}

export async function getAOSEvent(id: string): Promise<AOSEvent | null> {
  try {
    return await client.fetch<AOSEvent>(`/v1/aos/events/${id}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}

export async function listTraces(): Promise<AOSEvent[]> {
  return client.fetch<AOSEvent[]>("/v1/aos/traces");
}
