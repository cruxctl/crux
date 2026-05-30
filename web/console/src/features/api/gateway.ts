import { client } from "./client";
import type { GatewayStatus, GatewayRoute } from "./types";

export async function getGatewayStatus(): Promise<GatewayStatus> {
  return client.fetch<GatewayStatus>("/v1/gateway/status");
}

export async function getGatewayRoutes(): Promise<GatewayRoute[]> {
  return client.fetch<GatewayRoute[]>("/v1/gateway/routes");
}
