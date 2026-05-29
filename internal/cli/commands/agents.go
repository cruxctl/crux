package commands


func agentsCmd() *Cmd {
	return &Cmd{
		Name:  "agents",
		Short: "Manage registered agents",
		Subcommands: []*Cmd{
			{Name: "list", Run: notImplCmd("agents list")},
			{Name: "get", Run: notImplCmd("agents get")},
			{Name: "register", Run: notImplCmd("agents register")},
			{Name: "unregister", Run: notImplCmd("agents unregister")},
		},
	}
}
