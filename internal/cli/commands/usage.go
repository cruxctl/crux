package commands


func usageCmd() *Cmd {
	return &Cmd{
		Name:  "usage",
		Short: "View usage and costs",
		Subcommands: []*Cmd{
			{Name: "show", Run: notImplCmd("usage show")},
			{Name: "limits", Run: notImplCmd("usage limits")},
			{Name: "costs", Run: notImplCmd("usage costs")},
		},
	}
}
