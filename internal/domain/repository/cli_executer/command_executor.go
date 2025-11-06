package cli_executer

type CommandExecutor interface {
	Execute(cmd string, args ...string) ([]byte, error)
}
