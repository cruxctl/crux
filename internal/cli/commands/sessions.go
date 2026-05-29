package commands


func sessionsCmd() *Cmd {
	return &Cmd{
		Name:  "sessions",
		Short: "Manage agent sessions",
		Subcommands: []*Cmd{
			{Name: "list", Run: notImplCmd("sessions list")},
			{Name: "logs", Run: notImplCmd("sessions logs")},
			{Name: "replay", Run: notImplCmd("sessions replay")},
			{Name: "summarize", Run: notImplCmd("sessions summarize")},
			{Name: "continue", Run: notImplCmd("sessions continue")},
			{Name: "stop", Run: notImplCmd("sessions stop")},
		},
	}
}
