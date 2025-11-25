package cliexecuter

import "os/exec"

type SystemCommandExecutor struct{}

func NewSystemCommandExecutor() *SystemCommandExecutor {
	return &SystemCommandExecutor{}
}

func (s *SystemCommandExecutor) Execute(cmd string, args ...string) ([]byte, error) {
	return exec.Command(cmd, args...).Output()
}
