package commands


func policyCmd() *Cmd {
	return &Cmd{
		Name:  "policy",
		Short: "Manage policy profiles",
		Subcommands: []*Cmd{
			{Name: "list", Run: notImplCmd("policy list")},
			{Name: "get", Run: notImplCmd("policy get")},
			{Name: "apply", Run: notImplCmd("policy apply")},
			{Name: "delete", Run: notImplCmd("policy delete")},
			{Name: "evaluate", Run: notImplCmd("policy evaluate")},
		},
	}
}
