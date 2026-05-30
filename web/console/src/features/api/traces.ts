import { client } from "./client";
import type { AOSEvent } from "./types";

export async function listTraceEvents(): Promise<AOSEvent[]> {
  return client.fetch<AOSEvent[]>("/v1/aos/traces");
}
