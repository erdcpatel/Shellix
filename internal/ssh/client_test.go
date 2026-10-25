package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestHostKeyNormalization(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1:22", "192.168.1.1"},
		{"example.com:22", "example.com"},
		{"example.com:2222", "[example.com]:2222"},
		{"example.com", "example.com"},
	}

	for _, c := range cases {
		normalized := knownhosts.Normalize(c.input)
		if normalized != c.expected {
			t.Errorf("Normalize(%q): expected %q, got %q", c.input, c.expected, normalized)
		}
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	pubKey, err := gossh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to create public key: %v", err)
	}

	line := knownhosts.Line([]string{knownhosts.Normalize("myserver.internal:2222")}, pubKey)
	if !strings.HasPrefix(line, "[myserver.internal]:2222 ") {
		t.Errorf("expected line to start with normalized host '[myserver.internal]:2222 ', got %s", line)
	}

	line22 := knownhosts.Line([]string{knownhosts.Normalize("myserver.internal:22")}, pubKey)
	if !strings.HasPrefix(line22, "myserver.internal ") {
		t.Errorf("expected line to strip standard port 22 'myserver.internal ', got %s", line22)
	}
}
