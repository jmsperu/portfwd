package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Name        string    `json:"name"`
	Protocol    string    `json:"protocol"`
	ListenPort  int       `json:"listen_port"`
	RemoteHost  string    `json:"remote_host"`
	RemotePort  int       `json:"remote_port"`
	Connections int64     `json:"connections"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	LastActive  time.Time `json:"last_active"`
}

var mu sync.Mutex

func statsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".portfwd-stats.json")
}

func Load() ([]Entry, error) {
	data, err := os.ReadFile(statsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func Record(e Entry) error {
	mu.Lock()
	defer mu.Unlock()

	entries, err := Load()
	if err != nil {
		entries = nil
	}

	found := false
	for i, existing := range entries {
		if existing.Name == e.Name && existing.ListenPort == e.ListenPort {
			entries[i].Connections += e.Connections
			entries[i].BytesIn += e.BytesIn
			entries[i].BytesOut += e.BytesOut
			entries[i].LastActive = e.LastActive
			found = true
			break
		}
	}

	if !found {
		entries = append(entries, e)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statsPath(), data, 0644)
}
