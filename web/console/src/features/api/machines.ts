import { client } from "./client";
import type { Machine } from "./types";

export async function listMachines(): Promise<Machine[]> {
  return client.fetch<Machine[]>("/v1/machines");
}

export async function pairMachine(token: string): Promise<{ machine_id: string; status: string }> {
  return client.fetch<{ machine_id: string; status: string }>("/v1/machines/pair", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
}
