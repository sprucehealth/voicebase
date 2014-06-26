package cmd

import (
	"fmt"
	"io"
)

type ExitError struct {
	Status  int
	Signal  string
	Message string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exited with %d due to signal %s: %s", e.Status, e.Signal, e.Message)
}

type Cmd struct {
	Path   string
	Args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	commander privateCommander
	private   interface{}
}

type privateCommander interface {
	start(*Cmd) error
	run(*Cmd) error
	wait(*Cmd) error
	close(*Cmd) error
}

func (c *Cmd) Start() error {
	return c.commander.start(c)
}

func (c *Cmd) Run() error {
	return c.commander.run(c)
}

func (c *Cmd) Wait() error {
	return c.commander.wait(c)
}

func (c *Cmd) Close() error {
	return c.commander.close(c)
}

type Commander interface {
	Command(name string, args ...string) (*Cmd, error)
	Close() error
}
