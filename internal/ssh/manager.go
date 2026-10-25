package ssh

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProfileManager manages named SSH server targets
type ProfileManager struct {
	filePath string
	mu       sync.RWMutex
	store    HostStore
}

// NewProfileManager loads host profiles from disk
func NewProfileManager(customPath string) (*ProfileManager, error) {
	path := customPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(home, ".shellix")
		_ = os.MkdirAll(dir, 0755)
		path = filepath.Join(dir, "ssh_hosts.json")
	}

	pm := &ProfileManager{
		filePath: path,
		store: HostStore{
			Hosts: make(map[string]HostProfile),
		},
	}

	_ = pm.load()
	return pm, nil
}

func (pm *ProfileManager) load() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	data, err := os.ReadFile(pm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var hs HostStore
	if err := json.Unmarshal(data, &hs); err != nil {
		return err
	}
	if hs.Hosts == nil {
		hs.Hosts = make(map[string]HostProfile)
	}
	pm.store = hs
	return nil
}

func (pm *ProfileManager) save() error {
	data, err := json.MarshalIndent(pm.store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pm.filePath, data, 0600) // Secure permissions
}

// AddHost saves or updates an SSH host profile
func (pm *ProfileManager) AddHost(p HostProfile) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if p.Name == "" || p.Host == "" || p.User == "" {
		return fmt.Errorf("host profile requires 'name', 'host', and 'user'")
	}
	if p.Port == 0 {
		p.Port = 22
	}
	p.CreatedAt = time.Now()

	pm.store.Hosts[p.Name] = p
	return pm.save()
}

// GetHost retrieves a host by alias
func (pm *ProfileManager) GetHost(name string) (HostProfile, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	h, ok := pm.store.Hosts[name]
	return h, ok
}

// ListHosts returns all configured server targets
func (pm *ProfileManager) ListHosts() []HostProfile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	res := make([]HostProfile, 0, len(pm.store.Hosts))
	for _, h := range pm.store.Hosts {
		res = append(res, h)
	}
	return res
}

// DeleteHost removes a target
func (pm *ProfileManager) DeleteHost(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.store.Hosts, name)
	return pm.save()
}
