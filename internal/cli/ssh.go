package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"shellix/internal/session"
	"shellix/internal/ssh"

	"github.com/spf13/cobra"
)

// NewSSHCmd handles SSH host management and multi-exec
func NewSSHCmd(app *AppContext) *cobra.Command {
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Multi-server SSH connection manager and parallel command executor",
	}

	var sshHost, sshUser, sshKey, sshPass string
	var sshPort int

	sshAddCmd := &cobra.Command{
		Use:   "add <alias>",
		Short: "Save a remote Linux/Unix server target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := ssh.HostProfile{
				Name:     args[0],
				Host:     sshHost,
				Port:     sshPort,
				User:     sshUser,
				KeyPath:  sshKey,
				Password: sshPass,
			}
			if err := app.SSHMgr.AddHost(p); err != nil {
				return err
			}
			fmt.Printf("✔ Saved SSH target '%s' (%s@%s:%d)\n", p.Name, p.User, p.Host, p.Port)
			return nil
		},
	}
	sshAddCmd.Flags().StringVar(&sshHost, "host", "", "Remote server hostname or IP")
	sshAddCmd.Flags().StringVar(&sshUser, "user", "", "Remote SSH user")
	sshAddCmd.Flags().IntVar(&sshPort, "port", 22, "SSH port")
	sshAddCmd.Flags().StringVar(&sshKey, "key", "", "Path to private key (defaults to ~/.ssh/id_rsa or id_ed25519)")
	sshAddCmd.Flags().StringVar(&sshPass, "pass", "", "Password (optional)")
	_ = sshAddCmd.MarkFlagRequired("host")
	_ = sshAddCmd.MarkFlagRequired("user")

	sshListCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured SSH server profiles",
		Run: func(cmd *cobra.Command, args []string) {
			hosts := app.SSHMgr.ListHosts()
			if len(hosts) == 0 {
				fmt.Println("No SSH hosts configured. Use 'sx ssh add <name> --host <ip> --user <user>' to add one.")
				return
			}
			fmt.Println("Configured SSH Server Targets:")
			for _, h := range hosts {
				fmt.Printf("  • %-16s %s@%s:%d\n", h.Name, h.User, h.Host, h.Port)
			}
		},
	}

	sshConnectCmd := &cobra.Command{
		Use:   "connect <alias>",
		Short: "Connect to a remote server with an interactive PTY session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hp, ok := app.SSHMgr.GetHost(args[0])
			if !ok {
				return fmt.Errorf("host profile '%s' not found", args[0])
			}
			s := app.SessionMgr.RegisterSession(hp.Name, session.SessionSSH, hp.Host)
			defer func() { _ = app.SessionMgr.CloseSession(s.ID) }()
			return ssh.ConnectInteractive(hp)
		},
	}

	var targetHostsStr string
	sshExecCmd := &cobra.Command{
		Use:   "exec \"<command>\"",
		Short: "Execute a command concurrently across multiple Linux servers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetHostsStr == "" {
				return fmt.Errorf("please provide targets via --hosts <h1,h2,...>")
			}
			targets := strings.Split(targetHostsStr, ",")
			var profiles []ssh.HostProfile
			for _, t := range targets {
				name := strings.TrimSpace(t)
				hp, ok := app.SSHMgr.GetHost(name)
				if !ok {
					return fmt.Errorf("configured host '%s' not found", name)
				}
				profiles = append(profiles, hp)
			}

			_ = ssh.MultiExec(profiles, args[0])
			return nil
		},
	}
	sshExecCmd.Flags().StringVar(&targetHostsStr, "hosts", "", "Comma-separated list of host aliases")
	_ = sshExecCmd.MarkFlagRequired("hosts")

	sshTunnelCmd := &cobra.Command{
		Use:   "tunnel <alias> [localPort:]remoteHost:remotePort",
		Short: "Forward a local port to a remote server target over SSH",
		Long: `Open an encrypted local port-forwarding tunnel to a remote Linux/Unix host.

Examples:
  sx ssh tunnel prod-db 5432:localhost:5432   # Forward local 5432 to remote 5432
  sx ssh tunnel prod-web 8080:80             # Forward local 8080 to remote 80
  sx ssh tunnel kafka-node 9092              # Forward local 9092 to remote 9092`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			hp, ok := app.SSHMgr.GetHost(args[0])
			if !ok {
				return fmt.Errorf("configured host '%s' not found", args[0])
			}
			cfg, err := ssh.ParseTunnelSpec(args[1])
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return ssh.StartTunnel(ctx, hp, cfg)
		},
	}

	sshCmd.AddCommand(sshAddCmd, sshListCmd, sshConnectCmd, sshExecCmd, sshTunnelCmd)
	return sshCmd
}
