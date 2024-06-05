package utils

import (
	"Rail-Ticket-Notifier/utils/constants"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"os/exec"
	"runtime"
	"time"
)

func SetupChrome(window fyne.Window) bool {
	//dialog.ShowInformation("Setting Up", constants.CHROME_SETUP_MSG, window)

	label := widget.NewLabel(constants.CHROME_SETUP_MSG)
	customDialog := dialog.NewCustom("Setting Up Chrome", "OK", container.NewVBox(label), window)
	customDialog.Show()

	var chromePath string
	var closeCmd, checkCmd *exec.Cmd
	switch os := runtime.GOOS; os {
	case "windows":
		chromePath = constants.WINDOWS_CHROME_PATH
		closeCmd = exec.Command("taskkill", "/IM", "chrome.exe")
		checkCmd = exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe")
	case "darwin":
		chromePath = constants.MAC_CHROME_PATH
		closeCmd = exec.Command("pkill", "-TERM", "Google Chrome")
		checkCmd = exec.Command("pgrep", "Google Chrome")
	case "linux":
		chromePath = constants.LINUX_CHROME_PATH
		closeCmd = exec.Command("pkill", "-TERM", "chrome")
		checkCmd = exec.Command("pgrep", "chrome")
	default:
		fmt.Printf("Unsupported operating system: %s\n", os)
		return false
	}

	// Close existing Chrome instances gracefully
	err := closeCmd.Run()
	if err != nil {
		fmt.Printf("Error sending terminate signal to Chrome, might be already closed: %v\n", err)
		//return false
	} else {
		fmt.Println("Sent terminate signal to Chrome.")
	}

	// Wait for Chrome to close
	for {
		output, err := checkCmd.Output()
		if err != nil {
			//fmt.Printf("Error checking Chrome processes: %v\n", err)
			break
		}
		if len(output) == 0 {
			fmt.Println("Chrome has closed successfully.")
			break
		}
		fmt.Println("Waiting for Chrome to close...")
		time.Sleep(1 * time.Second)
	}

	// Launch Chrome with remote debugging and restore last session
	launchCmd := exec.Command(chromePath, "--remote-debugging-port=9222", "--restore-last-session")
	err = launchCmd.Start()
	if err != nil {
		fmt.Printf("Error launching Chrome with remote debugging: %v\n", err)
		return false
	} else {
		fmt.Println("Chrome launched with remote debugging port 9222 and previous session restored.")
	}

	label.SetText(constants.CHROME_SETUP_SUCCESS_MSG)
	return true
}
