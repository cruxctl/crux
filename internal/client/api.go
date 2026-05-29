package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/cruxctl/cruxd/pkg/cruxapi"
)

// --- Agents ---

func (c *Client) ListAgents(ctx context.Context) ([]cruxapi.Agent, error) {
	var out []cruxapi.Agent
	return out, c.do(ctx, http.MethodGet, "/v1/agents", nil, &out)
}

func (c *Client) GetAgent(ctx context.Context, name string) (cruxapi.Agent, error) {
	var out cruxapi.Agent
	return out, c.do(ctx, http.MethodGet, "/v1/agents/"+name, nil, &out)
}

func (c *Client) UpsertAgent(ctx context.Context, agent cruxapi.Agent) (cruxapi.Agent, error) {
	var out cruxapi.Agent
	return out, c.do(ctx, http.MethodPost, "/v1/agents", agent, &out)
}

func (c *Client) DeleteAgent(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/v1/agents/"+name, nil, nil)
}

func (c *Client) RefreshAgents(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/v1/agents/refresh", nil, nil)
}

// --- Discovery ---

func (c *Client) Discover(ctx context.Context) ([]cruxapi.DiscoveryResult, error) {
	var out []cruxapi.DiscoveryResult
	return out, c.do(ctx, http.MethodGet, "/v1/discover", nil, &out)
}

func (c *Client) InjectGateway(ctx context.Context, req cruxapi.GatewayInjectRequest) ([]cruxapi.GatewayInjectResult, error) {
	var out []cruxapi.GatewayInjectResult
	return out, c.do(ctx, http.MethodPost, "/v1/discover/inject-gateway", req, &out)
}

func (c *Client) UndoGatewayInject(ctx context.Context, req cruxapi.GatewayInjectRequest) ([]cruxapi.GatewayInjectResult, error) {
	var out []cruxapi.GatewayInjectResult
	return out, c.do(ctx, http.MethodPost, "/v1/discover/inject-gateway/undo", req, &out)
}

// --- Sessions ---

func (c *Client) ListSessions(ctx context.Context) ([]cruxapi.Session, error) {
	var out []cruxapi.Session
	return out, c.do(ctx, http.MethodGet, "/v1/sessions", nil, &out)
}

func (c *Client) GetSession(ctx context.Context, id string) (cruxapi.Session, error) {
	var out cruxapi.Session
	return out, c.do(ctx, http.MethodGet, "/v1/sessions/"+id, nil, &out)
}

func (c *Client) CreateSession(ctx context.Context, sess cruxapi.Session) (cruxapi.Session, error) {
	var out cruxapi.Session
	return out, c.do(ctx, http.MethodPost, "/v1/sessions", sess, &out)
}

func (c *Client) ContinueSession(ctx context.Context, id string, req cruxapi.ContinueSessionRequest) (cruxapi.Session, error) {
	var out cruxapi.Session
	return out, c.do(ctx, http.MethodPost, "/v1/sessions/"+id+"/continue", req, &out)
}

func (c *Client) StopSession(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/v1/sessions/"+id+"/stop", nil, nil)
}

func (c *Client) ReplaySession(ctx context.Context, id string) (cruxapi.Session, error) {
	var out cruxapi.Session
	return out, c.do(ctx, http.MethodPost, "/v1/sessions/"+id+"/replay", nil, &out)
}

// --- Policies ---

func (c *Client) ListPolicies(ctx context.Context) ([]cruxapi.PolicyProfile, error) {
	var out []cruxapi.PolicyProfile
	return out, c.do(ctx, http.MethodGet, "/v1/policies", nil, &out)
}

func (c *Client) GetPolicy(ctx context.Context, id string) (cruxapi.PolicyProfile, error) {
	var out cruxapi.PolicyProfile
	return out, c.do(ctx, http.MethodGet, "/v1/policies/"+id, nil, &out)
}

func (c *Client) CreatePolicy(ctx context.Context, policy cruxapi.PolicyProfile) (cruxapi.PolicyProfile, error) {
	var out cruxapi.PolicyProfile
	return out, c.do(ctx, http.MethodPost, "/v1/policies", policy, &out)
}

func (c *Client) DeletePolicy(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/policies/"+id, nil, nil)
}

func (c *Client) EvaluatePolicy(ctx context.Context, id string, input cruxapi.EvaluationInput) (cruxapi.Decision, error) {
	var out cruxapi.Decision
	return out, c.do(ctx, http.MethodPost, "/v1/policies/"+id+"/evaluate", input, &out)
}

// --- Approvals ---

func (c *Client) ListApprovals(ctx context.Context) ([]cruxapi.ApprovalRecord, error) {
	var out []cruxapi.ApprovalRecord
	return out, c.do(ctx, http.MethodGet, "/v1/approvals", nil, &out)
}

func (c *Client) GetApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	var out cruxapi.ApprovalRecord
	return out, c.do(ctx, http.MethodGet, "/v1/approvals/"+id, nil, &out)
}

func (c *Client) GrantApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	var out cruxapi.ApprovalRecord
	return out, c.do(ctx, http.MethodPost, "/v1/approvals/"+id+"/grant", nil, &out)
}

func (c *Client) DenyApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	var out cruxapi.ApprovalRecord
	return out, c.do(ctx, http.MethodPost, "/v1/approvals/"+id+"/deny", nil, &out)
}

// --- AOS Events ---

func (c *Client) ListAOSEvents(ctx context.Context, filter map[string]string) ([]cruxapi.AOSEvent, error) {
	path := "/v1/aos/events"
	if len(filter) > 0 {
		q := url.Values{}
		for k, v := range filter {
			q.Set(k, v)
		}
		path = path + "?" + q.Encode()
	}
	var out []cruxapi.AOSEvent
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

func (c *Client) GetAOSEvent(ctx context.Context, id string) (cruxapi.AOSEvent, error) {
	var out cruxapi.AOSEvent
	return out, c.do(ctx, http.MethodGet, "/v1/aos/events/"+id, nil, &out)
}

// --- Gateway ---

func (c *Client) GetGatewayStatus(ctx context.Context) (cruxapi.GatewayStatus, error) {
	var out cruxapi.GatewayStatus
	return out, c.do(ctx, http.MethodGet, "/v1/gateway/status", nil, &out)
}

func (c *Client) GetGatewayRoutes(ctx context.Context) ([]cruxapi.GatewayRoute, error) {
	var out []cruxapi.GatewayRoute
	return out, c.do(ctx, http.MethodGet, "/v1/gateway/routes", nil, &out)
}

// --- MCP ---

func (c *Client) ListMCPServers(ctx context.Context) ([]cruxapi.MCPServer, error) {
	var out []cruxapi.MCPServer
	return out, c.do(ctx, http.MethodGet, "/v1/mcp/servers", nil, &out)
}

func (c *Client) GetMCPServer(ctx context.Context, id string) (cruxapi.MCPServer, error) {
	var out cruxapi.MCPServer
	return out, c.do(ctx, http.MethodGet, "/v1/mcp/servers/"+id, nil, &out)
}

func (c *Client) ListMCPTools(ctx context.Context) ([]cruxapi.MCPTool, error) {
	var out []cruxapi.MCPTool
	return out, c.do(ctx, http.MethodGet, "/v1/mcp/tools", nil, &out)
}

func (c *Client) CallMCPTool(ctx context.Context, req cruxapi.MCPCallRequest) (cruxapi.MCPCallResult, error) {
	var out cruxapi.MCPCallResult
	return out, c.do(ctx, http.MethodPost, "/v1/mcp/calls", req, &out)
}

// --- Usage / Cost ---

func (c *Client) GetUsage(ctx context.Context, agent, project, since string) ([]cruxapi.UsageReport, error) {
	q := url.Values{}
	if agent != "" {
		q.Set("agent", agent)
	}
	if project != "" {
		q.Set("project", project)
	}
	if since != "" {
		q.Set("since", since)
	}
	path := "/v1/usage"
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	var out []cruxapi.UsageReport
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

func (c *Client) GetCosts(ctx context.Context, agent, project, since string) ([]cruxapi.CostReport, error) {
	q := url.Values{}
	if agent != "" {
		q.Set("agent", agent)
	}
	if project != "" {
		q.Set("project", project)
	}
	if since != "" {
		q.Set("since", since)
	}
	path := "/v1/costs"
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	var out []cruxapi.CostReport
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

func (c *Client) GetUsageLimits(ctx context.Context) (cruxapi.UsageLimits, error) {
	var out cruxapi.UsageLimits
	return out, c.do(ctx, http.MethodGet, "/v1/usage/limits", nil, &out)
}

func (c *Client) SetUsageLimits(ctx context.Context, limits cruxapi.UsageLimits) (cruxapi.UsageLimits, error) {
	var out cruxapi.UsageLimits
	return out, c.do(ctx, http.MethodPut, "/v1/usage/limits", limits, &out)
}

// --- Machines ---

func (c *Client) ListMachines(ctx context.Context) ([]cruxapi.Machine, error) {
	var out []cruxapi.Machine
	return out, c.do(ctx, http.MethodGet, "/v1/machines", nil, &out)
}

func (c *Client) PairMachine(ctx context.Context, req cruxapi.EnrollmentRequest) (cruxapi.EnrollmentResponse, error) {
	var out cruxapi.EnrollmentResponse
	return out, c.do(ctx, http.MethodPost, "/v1/machines/pair", req, &out)
}

// --- Metrics ---

func (c *Client) GetMetrics(ctx context.Context) ([]cruxapi.MetricValue, error) {
	var out []cruxapi.MetricValue
	return out, c.do(ctx, http.MethodGet, "/v1/metrics", nil, &out)
}

// --- AgBOM ---

func (c *Client) GenerateAgBOM(ctx context.Context, agentID, projectID, sessionID string) (cruxapi.AgBOM, error) {
	q := url.Values{}
	if agentID != "" {
		q.Set("agentId", agentID)
	}
	if projectID != "" {
		q.Set("projectId", projectID)
	}
	if sessionID != "" {
		q.Set("sessionId", sessionID)
	}
	path := "/v1/agbom/generate"
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	var out cruxapi.AgBOM
	return out, c.do(ctx, http.MethodPost, path, nil, &out)
}
