import { client } from "./client";
import type { MetricValue } from "./types";

export async function getMetrics(): Promise<MetricValue[]> {
  return client.fetch<MetricValue[]>("/v1/metrics");
}
