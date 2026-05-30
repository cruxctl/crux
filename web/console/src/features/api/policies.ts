import { client } from "./client";
import type { PolicyProfile, Decision } from "./types";

export async function listPolicies(): Promise<PolicyProfile[]> {
  return client.fetch<PolicyProfile[]>("/v1/policies");
}

export async function getPolicy(id: string): Promise<PolicyProfile | null> {
  try {
    return await client.fetch<PolicyProfile>(`/v1/policies/${id}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}

export async function createPolicy(policy: PolicyProfile): Promise<PolicyProfile> {
  return client.fetch<PolicyProfile>("/v1/policies", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(policy),
  });
}

export async function deletePolicy(id: string): Promise<void> {
  await client.fetch(`/v1/policies/${id}`, { method: "DELETE" });
}

export async function evaluatePolicy(
  id: string,
  input: unknown
): Promise<Decision> {
  return client.fetch<Decision>(`/v1/policies/${id}/evaluate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}
