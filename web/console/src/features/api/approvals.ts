import { client } from "./client";
import type { ApprovalRecord } from "./types";

export async function listApprovals(): Promise<ApprovalRecord[]> {
  return client.fetch<ApprovalRecord[]>("/v1/approvals");
}

export async function getApproval(id: string): Promise<ApprovalRecord | null> {
  try {
    return await client.fetch<ApprovalRecord>(`/v1/approvals/${id}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}

export async function grantApproval(id: string): Promise<ApprovalRecord> {
  return client.fetch<ApprovalRecord>(`/v1/approvals/${id}/grant`, {
    method: "POST",
  });
}

export async function denyApproval(id: string): Promise<ApprovalRecord> {
  return client.fetch<ApprovalRecord>(`/v1/approvals/${id}/deny`, {
    method: "POST",
  });
}
