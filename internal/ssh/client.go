package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// getHostKeyCallback initializes secure known_hosts verification (Trust On First Use + Strict MITM protection)
func getHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate home directory for SSH: %w", err)
	}
	knownHostsFile := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHostsFile); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(knownHostsFile), 0700)
		f, createErr := os.OpenFile(knownHostsFile, os.O_CREATE|os.O_WRONLY, 0600)
		if createErr == nil {
			_ = f.Close()
		}
	}

	khCallback, err := knownhosts.New(knownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse known_hosts (%s): %w", knownHostsFile, err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := khCallback(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			if len(keyErr.Want) > 0 {
				// Host key mismatch: MITM detected!
				return fmt.Errorf("CRITICAL SECURITY ERROR: Remote host identification for '%s' has changed (possible MITM attack)! Verification failed: %w", hostname, err)
			}
			// Host key not found: Trust On First Use (TOFU)
			f, appendErr := os.OpenFile(knownHostsFile, os.O_APPEND|os.O_WRONLY, 0600)
			if appendErr == nil {
				defer f.Close()
				line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
				_, _ = f.WriteString(line + "\n")
				return nil
			}
		}
		return err
	}, nil
}

// Dial establishes an SSH connection to the target host
func Dial(p HostProfile) (*ssh.Client, error) {
	authMethods := []ssh.AuthMethod{}

	// 1. Password auth if provided
	if p.Password != "" {
		authMethods = append(authMethods, ssh.Password(p.Password))
	}

	// 2. Private key auth
	keyPaths := []string{}
	if p.KeyPath != "" {
		keyPaths = append(keyPaths, expandHome(p.KeyPath))
	} else {
		home, _ := os.UserHomeDir()
		keyPaths = append(keyPaths,
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
		)
	}

	for _, kp := range keyPaths {
		if keyData, err := os.ReadFile(kp); err == nil {
			if signer, err := ssh.ParsePrivateKey(keyData); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				break
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no usable SSH authentication method found for %s", p.Name)
	}

	hkCallback, err := getHostKeyCallback()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            p.User,
		Auth:            authMethods,
		HostKeyCallback: hkCallback,
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", p.Host, p.Port)
	return ssh.Dial("tcp", addr, config)
}

// ConnectInteractive launches an interactive PTY session on the remote server
func ConnectInteractive(p HostProfile) error {
	client, err := Dial(p)
	if err != nil {
		return fmt.Errorf("failed to connect to %s (%s): %w", p.Name, p.Host, err)
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Configure terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	width, height := 80, 24
	if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		width, height = w, h
	}

	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Raw mode locally
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()
	}

	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start remote shell: %w", err)
	}

	return session.Wait()
}

// ExecuteCommand runs a non-interactive command and returns its combined output
func ExecuteCommand(client *ssh.Client, cmdStr string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer func() { _ = session.Close() }()

	out, err := session.CombinedOutput(cmdStr)
	return string(out), err
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// CheckHostReachability checks if port is open before full handshake
func CheckHostReachability(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
