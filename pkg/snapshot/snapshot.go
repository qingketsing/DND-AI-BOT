package snapshot

import (
	"dndbot/pkg/game"
	"dndbot/pkg/session"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type Snapshot struct {
	Timestamp         time.Time
	CurrentBackground string
	Sessions          map[int64]*session.SessionData
	GameStates        map[int64]*game.GroupStateData
}

// SaveSnapshot saves the current state to a JSON file (with .ss extension)
func SaveSnapshot(bg string) (string, error) {
	// Generate filename with timestamp
	filename := fmt.Sprintf("snapshot_%s.ss", time.Now().Format("20060102_150405"))

	snap := Snapshot{
		Timestamp:         time.Now(),
		CurrentBackground: bg,
		Sessions:          session.GlobalManager.ExportData(),
		GameStates:        game.GlobalGameState.ExportData(),
	}

	file, err := os.Create(filename)
	if err != nil {
		logrus.Errorf("Failed to create snapshot file: %v", err)
		return "", err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		logrus.Errorf("Failed to encode snapshot: %v", err)
		return "", err
	}

	return filename, nil
}

// LoadLatestSnapshot finds the latest snapshot file and loads it
func LoadLatestSnapshot() (*Snapshot, string, error) {
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, "", err
	}

	var latestFile string
	var latestTime time.Time

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		// Basic check for snapshot_ prefix and .ss suffix
		if len(f.Name()) > 12 && f.Name()[:9] == "snapshot_" && f.Name()[len(f.Name())-3:] == ".ss" {
			info, err := f.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = f.Name()
			}
		}
	}

	if latestFile == "" {
		return nil, "", nil // No snapshot found
	}

	logrus.Infof("Loading snapshot from %s...", latestFile)
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, latestFile, err
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, latestFile, err
	}

	return &snap, latestFile, nil
}

// DeleteLatestSnapshot deletes the most recent snapshot file
func DeleteLatestSnapshot() (string, error) {
	_, filename, err := LoadLatestSnapshot()
	if err != nil {
		return "", err
	}
	if filename == "" {
		return "", fmt.Errorf("no snapshot found")
	}

	if err := os.Remove(filename); err != nil {
		return filename, err
	}
	return filename, nil
}
