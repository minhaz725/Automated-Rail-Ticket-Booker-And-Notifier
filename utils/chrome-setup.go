package utils

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils/constants"
	"bytes"
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/chromedp/chromedp"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// activeChromePort holds the debug port chosen for this app instance.
var activeChromePort int

// GetDebugChromeURL returns the CDP endpoint URL for this instance's Chrome.
func GetDebugChromeURL() string {
	return fmt.Sprintf("http://localhost:%d", activeChromePort)
}

// GetDebugModeCheckURL returns the /json/version endpoint for this instance's Chrome.
func GetDebugModeCheckURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/json/version", activeChromePort)
}

// EnsureChrome launches a fresh isolated Chrome instance on the port that
// corresponds to the selected instance index (index 1 → port 9222, 2 → 9223,
// etc.) and waits until it is ready. Uses a stable profile dir so the user's
// login session persists across restarts. No-op if already initialised.
func EnsureChrome() error {
	if activeChromePort != 0 {
		return nil
	}

	index := arguments.INSTANCE_INDEX
	if index < 1 || index > 10 {
		index = 1
	}
	port := 9221 + index
	activeChromePort = port
	log.Printf("EnsureChrome: instance %d, debug port %d", index, port)

	chromePath := getChromePath()
	if chromePath == "" {
		activeChromePort = 0
		return fmt.Errorf("Chrome not found on this system")
	}

	// Stable profile dir — login persists across app restarts for this slot
	profileDir := ""
	if cacheBase, err := os.UserCacheDir(); err == nil {
		profileDir = filepath.Join(cacheBase, "Rail-Ticket-Notifier", fmt.Sprintf("chrome-profile-%d", index))
		if err := os.MkdirAll(profileDir, 0700); err != nil {
			log.Printf("EnsureChrome: failed to create profile dir: %v", err)
			profileDir = ""
		}
	}

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--restore-last-session",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
	}
	if profileDir != "" {
		args = append(args, fmt.Sprintf("--user-data-dir=%s", profileDir))
	}
	args = append(args, constants.HOME_URL)

	if err := exec.Command(chromePath, args...).Start(); err != nil {
		activeChromePort = 0
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}
	log.Printf("EnsureChrome: Chrome launched on port %d, waiting for debug port...", port)

	if !waitForDebugPort(20 * time.Second) {
		activeChromePort = 0
		return fmt.Errorf("Chrome debug port %d not reachable within timeout", port)
	}
	log.Printf("EnsureChrome: Chrome ready on port %d", port)
	return nil
}

func SetupChrome(window fyne.Window) bool {

	label := widget.NewLabel(constants.CHROME_SETUP_MSG)
	customDialog := dialog.NewCustom("Setting Up Chrome, Please Wait", "OK", container.NewVBox(label), window)
	// POPUP
	customDialog.Show()

	chromePath := getChromePath()
	if chromePath == "" {
		log.Println("Failed to find Chrome on this system. Aborting")
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}

	// Use the same fixed-port logic as EnsureChrome
	index := arguments.INSTANCE_INDEX
	if index < 1 || index > 10 {
		index = 1
	}
	activeChromePort = 9221 + index
	if activeChromePort == 0 {
		log.Println("Invalid instance index. Aborting")
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}
	log.Printf("Using instance %d, debug port %d", index, activeChromePort)

	// Stable profile dir keyed to instance index
	profileDir := ""
	if cacheBase, err := os.UserCacheDir(); err == nil {
		profileDir = filepath.Join(cacheBase, "Rail-Ticket-Notifier", fmt.Sprintf("chrome-profile-%d", index))
		if err := os.MkdirAll(profileDir, 0700); err != nil {
			log.Printf("Failed to create profile dir: %v\n", err)
			profileDir = ""
		}
	}

	label.SetText("Launching Chrome in debug mode...")
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", activeChromePort),
		"--restore-last-session",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
	}
	if profileDir != "" {
		args = append(args, fmt.Sprintf("--user-data-dir=%s", profileDir))
	}
	// Add the URL to open directly
	args = append(args, constants.HOME_URL)

	launchCmd := exec.Command(chromePath, args...)
	if err := launchCmd.Start(); err != nil {
		log.Printf("Error launching Chrome with remote debugging: %v\n", err)
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}
	log.Printf("Chrome launched with remote debugging port %d.", activeChromePort)

	label.SetText("Waiting for debug port to become ready...")
	if !waitForDebugPort(15 * time.Second) {
		log.Println("Debug port not reachable within timeout.")
		label.SetText(constants.CHROME_SETUP_FAILURE_MSG)
		return false
	}

	// Now show login verification dialog
	customDialog.Hide()

	var loginDialog *dialog.CustomDialog
	loginLabel := widget.NewLabel(constants.CHROME_SETUP_LOGIN_MSG)
	loginLabel.Wrapping = fyne.TextWrapWord

	var testLoginButton *widget.Button
	testLoginButton = widget.NewButton("Test Login", func() {
		testLoginButton.Disable()
		loginLabel.SetText("Verifying login, please wait...")
		go func() {
			if verifyLogin() {
				loginLabel.SetText(constants.CHROME_SETUP_LOGIN_SUCCESS_MSG)
				loginDialog.SetDismissText("Continue")
				testLoginButton.Hide()
			} else {
				loginLabel.SetText(constants.CHROME_SETUP_LOGIN_RETRY_MSG)
				testLoginButton.Enable()
			}
		}()
	})

	loginContent := container.NewVBox(loginLabel, testLoginButton)
	// Create a padded container to make dialog wider
	paddedContent := container.NewPadded(loginContent)
	paddedContent.Resize(fyne.NewSize(350, 150))
	loginDialog = dialog.NewCustom("Login Verification", "Cancel", container.NewCenter(container.NewGridWrap(fyne.NewSize(350, 150), loginContent)), window)
	// POPUP
	//loginDialog.Show()

	// Don't block here - return true and let the dialog handle verification
	// The actual verification happens when user clicks Test Login
	// For now, we return true to proceed, verification happens in dialog
	return true
}

// navigateToEticketSite opens the railway ticket website in the first existing tab of debug Chrome
func navigateToEticketSite() error {
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), GetDebugChromeURL())
	defer cancelAlloc()

	// Get existing targets (tabs)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	targets, err := chromedp.Targets(allocCtx)
	if err != nil {
		return fmt.Errorf("failed to get targets: %w", err)
	}

	if len(targets) == 0 {
		return fmt.Errorf("no browser tabs found")
	}

	// Use the first available page target
	var targetID string
	for _, t := range targets {
		if t.Type == "page" {
			targetID = string(t.TargetID)
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("no page target found")
	}

	// Connect to the existing tab
	tabCtx, tabCancel := chromedp.NewContext(allocCtx, chromedp.WithTargetID(targets[0].TargetID))
	defer tabCancel()

	// Set timeout
	tabCtx, cancel = context.WithTimeout(tabCtx, 15*time.Second)
	defer cancel()

	_ = ctx // suppress unused warning

	return chromedp.Run(tabCtx,
		chromedp.Navigate(constants.HOME_URL),
		chromedp.WaitReady("body"),
	)
}

// verifyLogin checks if user is logged in by navigating to login URL and checking if it redirects
func verifyLogin() bool {
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), GetDebugChromeURL())
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for the verification
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var currentURL string
	err := chromedp.Run(ctx,
		chromedp.Navigate(constants.LOGIN_URL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second), // Wait for any redirects
		chromedp.Location(&currentURL),
	)
	if err != nil {
		log.Printf("Login verification error: %v\n", err)
		return false
	}

	log.Printf("Login verification - Current URL: %s\n", currentURL)

	// If URL is not login URL anymore, user is logged in (redirected to home or dashboard)
	return !strings.HasSuffix(currentURL, "/login")
}

// debugIsReady does a quick GET to the debug version endpoint.
func debugIsReady() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(GetDebugModeCheckURL())
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
		resp, err := client.Get(GetDebugModeCheckURL())
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
