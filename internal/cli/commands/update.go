package commands

func updateCmd() *Cmd {
	return &Cmd{
		Name:  "update",
		Short: "Install or update crux and cruxd",
		Run:   notImplCmd("update"),
	}
}
