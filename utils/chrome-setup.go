package utils

import (
	"Rail-Ticket-Notifier/utils/constants"
	"bytes"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"log"
	"net/http"
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
		return false
	}

	var closeCmd, checkCmd *exec.Cmd

	switch os := runtime.GOOS; os {
	case "windows":
		closeCmd = exec.Command("taskkill", "/IM", "chrome.exe")
		checkCmd = exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe")
	case "darwin":
		closeCmd = exec.Command("pkill", "-TERM", "Google Chrome")
		checkCmd = exec.Command("pgrep", "Google Chrome")
	case "linux":
		closeCmd = exec.Command("pkill", "-TERM", "chrome")
		checkCmd = exec.Command("pgrep", "chrome")
	default:
		log.Printf("Unsupported operating system: %s\n", os)
		return false
	}

	debugModeCheck, err := http.Get(constants.DEBUG_MODE_CHECK_URL)
	if err == nil {
		if debugModeCheck.StatusCode == http.StatusOK {
			log.Println("Chrome is already running in debug mode.")
			label.SetText(constants.CHROME_SETUP_SUCCESS_MSG)
			customDialog.SetDismissText("Continue")
			return true
		}
	}

	// Close existing Chrome instances gracefully
	err = closeCmd.Run()
	if err != nil {
		log.Printf("Error sending terminate signal to Chrome, might be already closed: %v\n", err)
		//return false
	} else {
		log.Println("Sent terminate signal to Chrome.")
	}

	// Wait for Chrome to close
	for {
		output, err := checkCmd.Output()
		if err != nil {
			//log.Printf("Error checking Chrome processes: %v\n", err)
			break
		}
		if len(output) == 0 {
			log.Println("Chrome has closed successfully.")
			break
		}
		log.Println("Waiting for Chrome to close...")
		time.Sleep(1 * time.Second)
	}

	// Launch Chrome with remote debugging and restore last session
	launchCmd := exec.Command(chromePath, "--remote-debugging-port=9222", "--restore-last-session")
	err = launchCmd.Start()
	if err != nil {
		log.Printf("Error launching Chrome with remote debugging: %v\n", err)
		return false
	} else {
		log.Println("Chrome launched with remote debugging port 9222 and previous session restored.")
	}

	label.SetText(constants.CHROME_SETUP_SUCCESS_MSG)
	customDialog.SetDismissText("Continue")
	return true
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
