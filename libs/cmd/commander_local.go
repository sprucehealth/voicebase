package cmd

import (
	"os/exec"
	"syscall"
)

type localCommander struct{}

var LocalCommander Commander = localCommander{}

func (lc localCommander) Close() error {
	return nil
}

func (lc localCommander) Command(cmd string, args ...string) (*Cmd, error) {
	return &Cmd{
		Path: cmd,
		Args: args,

		commander: lc,
		private:   exec.Command(cmd, args...),
	}, nil
}

func (lc localCommander) cmd(c *Cmd) *exec.Cmd {
	cmd := c.private.(*exec.Cmd)
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	return cmd
}

func (lc localCommander) run(c *Cmd) error {
	return mapLocalExitError(lc.cmd(c).Run())
}

func (lc localCommander) start(c *Cmd) error {
	return lc.cmd(c).Start()
}

func (lc localCommander) wait(c *Cmd) error {
	return mapLocalExitError(lc.cmd(c).Wait())
}

func (lc localCommander) close(c *Cmd) error {
	return nil
}

func mapLocalExitError(err error) error {
	switch e := err.(type) {
	default:
		return e
	case *exec.ExitError:
		if st, ok := e.Sys().(syscall.WaitStatus); ok {
			return &ExitError{
				Status:  st.ExitStatus(),
				Signal:  st.Signal().String(),
				Message: e.String(),
			}
		}
		return e
	}
}
