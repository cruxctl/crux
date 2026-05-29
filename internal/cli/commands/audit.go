package commands


func auditCmd() *Cmd {
	return &Cmd{
		Name:  "audit",
		Short: "View audit trail",
		Subcommands: []*Cmd{
			{Name: "log", Run: notImplCmd("audit log")},
			{Name: "export", Run: notImplCmd("audit export")},
		},
	}
}
