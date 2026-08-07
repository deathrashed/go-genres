package genrenorm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type UndoEntry struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
}

type UndoManifest struct {
	Tool      string      `json:"tool"`
	CreatedAt time.Time   `json:"created_at"`
	Files     []UndoEntry `json:"files"`
}

type UndoSessionInfo struct {
	Tool      string
	Timestamp time.Time
	Dir       string
	FileCount int
	SizeBytes int64
}

type UndoSession struct {
	manifest     UndoManifest
	manifestPath string
	backupDir    string
}

func StartUndoSession(tool string) (*UndoSession, error) {
	base, err := undoBaseDir(tool)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	backupDir := filepath.Join(base, now.Format("20060102-150405.000000000"))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, err
	}
	return &UndoSession{
		manifest: UndoManifest{
			Tool:      tool,
			CreatedAt: now,
		},
		manifestPath: filepath.Join(base, "latest.json"),
		backupDir:    backupDir,
	}, nil
}

func (s *UndoSession) Backup(path string) error {
	if s == nil {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	backupPath := filepath.Join(s.backupDir, fmt.Sprintf("%03d-%s", len(s.manifest.Files)+1, safeBackupName(abs)))
	if err := copyFile(abs, backupPath); err != nil {
		return err
	}
	s.manifest.Files = append(s.manifest.Files, UndoEntry{OriginalPath: abs, BackupPath: backupPath})
	return s.save()
}

func RestoreLatestUndo(tool string) (int, error) {
	base, err := undoBaseDir(tool)
	if err != nil {
		return 0, err
	}
	manifestPath := filepath.Join(base, "latest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return 0, err
	}
	var manifest UndoManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return 0, err
	}
	if len(manifest.Files) == 0 {
		return 0, fmt.Errorf("no files in latest undo manifest")
	}
	for i := len(manifest.Files) - 1; i >= 0; i-- {
		entry := manifest.Files[i]
		if err := copyFile(entry.BackupPath, entry.OriginalPath); err != nil {
			return len(manifest.Files) - 1 - i, err
		}
	}
	return len(manifest.Files), nil
}

func (s *UndoSession) save() error {
	data, err := json.MarshalIndent(s.manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.manifestPath, data, 0644)
}

func undoBaseDir(tool string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cleanTool := safeBackupName(tool)
	if cleanTool == "" {
		cleanTool = "genres"
	}
	base := filepath.Join(home, ".config", "genres", "undo", cleanTool)
	return base, os.MkdirAll(base, 0755)
}

func safeBackupName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func ListUndoSessions(tool string) ([]UndoSessionInfo, error) {
	base, err := undoBaseDir(tool)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var sessions []UndoSessionInfo
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "latest.json" {
			continue
		}
		dirPath := filepath.Join(base, entry.Name())
		t, err := time.Parse("20060102-150405.000000000", entry.Name())
		if err != nil {
			continue
		}
		var count int
		var size int64
		filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
			count++
			size += info.Size()
			return nil
		})
		sessions = append(sessions, UndoSessionInfo{
			Tool:      tool,
			Timestamp: t,
			Dir:       dirPath,
			FileCount: count,
			SizeBytes: size,
		})
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})
	return sessions, nil
}

func CleanOldUndoSessions(tool string, keepCount int) (int, error) {
	sessions, err := ListUndoSessions(tool)
	if err != nil {
		return 0, err
	}
	if len(sessions) <= keepCount {
		return 0, nil
	}
	removed := 0
	for i := keepCount; i < len(sessions); i++ {
		if err := os.RemoveAll(sessions[i].Dir); err != nil {
			continue
		}
		removed++
	}
	return removed, nil
}

func FormatBytes(b int64) string {
	f := float64(b)
	switch {
	case f >= 1<<30:
		return fmt.Sprintf("%.1f GB", f/(1<<30))
	case f >= 1<<20:
		return fmt.Sprintf("%.1f MB", f/(1<<20))
	case f >= 1<<10:
		return fmt.Sprintf("%.1f KB", f/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ClearAllBackups removes all backup sessions for a given tool
func ClearAllBackups(tool string) (int, error) {
	base, err := undoBaseDir(tool)
	if err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "latest.json" {
			if err := os.RemoveAll(filepath.Join(base, entry.Name())); err == nil {
				removed++
			}
		}
	}

	// Also remove the latest.json manifest
	manifestPath := filepath.Join(base, "latest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		os.Remove(manifestPath)
	}

	return removed, nil
}

// GetBackupSize returns the total size of all backups for a tool
func GetBackupSize(tool string) (int64, int, error) {
	sessions, err := ListUndoSessions(tool)
	if err != nil {
		return 0, 0, err
	}

	var totalSize int64
	totalFiles := 0
	for _, session := range sessions {
		totalSize += session.SizeBytes
		totalFiles += session.FileCount
	}

	return totalSize, totalFiles, nil
}
