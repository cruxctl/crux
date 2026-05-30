import { getEnv } from "@/shared/env";

export interface ClientConfig {
  baseURL: string;
  apiKey?: string;
}

export class ApiError extends Error {
  status: number;
  body: string;
  constructor(status: number, body: string) {
    super(`API error ${status}: ${body}`);
    this.status = status;
    this.body = body;
  }
}

export class CruxClient {
  private cfg: ClientConfig;

  constructor(cfg?: Partial<ClientConfig>) {
    this.cfg = {
      baseURL:
        cfg?.baseURL ?? getEnv("CRUX_CONSOLE_API_URL", "http://localhost:4357"),
      apiKey: cfg?.apiKey ?? getEnv("CRUX_CONSOLE_API_KEY", ""),
    };
  }

  async fetch<T>(path: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers);
    headers.set("Accept", "application/json");
    if (this.cfg.apiKey) {
      headers.set("Authorization", `Bearer ${this.cfg.apiKey}`);
    }
    const res = await fetch(this.cfg.baseURL + path, {
      ...init,
      headers,
      cache: "no-store",
    });
    if (!res.ok) {
      throw new ApiError(res.status, await res.text());
    }
    return (await res.json()) as T;
  }
}

export const client = new CruxClient();
