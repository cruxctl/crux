import { client } from "./client";
import type { Session } from "./types";

export async function listSessions(): Promise<Session[]> {
  return client.fetch<Session[]>("/v1/sessions");
}

export async function getSession(id: string): Promise<Session | null> {
  try {
    return await client.fetch<Session>(`/v1/sessions/${id}`);
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}

export async function createSession(sess: Session): Promise<Session> {
  return client.fetch<Session>("/v1/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sess),
  });
}

export async function continueSession(
  id: string,
  req: { with: string; prompt?: string }
): Promise<Session> {
  return client.fetch<Session>(`/v1/sessions/${id}/continue`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
}

export async function stopSession(id: string): Promise<void> {
  await client.fetch(`/v1/sessions/${id}/stop`, { method: "POST" });
}

export async function replaySession(id: string): Promise<Session> {
  return client.fetch<Session>(`/v1/sessions/${id}/replay`, { method: "POST" });
}
