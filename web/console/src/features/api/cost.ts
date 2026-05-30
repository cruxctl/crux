import { client } from "./client";
import type { CostReport } from "./types";

export async function getCosts(
  agent?: string,
  project?: string,
  since?: string
): Promise<CostReport[]> {
  const params = new URLSearchParams();
  if (agent) params.set("agent", agent);
  if (project) params.set("project", project);
  if (since) params.set("since", since);
  const qs = params.toString();
  return client.fetch<CostReport[]>(`/v1/costs${qs ? "?" + qs : ""}`);
}
