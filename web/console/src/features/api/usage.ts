import { client } from "./client";
import type { UsageReport, UsageLimits } from "./types";

export async function getUsage(
  agent?: string,
  project?: string,
  since?: string
): Promise<UsageReport[]> {
  const params = new URLSearchParams();
  if (agent) params.set("agent", agent);
  if (project) params.set("project", project);
  if (since) params.set("since", since);
  const qs = params.toString();
  return client.fetch<UsageReport[]>(`/v1/usage${qs ? "?" + qs : ""}`);
}

export async function getUsageLimits(): Promise<UsageLimits> {
  return client.fetch<UsageLimits>("/v1/usage/limits");
}

export async function setUsageLimits(limits: UsageLimits): Promise<UsageLimits> {
  return client.fetch<UsageLimits>("/v1/usage/limits", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(limits),
  });
}
