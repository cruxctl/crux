package commands


func runCmd() *Cmd {
	return &Cmd{
		Name:  "run",
		Short: "Run a task through an agent",
		Run:   notImplCmd("run"),
	}
}
