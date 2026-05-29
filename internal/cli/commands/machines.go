package commands


func machinesCmd() *Cmd {
	return &Cmd{
		Name:  "machines",
		Short: "Manage machine inventory",
		Subcommands: []*Cmd{
			{Name: "list", Run: notImplCmd("machines list")},
			{Name: "pair", Run: notImplCmd("machines pair")},
			{Name: "unpair", Run: notImplCmd("machines unpair")},
		},
	}
}
