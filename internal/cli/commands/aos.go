package commands


func aosCmd() *Cmd {
	return &Cmd{
		Name:  "aos",
		Short: "Agent Observability Stack",
		Subcommands: []*Cmd{
			{Name: "events", Run: notImplCmd("aos events")},
			{Name: "export", Run: notImplCmd("aos export")},
			{Name: "traces", Run: notImplCmd("aos traces")},
		},
	}
}
