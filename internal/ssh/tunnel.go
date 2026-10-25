package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// TunnelConfig holds local port forwarding parameters
type TunnelConfig struct {
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int
}

// ParseTunnelSpec parses forwarding strings like:
// - "5432:localhost:5432" (local:remote_host:remote_port)
// - "8080:80"             (local:remote_port on localhost)
// - "9090"                (same port on local and remote localhost)
func ParseTunnelSpec(spec string) (*TunnelConfig, error) {
	parts := strings.Split(strings.TrimSpace(spec), ":")
	switch len(parts) {
	case 1:
		port, err := strconv.Atoi(parts[0])
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("invalid port '%s'", parts[0])
		}
		return &TunnelConfig{
			LocalHost:  "127.0.0.1",
			LocalPort:  port,
			RemoteHost: "localhost",
			RemotePort: port,
		}, nil

	case 2:
		lPort, err := strconv.Atoi(parts[0])
		if err != nil || lPort <= 0 || lPort > 65535 {
			return nil, fmt.Errorf("invalid local port '%s'", parts[0])
		}
		rPort, err := strconv.Atoi(parts[1])
		if err != nil || rPort <= 0 || rPort > 65535 {
			return nil, fmt.Errorf("invalid remote port '%s'", parts[1])
		}
		return &TunnelConfig{
			LocalHost:  "127.0.0.1",
			LocalPort:  lPort,
			RemoteHost: "localhost",
			RemotePort: rPort,
		}, nil

	case 3:
		lPort, err := strconv.Atoi(parts[0])
		if err != nil || lPort <= 0 || lPort > 65535 {
			return nil, fmt.Errorf("invalid local port '%s'", parts[0])
		}
		rHost := parts[1]
		if rHost == "" {
			rHost = "localhost"
		}
		rPort, err := strconv.Atoi(parts[2])
		if err != nil || rPort <= 0 || rPort > 65535 {
			return nil, fmt.Errorf("invalid remote port '%s'", parts[2])
		}
		return &TunnelConfig{
			LocalHost:  "127.0.0.1",
			LocalPort:  lPort,
			RemoteHost: rHost,
			RemotePort: rPort,
		}, nil

	default:
		return nil, fmt.Errorf("invalid tunnel format '%s'. Use [localPort:]remoteHost:remotePort", spec)
	}
}

// StartTunnel opens a local TCP listener and forwards incoming connections over SSH
func StartTunnel(ctx context.Context, p HostProfile, cfg *TunnelConfig) error {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))

	client, err := Dial(p)
	if err != nil {
		return fmt.Errorf("failed to connect to %s (%s): %w", p.Name, p.Host, err)
	}
	defer func() { _ = client.Close() }()

	localAddr := fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to open local listener on %s: %w", localAddr, err)
	}
	defer func() { _ = listener.Close() }()

	remoteTarget := fmt.Sprintf("%s:%d", cfg.RemoteHost, cfg.RemotePort)

	fmt.Println(titleStyle.Render("🚇 Shellix SSH Secure Port Forwarding Tunnel"))
	fmt.Printf("   %s Target Server:  %s (%s@%s:%d)\n", successStyle.Render("✔"), p.Name, p.User, p.Host, p.Port)
	fmt.Printf("   %s Local Endpoint: http://%s\n", successStyle.Render("✔"), localAddr)
	fmt.Printf("   %s Forwarding To:  %s\n", successStyle.Render("✔"), remoteTarget)
	fmt.Println(metaStyle.Render("   Press Ctrl+C to close the secure tunnel."))
	fmt.Println()

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	var wg sync.WaitGroup
	for {
		localConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				fmt.Println(warnStyle.Render("\n🛑 Tunnel stopped cleanly."))
				wg.Wait()
				return nil
			default:
				return fmt.Errorf("listener accept error: %w", err)
			}
		}

		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			defer func() { _ = conn.Close() }()

			remoteConn, err := client.Dial("tcp", remoteTarget)
			if err != nil {
				fmt.Printf("   %s Failed to dial remote %s: %v\n", warnStyle.Render("▲"), remoteTarget, err)
				return
			}
			defer func() { _ = remoteConn.Close() }()

			now := time.Now().Format("15:04:05")
			fmt.Printf("   %s [%s] Forwarding connection from %s -> %s\n",
				metaStyle.Render("⇄"), now, conn.RemoteAddr(), remoteTarget)

			pipe(conn, remoteConn)
		}(localConn)
	}
}

// pipe bidirectionally forwards data between local and remote connections
func pipe(local, remote net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(remote, local)
		if tc, ok := remote.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(local, remote)
		if tc, ok := local.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	wg.Wait()
}
