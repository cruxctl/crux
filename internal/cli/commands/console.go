package commands


func consoleCmd() *Cmd {
	return &Cmd{
		Name:  "console",
		Short: "Open the web console",
		Subcommands: []*Cmd{
			{Name: "open", Run: notImplCmd("console open")},
			{Name: "status", Run: notImplCmd("console status")},
		},
	}
}
