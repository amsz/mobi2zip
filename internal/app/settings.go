package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// settings is what the app keeps between runs.
type settings struct {
	// Archive is the folder converted originals move to, "" for none.
	Archive string `json:"archive,omitempty"`
	// CalibrePath is the ebook-convert chosen by hand, "" to autodetect.
	CalibrePath string `json:"calibrePath,omitempty"`
	// Notify tells the system's notification service when a run ends.
	Notify bool `json:"notify"`
}

// defaultSettings is what a first run uses.
func defaultSettings() settings {
	return settings{Notify: true}
}

// settingsPath is where the settings live: a file in the user data
// directory the system gives the app.
func settingsPath() (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// loadSettings reads the settings, and the defaults when there are none yet.
func loadSettings() settings {
	s := defaultSettings()
	p, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	// Fields the file leaves out keep their defaults, so a file written
	// before a setting existed still turns it on.
	if json.Unmarshal(data, &s) != nil {
		return defaultSettings()
	}
	return s
}

// save writes the settings.
func (s settings) save() error {
	p, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
