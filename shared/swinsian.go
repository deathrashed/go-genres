package genrenorm

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsSwinsianRunning reports whether Swinsian is currently running.
func IsSwinsianRunning() bool {
	// osascript returns "true" / "false"
	out, err := exec.Command("osascript", "-e",
		`tell application "System Events" to (name of processes) contains "Swinsian"`).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// CurrentTrackPath returns the POSIX path of the track currently playing in Swinsian.
func CurrentTrackPath() (string, error) {
	if !IsSwinsianRunning() {
		return "", fmt.Errorf("Swinsian is not running")
	}

	script := `tell application "Swinsian"
	set t to current track
	if t is missing value then
		return ""
	else
		try
			return path of t
		on error
			try
				return location of t
			on error
				return ""
			end try
		end try
	end if
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("Swinsian AppleScript error: %w", err)
	}

	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", fmt.Errorf("no track currently playing in Swinsian")
	}
	return path, nil
}

// CurrentAlbumDir returns the directory containing the currently playing track
// (i.e. the album folder for typical library layouts).
func CurrentAlbumDir() (string, error) {
	track, err := CurrentTrackPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(track), nil
}

// SelectedTrackPaths returns POSIX paths for the tracks selected in Swinsian’s
// front window. Paths are one per line from AppleScript so spaces are safe.
func SelectedTrackPaths() ([]string, error) {
	if !IsSwinsianRunning() {
		return nil, fmt.Errorf("Swinsian is not running")
	}

	script := `tell application "Swinsian"
	set selectedTracks to selection of window 1
	if selectedTracks is {} then
		return ""
	else
		set pathList to {}
		repeat with t in selectedTracks
			try
				set end of pathList to path of t
			on error
				try
					set end of pathList to location of t
				end try
			end try
		end repeat
		set AppleScript's text item delimiters to linefeed
		return pathList as text
	end if
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, fmt.Errorf("Swinsian AppleScript error: %w", err)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return nil, fmt.Errorf("no tracks selected in Swinsian")
	}

	var paths []string
	for _, line := range strings.Split(result, "\n") {
		p := strings.TrimSpace(line)
		if p != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no valid track paths found")
	}
	return paths, nil
}
