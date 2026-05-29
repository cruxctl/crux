package commands


func daemonCmd() *Cmd {
	return &Cmd{
		Name:  "daemon",
		Short: "Manage the cruxd daemon",
		Subcommands: []*Cmd{
			{Name: "start", Run: notImplCmd("daemon start")},
			{Name: "stop", Run: notImplCmd("daemon stop")},
			{Name: "restart", Run: notImplCmd("daemon restart")},
			{Name: "status", Run: notImplCmd("daemon status")},
			{Name: "logs", Run: notImplCmd("daemon logs")},
		},
	}
}
