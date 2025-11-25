package cliexecuter

type CommandExecutor interface {
	Execute(cmd string, args ...string) ([]byte, error)
}
