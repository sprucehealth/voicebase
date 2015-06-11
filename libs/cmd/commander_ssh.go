package cmd

import (
	"bufio"
	"log"
	"net"
	"os"
	"path"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/crypto/ssh/agent"
)

type sshCommander struct {
	conn        *ssh.Client
	bastionConn *ssh.Client
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

	var auths []ssh.AuthMethod
	if authSock := os.Getenv("SSH_AUTH_SOCK"); authSock != "" {
		if agentConn, err := net.Dial("unix", authSock); err == nil {
			auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
		} else {
			log.Printf("Failed to connect to SSH agent: %s", err.Error())
		}
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

		cc, chans, reqs, err := ssh.NewClientConn(cliConn, host, config)
		if err != nil {
			cliConn.Close()
			conn.Close()
			return nil, err
		}
		c2 := ssh.NewClient(cc, chans, reqs)

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

type sshHostConfig struct {
	ProxyCommand string
	Bastion      string
}

type sshConfig struct {
	Default *sshHostConfig
	Hosts   map[string]*sshHostConfig
}

func (cnf *sshConfig) ConfigForHost(name string) *sshHostConfig {
	c := *cnf.Default

	return &c
}

// does not support unicode and only supports one asterisk
// func matchAsterisk(toMatch, pattern string) bool {
// 	for i, c := range pattern {
// 		if c == '*' {
// 			for j := 0; j < len(pattern)-i; j++ {
// 				if toMatch[len(toMatch)-j-1] != pattern[len(pattern)-j-1] {
// 					return false
// 				}
// 			}
// 			return true
// 		}
// 		if toMatch[i] != c {
// 			return false
// 		}
// 	}
// 	return true
// }

// parseSSHConfig is a super janky parser for ~/.ssh/config
func parseSSHConfig() *sshConfig {
	configPath := path.Join(os.Getenv("HOME"), ".ssh", "config")
	fi, err := os.Open(configPath)
	if err != nil {
		return &sshConfig{}
	}
	defer fi.Close()
	scan := bufio.NewScanner(fi)
	curHost := &sshHostConfig{}
	config := &sshConfig{
		Default: curHost,
		Hosts:   map[string]*sshHostConfig{},
	}
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		idx := strings.IndexByte(line, ' ')
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+1:])
		switch strings.ToLower(line[:idx]) {
		case "proxycommand":
			curHost.ProxyCommand = rest
			if strings.HasPrefix(rest, "ssh ") {
				rest = strings.TrimSpace(rest[4:])
				if idx := strings.IndexByte(rest, ' '); idx > 0 {
					curHost.Bastion = rest[:idx]
				}
			}
		case "host":
			curHost = config.Hosts[rest]
			if curHost == nil {
				curHost = &sshHostConfig{}
				config.Hosts[rest] = curHost
			}
		}
	}
	return config
}
