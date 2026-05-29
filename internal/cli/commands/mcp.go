package commands


func mcpCmd() *Cmd {
	return &Cmd{
		Name:  "mcp",
		Short: "Manage MCP servers and tools",
		Subcommands: []*Cmd{
			{Name: "servers", Run: notImplCmd("mcp servers")},
			{Name: "tools", Run: notImplCmd("mcp tools")},
			{Name: "calls", Run: notImplCmd("mcp calls")},
			{Name: "proxy", Run: notImplCmd("mcp proxy")},
		},
	}
}
