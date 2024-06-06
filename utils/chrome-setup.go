package utils

import (
	"Rail-Ticket-Notifier/utils/constants"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

func SetupChrome(window fyne.Window) bool {
	//dialog.ShowInformation("Setting Up", constants.CHROME_SETUP_MSG, window)

	label := widget.NewLabel(constants.CHROME_SETUP_MSG)
	customDialog := dialog.NewCustom("Setting Up Chrome, Please Wait", "Ok", container.NewVBox(label), window)
	customDialog.Show()

	var chromePath string
	var closeCmd, checkCmd *exec.Cmd

	switch os := runtime.GOOS; os {
	case "windows":
		chromePath = constants.WINDOWS_CHROME_PATH
		closeCmd = exec.Command("taskkill", "/IM", "chrome.exe")
		checkCmd = exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe")
		//debugCheckCmd = exec.Command("cmd", "/C", `tasklist /fi "imagename eq chrome.exe" | findstr "--remote-debugging-port"`)
	case "darwin":
		chromePath = constants.MAC_CHROME_PATH
		closeCmd = exec.Command("pkill", "-TERM", "Google Chrome")
		checkCmd = exec.Command("pgrep", "Google Chrome")
		//debugCheckCmd = exec.Command("pgrep", "-fl", "--", "--remote-debugging-port")
	case "linux":
		chromePath = constants.LINUX_CHROME_PATH
		closeCmd = exec.Command("pkill", "-TERM", "chrome")
		checkCmd = exec.Command("pgrep", "chrome")
		//debugCheckCmd = exec.Command("pgrep", "-fl", "--", "--remote-debugging-port")
	default:
		log.Printf("Unsupported operating system: %s\n", os)
		return false
	}

	debugModeCheck, err := http.Get(constants.DEBUG_MODE_CHECK_URL)
	if err == nil {
		if debugModeCheck.StatusCode == http.StatusOK {
			log.Println("Chrome is already running in debug mode.")
			label.SetText(constants.CHROME_SETUP_SUCCESS_MSG)
			customDialog.SetDismissText("Go!")
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
	customDialog.SetDismissText("Go!")
	return true
}
