package commands


func agentCmd() *Cmd {
	return &Cmd{
		Name:  "agent",
		Short: "Operate a single agent",
		Subcommands: []*Cmd{
			{Name: "describe", Run: notImplCmd("agent describe")},
			{Name: "usage", Run: notImplCmd("agent usage")},
			{Name: "exec", Run: notImplCmd("agent exec")},
			{Name: "conversations", Run: notImplCmd("agent conversations")},
		},
	}
}
