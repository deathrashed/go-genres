package genrenorm

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GlobalSettings holds configuration shared across all genre taggers
type GlobalSettings struct {
	// Backup settings
	EnableBackup     bool   `json:"enable_backup"`
	PromptForBackup  bool   `json:"prompt_for_backup"`
	
	// Downloads folder settings
	DownloadsPath    string `json:"downloads_path"`
	
	// Swinsian integration
	EnableSwinsian   bool   `json:"enable_swinsian"`
}

// DefaultGlobalSettings returns the default global settings
func DefaultGlobalSettings() GlobalSettings {
	return GlobalSettings{
		EnableBackup:    true,
		PromptForBackup: true,
		DownloadsPath:   "/Volumes/Eksternal/Music/Downloads",
		EnableSwinsian:  false,
	}
}

// LoadGlobalSettings loads global settings from the config file
func LoadGlobalSettings() (GlobalSettings, error) {
	configPath := globalConfigPath()
	settings := DefaultGlobalSettings()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			if saveErr := SaveGlobalSettings(settings); saveErr != nil {
				return settings, saveErr
			}
			return settings, nil
		}
		return settings, err
	}
	
	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, err
	}
	
	return settings, nil
}

// SaveGlobalSettings saves global settings to the config file
func SaveGlobalSettings(settings GlobalSettings) error {
	configPath := globalConfigPath()
	
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// globalConfigPath returns the path to the global settings file
func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "genres", "global-settings.json")
}
