package cmd

import (
	"net"
	"os"
	"strings"

	"code.google.com/p/go.crypto/ssh"
)

type sshCommander struct {
	conn        *ssh.ClientConn
	bastionConn *ssh.ClientConn
	proxyConn   net.Conn
}

func parseSSHAddr(addr, defUser string) (user, host string) {
	user = defUser
	host = addr

	idx := strings.IndexByte(addr, '@')
	if idx > 0 {
		user = addr[:idx]
		host = addr[idx+1:]
	}
	return
}

func NewSSHCommander(addr, bastionAddr string) (Commander, error) {
	defUser := os.Getenv("USER")

	cm := &sshCommander{}

	var auths []ssh.ClientAuth
	if agent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.ClientAuthAgent(ssh.NewAgentClient(agent)))
	}

	user, host := parseSSHAddr(addr, defUser)
	config := &ssh.ClientConfig{
		User: user,
		Auth: auths,
	}

	if bastionAddr != "" {
		bastionUser, bastionHost := parseSSHAddr(bastionAddr, defUser)

		bastionConfig := &ssh.ClientConfig{
			User: bastionUser,
			Auth: auths,
		}
		conn, err := ssh.Dial("tcp", bastionHost, bastionConfig)
		if err != nil {
			return nil, err
		}

		cliConn, err := conn.Dial("tcp4", host)
		if err != nil {
			conn.Close()
			return nil, err
		}

		c2, err := ssh.Client(cliConn, config)
		if err != nil {
			cliConn.Close()
			conn.Close()
			return nil, err
		}

		cm.bastionConn = conn
		cm.proxyConn = cliConn
		cm.conn = c2
	} else {
		conn, err := ssh.Dial("tcp", host, config)
		if err != nil {
			return nil, err
		}
		cm.conn = conn
	}

	return cm, nil
}

func (sc *sshCommander) Close() error {
	return sc.conn.Close()
}

func (sc *sshCommander) Command(cmd string, args ...string) (*Cmd, error) {
	sess, err := sc.conn.NewSession()
	if err != nil {
		return nil, err
	}
	// modes := ssh.TerminalModes{
	// 	ssh.ECHO:          1,     // disable echoing
	// 	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	// 	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	// }
	// sess.RequestPty("xterm", 24, 80, modes)
	c := &Cmd{
		Path: cmd,
		Args: args,

		commander: sc,
		private:   sess,
	}
	return c, nil
}

func (sc *sshCommander) sessionFromCommand(c *Cmd) (*ssh.Session, string, error) {
	sess := c.private.(*ssh.Session)
	sess.Stdin = c.Stdin
	sess.Stdout = c.Stdout
	sess.Stderr = c.Stderr
	cmdString := c.Path
	if len(c.Args) > 0 {
		for i, arg := range c.Args {
			c.Args[i] = `"` + strings.Replace(arg, `"`, `\"`, -1) + `"`
		}
		cmdString += " " + strings.Join(c.Args, " ")
	}
	return sess, cmdString, nil
}

func (sc *sshCommander) run(c *Cmd) error {
	sess, cmdString, err := sc.sessionFromCommand(c)
	if err != nil {
		return err
	}
	return mapSSHExitError(sess.Run(cmdString))
}

func (sc *sshCommander) start(c *Cmd) error {
	sess, cmdString, err := sc.sessionFromCommand(c)
	if err != nil {
		return err
	}
	return sess.Start(cmdString)
}

func (sc *sshCommander) wait(c *Cmd) error {
	return mapSSHExitError(c.private.(*ssh.Session).Wait())
}

func (sc *sshCommander) close(c *Cmd) error {
	sess := c.private.(*ssh.Session)
	return sess.Close()
}

func mapSSHExitError(err error) error {
	switch e := err.(type) {
	default:
		return e
	case *ssh.ExitError:
		return &ExitError{
			Status:  e.ExitStatus(),
			Signal:  e.Signal(),
			Message: e.String(),
		}
	}
}
