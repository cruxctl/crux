import { client } from "./client";
import type { AgBOM } from "./types";

export async function getAgBOM(
  namespace: string,
  name: string
): Promise<AgBOM | null> {
  try {
    return await client.fetch<AgBOM>(
      `/v1/namespaces/${namespace}/agents/${name}/agbom`
    );
  } catch (e) {
    if ((e as { status?: number }).status === 404) return null;
    throw e;
  }
}
