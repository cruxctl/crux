// Types match cruxd pkg/cruxapi types exactly

export type AgentStatus = "ready" | "busy" | "error" | "offline";

export interface Agent {
  id: string;
  name: string;
  description?: string;
  labels?: Record<string, string>;
  command: CommandSpec;
  status: AgentStatus;
  createdAt: string;
  updatedAt: string;
}

export interface CommandSpec {
  path: string;
  args?: string[];
  env?: Record<string, string>;
  workingDir?: string;
  timeoutSeconds?: number;
}

export type SessionStatus = "created" | "running" | "completed" | "failed" | "canceled" | "continued";

export interface Session {
  id: string;
  agent_id: string;
  project_id?: string;
  user_id?: string;
  machine_id?: string;
  status: SessionStatus;
  started_at: string;
  ended_at?: string;
  parent_session_id?: string;
  fallback_from_session_id?: string;
}

export interface PolicyProfile {
  apiVersion?: string;
  kind?: string;
  metadata: PolicyMetadata;
  rules: PolicyRule[];
}

export interface PolicyMetadata {
  id: string;
}

export interface PolicyRule {
  id: string;
  match: Record<string, unknown>;
  action: string;
  rate?: PolicyRate;
}

export interface PolicyRate {
  requests_per_minute?: number;
  tokens_per_day?: number;
}

export type ApprovalStatus = "pending" | "granted" | "denied" | "expired";

export interface ApprovalRecord {
  id: string;
  tool_call: string;
  policy_id: string;
  agent_id: string;
  session_id?: string;
  status: ApprovalStatus;
  approvers?: string[];
  created_at: string;
  expires_at: string;
}

export interface AOSEvent {
  schema: string;
  event_id: string;
  timestamp: string;
  event_type: string;
  actor: AOSActor;
  project?: ProjectRef;
  agent?: AgentRef;
  tool?: ToolRef;
  policy?: PolicyRef;
  trace?: TraceRef;
  payload?: Record<string, unknown>;
}

export interface AOSActor {
  user_id?: string;
  machine_id?: string;
  agent_id?: string;
  session_id?: string;
}

export interface ProjectRef {
  id: string;
  path_hash?: string;
}

export interface AgentRef {
  name: string;
  provider?: string;
  version?: string;
}

export interface ToolRef {
  name: string;
  server?: string;
  transport?: string;
}

export interface PolicyRef {
  decision: string;
}

export interface TraceRef {
  trace_id: string;
  span_id?: string;
}

export interface Execution {
  id: string;
  agentName: string;
  prompt: string;
  workingDir?: string;
  resumeSession?: string;
  sourceExecutionId?: string;
  fallbackAgents?: string[];
  status: string;
  exitCode: number;
  stdout?: string;
  stderr?: string;
  error?: string;
  queuedAt: string;
  startedAt?: string;
  endedAt?: string;
}

export interface CostReport {
  agent_id: string;
  session_id?: string;
  tokens_in: number;
  tokens_out: number;
  usd: number;
  model?: string;
}

export interface UsageReport {
  agent_id: string;
  project_id?: string;
  period: string;
  tokens_in: number;
  tokens_out: number;
  usd: number;
  sessions: number;
}

export interface UsageLimits {
  daily_usd_warn: number;
  daily_usd_block: number;
  agent_id?: string;
  project_id?: string;
}

export interface GatewayStatus {
  enabled: boolean;
  version?: string;
  uptime?: string;
  routes?: string[];
  injected_agents?: string[];
  ready: boolean;
}

export interface GatewayRoute {
  id: string;
  path: string;
  target: string;
  agent_id?: string;
  methods?: string[];
  headers?: Record<string, string>;
  enabled: boolean;
  created_at: string;
}

export interface MCPServer {
  id: string;
  name: string;
  url?: string;
  version?: string;
  tools?: string[];
  status: string;
  last_seen?: string;
}

export interface MCPTool {
  id: string;
  name: string;
  server_id: string;
  description?: string;
  input_schema?: string;
}

export interface MCPCallResult {
  success: boolean;
  data?: Record<string, unknown>;
  error?: string;
}

export interface AgBOM {
  schema: string;
  agent: AgentRef;
  runtime: AgBOMRuntime;
  models?: AgBOMModel[];
  tools?: AgBOMTool[];
  mcp_servers?: AgBOMMCP[];
  memory?: AgBOMMemory[];
  skills?: AgBOMSkill[];
  permissions?: AgBOMPerm[];
  sessions: AgBOMSessions;
}

export interface AgBOMRuntime {
  os: string;
  arch: string;
  version: string;
}

export interface AgBOMModel {
  name: string;
  provider: string;
}

export interface AgBOMTool {
  name: string;
  server?: string;
}

export interface AgBOMMCP {
  name: string;
  url?: string;
}

export interface AgBOMMemory {
  type: string;
  path?: string;
}

export interface AgBOMSkill {
  name: string;
  source?: string;
}

export interface AgBOMPerm {
  resource: string;
  access: string;
}

export interface AgBOMSessions {
  count: number;
  last_seen: string;
}

export interface Decision {
  action: string;
  rule_id?: string;
  reason?: string;
}

export interface MetricValue {
  name: string;
  value: number;
  labels?: Record<string, string>;
  timestamp: string;
}

export interface Machine {
  id: string;
  name?: string;
  status: string;
  os?: string;
  arch?: string;
  enrolled_at?: string;
  last_seen_at?: string;
  tags?: Record<string, string>;
}
