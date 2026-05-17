package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var prefsMu sync.Mutex

func getPrefsDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dir := filepath.Join(configDir, "fyne", "Rail-Ticket-Notifier")
	os.MkdirAll(dir, 0700)
	return dir
}

func getInstancePrefsPath(index int) string {
	return filepath.Join(getPrefsDir(), fmt.Sprintf("preferences-%d.json", index))
}

func LoadLastInstanceIndex() int {
	path := filepath.Join(getPrefsDir(), "last-index.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return 1
	}
	idx, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || idx < 1 || idx > 10 {
		return 1
	}
	return idx
}

func SaveLastInstanceIndex(index int) {
	path := filepath.Join(getPrefsDir(), "last-index.txt")
	os.WriteFile(path, []byte(strconv.Itoa(index)), 0600)
}

func LoadInstancePrefs(index int) map[string]string {
	prefsMu.Lock()
	defer prefsMu.Unlock()

	path := getInstancePrefsPath(index)
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]string)
	}
	var prefs map[string]string
	if err := json.Unmarshal(data, &prefs); err != nil {
		log.Printf("Failed to parse preferences-%d.json: %v", index, err)
		return make(map[string]string)
	}
	return prefs
}

func SaveInstancePrefs(index int, prefs map[string]string) {
	prefsMu.Lock()
	defer prefsMu.Unlock()

	path := getInstancePrefsPath(index)
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal preferences-%d.json: %v", index, err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Printf("Failed to save preferences-%d.json: %v", index, err)
	}
}

func GetPrefWithFallback(prefs map[string]string, key, fallback string) string {
	if v, ok := prefs[key]; ok && v != "" {
		return v
	}
	return fallback
}
