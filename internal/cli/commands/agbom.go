package commands


func agbomCmd() *Cmd {
	return &Cmd{
		Name:  "agbom",
		Short: "Agent Bill of Materials",
		Subcommands: []*Cmd{
			{Name: "generate", Run: notImplCmd("agbom generate")},
			{Name: "show", Run: notImplCmd("agbom show")},
		},
	}
}
