package ssh

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var serverColors = []string{
	"#7AA2F7", "#04B575", "#FF79C6", "#E0AF68",
	"#BB9AF7", "#7DCFFF", "#F7768E", "#9ECE6A",
}

// MultiExec runs a command concurrently across multiple hosts
func MultiExec(hosts []HostProfile, command string) []RemoteExecResult {
	results := make([]RemoteExecResult, len(hosts))
	var wg sync.WaitGroup

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	fmt.Printf("%s Dispatching '%s' to %d server(s)...\n\n",
		titleStyle.Render("⚡ [SSH MULTI-EXEC]"), command, len(hosts))

	for i := range hosts {
		wg.Add(1)
		colorHex := serverColors[i%len(serverColors)]

		go func(idx int, hp HostProfile, color string) {
			defer wg.Done()
			start := time.Now()

			tagStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
			prefix := tagStyle.Render(fmt.Sprintf("[%s]", hp.Name))

			client, err := Dial(hp)
			if err != nil {
				fmt.Printf("%s ✖ Connection failed: %v\n", prefix, err)
				results[idx] = RemoteExecResult{
					HostName: hp.Name,
					Duration: time.Since(start),
					Success:  false,
					Error:    err,
				}
				return
			}
			defer func() { _ = client.Close() }()

			output, err := ExecuteCommand(client, command)
			duration := time.Since(start)

			scanner := bufio.NewScanner(strings.NewReader(output))
			for scanner.Scan() {
				fmt.Printf("%-24s %s\n", prefix, scanner.Text())
			}

			results[idx] = RemoteExecResult{
				HostName: hp.Name,
				Output:   output,
				Duration: duration,
				Success:  err == nil,
				Error:    err,
			}
		}(i, hosts[i], colorHex)
	}

	wg.Wait()
	return results
}
