package commands


func gatewayCmd() *Cmd {
	return &Cmd{
		Name:  "gateway",
		Short: "Manage Crux Gateway",
		Subcommands: []*Cmd{
			{Name: "status", Run: notImplCmd("gateway status")},
			{Name: "routes", Run: notImplCmd("gateway routes")},
			{Name: "inject", Run: notImplCmd("gateway inject")},
			{Name: "undo", Run: notImplCmd("gateway undo")},
		},
	}
}
