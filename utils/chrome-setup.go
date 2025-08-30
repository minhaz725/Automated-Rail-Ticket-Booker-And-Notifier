package utils

import (
	"Rail-Ticket-Notifier/utils/constants"
	"bytes"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func SetupChrome(window fyne.Window) bool {

	label := widget.NewLabel(constants.CHROME_SETUP_MSG)
	customDialog := dialog.NewCustom("Setting Up Chrome, Please Wait", "OK", container.NewVBox(label), window)
	customDialog.Show()

	chromePath := getChromePath()
	if chromePath == "" {
		log.Println("Failed to find Chrome on this system. Aborting")
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}

	// Quick check: is a chrome already running with debugging enabled?
	if debugIsReady() {
		log.Println("Chrome is already running in debug mode.")
		label.SetText(constants.CHROME_SETUP_SUCCESS_MSG)
		customDialog.SetDismissText("Continue")
		return true
	}

	label.SetText("Closing existing Chrome processes (graceful)...")
	if err := killChrome(false); err != nil {
		log.Printf("Graceful chrome termination attempt issue: %v\n", err)
	}

	// Wait for chrome to exit (grace period) else force kill
	if !waitForChromeExit(5 * time.Second) {
		label.SetText("Forcing Chrome to close...")
		if err := killChrome(true); err != nil {
			log.Printf("Force chrome termination attempt issue: %v\n", err)
		}
		if !waitForChromeExit(10 * time.Second) { // longer wait after force
			log.Println("Chrome did not close after force kill, proceeding anyway.")
		}
	}

	// Prepare isolated user data dir so we are sure flags are honored
	profileDir, err := os.MkdirTemp("", "chrome-debug-profile-")
	if err != nil {
		log.Printf("Failed to create temp profile dir: %v\n", err)
	}

	label.SetText("Launching Chrome in debug mode...")
	args := []string{"--remote-debugging-port=9222", "--restore-last-session", "--no-first-run", "--no-default-browser-check"}
	if profileDir != "" {
		args = append(args, fmt.Sprintf("--user-data-dir=%s", profileDir))
	}
	launchCmd := exec.Command(chromePath, args...)
	if err := launchCmd.Start(); err != nil {
		log.Printf("Error launching Chrome with remote debugging: %v\n", err)
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}
	log.Println("Chrome launched with remote debugging port 9222.")

	label.SetText("Waiting for debug port to become ready...")
	if !waitForDebugPort(15 * time.Second) {
		log.Println("Debug port not reachable within timeout.")
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}

	label.SetText(constants.CHROME_SETUP_SUCCESS_MSG)
	customDialog.SetDismissText("Continue")
	return true
}

// debugIsReady does a quick GET to the debug version endpoint.
func debugIsReady() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(constants.DEBUG_MODE_CHECK_URL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// killChrome attempts to terminate chrome processes.
// force = true uses a stronger signal/flag.
func killChrome(force bool) error {
	switch runtime.GOOS {
	case "windows":
		if force {
			return exec.Command("taskkill", "/IM", "chrome.exe", "/F", "/T").Run()
		}
		return exec.Command("taskkill", "/IM", "chrome.exe").Run()
	case "darwin":
		if force {
			return exec.Command("pkill", "-KILL", "Google Chrome").Run()
		}
		return exec.Command("pkill", "-TERM", "Google Chrome").Run()
	case "linux":
		if force {
			return exec.Command("pkill", "-KILL", "chrome").Run()
		}
		return exec.Command("pkill", "-TERM", "chrome").Run()
	default:
		return errors.New("unsupported os")
	}
}

// isChromeRunning checks if any chrome process is still present.
func isChromeRunning() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe")
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		// If chrome not running, output contains only header lines w/o "chrome.exe" occurrence.
		return strings.Contains(strings.ToLower(string(out)), "chrome.exe")
	case "darwin", "linux":
		cmd := exec.Command("pgrep", "chrome")
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("pgrep", "Google Chrome")
		}
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(out)) != ""
	default:
		return false
	}
}

// waitForChromeExit waits up to d for chrome processes to disappear.
func waitForChromeExit(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if !isChromeRunning() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return !isChromeRunning()
}

// waitForDebugPort polls the debug endpoint until available or timeout.
func waitForDebugPort(d time.Duration) bool {
	deadline := time.Now().Add(d)
	client := &http.Client{Timeout: 1 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(constants.DEBUG_MODE_CHECK_URL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func getChromePath() string {
	var cmd *exec.Cmd

	switch os := runtime.GOOS; os {
	case "windows":
		cmd = exec.Command("reg", "query", "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\App Paths\\chrome.exe", "/v", "Path")
	case "darwin":
		cmd = exec.Command("mdfind", "kMDItemCFBundleIdentifier == 'com.google.Chrome'")
	case "linux":
		cmd = exec.Command("which", "google-chrome")
		// Use "which chromium-browser" if you use Chromium instead of Google Chrome
	default:
		return ""
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return ""
	}
	output := strings.TrimSpace(out.String())

	if runtime.GOOS == "windows" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "REG_SZ") {
				pathParts := strings.Split(line, "REG_SZ")
				if len(pathParts) > 1 {
					return strings.TrimSpace(pathParts[1]) + "\\chrome.exe"
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		if output != "" {
			return output + "/Contents/MacOS/Google Chrome"
		}
	} else if runtime.GOOS == "linux" {
		if output != "" {
			return output
		}
	}
	return ""
}
