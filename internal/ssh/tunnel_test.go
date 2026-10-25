package ssh

import (
	"testing"
)

func TestParseTunnelSpec(t *testing.T) {
	cases := []struct {
		input       string
		expectErr   bool
		expectedL   int
		expectedR   int
		expectedRH  string
	}{
		{"8080", false, 8080, 8080, "localhost"},
		{"3000:80", false, 3000, 80, "localhost"},
		{"5432:db.internal:5432", false, 5432, 5432, "db.internal"},
		{"8080::80", false, 8080, 80, "localhost"},
		{"0", true, 0, 0, ""},
		{"70000", true, 0, 0, ""},
		{"invalid", true, 0, 0, ""},
		{"8080:invalid", true, 0, 0, ""},
		{"8080:host:invalid", true, 0, 0, ""},
	}

	for _, c := range cases {
		cfg, err := ParseTunnelSpec(c.input)
		if c.expectErr {
			if err == nil {
				t.Errorf("input %q: expected error, got nil", c.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("input %q: unexpected error: %v", c.input, err)
			continue
		}

		if cfg.LocalPort != c.expectedL {
			t.Errorf("input %q: expected local port %d, got %d", c.input, c.expectedL, cfg.LocalPort)
		}
		if cfg.RemotePort != c.expectedR {
			t.Errorf("input %q: expected remote port %d, got %d", c.input, c.expectedR, cfg.RemotePort)
		}
		if cfg.RemoteHost != c.expectedRH {
			t.Errorf("input %q: expected remote host %q, got %q", c.input, c.expectedRH, cfg.RemoteHost)
		}
	}
}
