// Package sshrunner opens an interactive SSH shell to a Cisco device, sends commands,
// waits for the prompt, and recovers from timeouts via Ctrl+C before returning clean output.
package sshrunner

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// promptRe matches a Cisco IOS-XE/NX-OS prompt at the very end of the
// accumulated output, e.g. "switch1#", "router1>", "switch1(config)#".
// No (?m) flag: $ matches only end-of-string, so a > or # mid-output
// (e.g. the "<string>" literal in NX-OS route legend lines) never fires early.
var promptRe = regexp.MustCompile(`(?:^|[\r\n])[\w\-\.]+(?:\([^)]*\))?[>#][ \t]*$`)

// ErrSessionDead is returned by Run when a previous timeout left the session
// in an unrecoverable state. The caller should stop sending more commands.
var ErrSessionDead = errors.New("session dead after unrecoverable timeout")

type readResult struct {
	data []byte
	err  error
}

type Session struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	chunks  chan readResult
	timeout time.Duration
	healthy bool
}

func Open(host string, port int, username, password string, timeout time.Duration) (*Session, error) {
	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("new session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("vt100", 200, 512, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}

	// Single reader goroutine for the session lifetime. Buffered so it never
	// blocks between readUntilPromptFor calls.
	chunks := make(chan readResult, 32)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				out := make([]byte, n)
				copy(out, buf[:n])
				chunks <- readResult{data: out}
			}
			if err != nil {
				chunks <- readResult{err: err}
				return
			}
		}
	}()

	s := &Session{
		client:  client,
		session: sshSession,
		stdin:   stdin,
		chunks:  chunks,
		timeout: timeout,
		healthy: true,
	}

	if _, err := s.readUntilPromptFor(s.timeout); err != nil {
		s.Close()
		return nil, fmt.Errorf("wait for initial prompt: %w", err)
	}

	if _, err := s.Run("terminal length 0"); err != nil {
		s.Close()
		return nil, fmt.Errorf("disable paging: %w", err)
	}

	return s, nil
}

func (s *Session) Run(command string) (string, error) {
	if !s.healthy {
		return "", ErrSessionDead
	}
	if _, err := s.stdin.Write([]byte(command + "\n")); err != nil {
		s.healthy = false
		return "", fmt.Errorf("write command: %w", err)
	}
	out, err := s.readUntilPromptFor(s.timeout)
	if err != nil {
		// Send Ctrl+C to abort the in-flight command on the device and wait
		// briefly for the prompt. If recovery fails the session is dead.
		s.healthy = s.tryRecover()
		return "", err
	}
	return stripEcho(out, command), nil
}

func (s *Session) Close() {
	s.session.Close()
	s.client.Close()
}

// tryRecover sends Ctrl+C to interrupt whatever the device is doing and waits
// up to 5 seconds for the prompt to reappear returns true on success.
func (s *Session) tryRecover() bool {
	_, _ = s.stdin.Write([]byte("\x03\n"))
	_, err := s.readUntilPromptFor(5 * time.Second)
	return err == nil
}

// readUntilPromptFor reads from the session-lifetime chunks channel until the
// device prompt reappears or the idle timeout elapses with no new data.
func (s *Session) readUntilPromptFor(timeout time.Duration) (string, error) {
	var sb strings.Builder
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case c := <-s.chunks:
			if c.err != nil {
				if sb.Len() > 0 {
					return sb.String(), nil
				}
				return "", fmt.Errorf("read output: %w", c.err)
			}
			sb.Write(c.data)
			if promptRe.MatchString(sb.String()) {
				return sb.String(), nil
			}
			// Reset idle timer whenever new data arrives.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(timeout)
		case <-timer.C:
			return "", fmt.Errorf("timed out waiting for prompt after %s", timeout)
		}
	}
}

// stripEcho removes the echoed command and trailing prompt line from
// the raw output captured between writing a command and seeing the prompt again.
func stripEcho(raw, command string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == strings.TrimSpace(command) {
		lines = lines[1:]
	}
	if len(lines) > 0 && promptRe.MatchString(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}
