package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	genrenorm "go-genres/shared"

	"github.com/bogem/id3v2/v2"
	"github.com/charmbracelet/lipgloss"
)

// ─── Configuration ──────────────────────────────────────────────

type config struct {
	write     bool
	undo      bool
	maxGenres int
	delayMs   int
	target    string
}

var cfg config

var httpClient = &http.Client{Timeout: 20 * time.Second}

// ─── Lip Gloss Styles (Spotify Green) ───────────────────────────

var (
	sHeaderStyle      lipgloss.Style
	sLabelStyle       lipgloss.Style
	sSuccessStyle     lipgloss.Style
	sErrorStyle       lipgloss.Style
	sWarnStyle        lipgloss.Style
	sArtistStyle      lipgloss.Style
	sGenreStyle       lipgloss.Style
	sDimStyle         lipgloss.Style
	sCountStyle       lipgloss.Style
	sSeparatorStyle   lipgloss.Style
	sBorderStyle      lipgloss.Style
	sTitleStyle       lipgloss.Style
	sTitleIconStyle   lipgloss.Style
	sDescStyle        lipgloss.Style
	sSectionStyle     lipgloss.Style
	sSectionIconStyle lipgloss.Style
	sKeyStyle         lipgloss.Style
	sActionLabelStyle lipgloss.Style
	sMenuDescStyle    lipgloss.Style
	sSepStyle         lipgloss.Style
	sPromptStyle      lipgloss.Style
	sMutedStyle       lipgloss.Style
	sValueStyle       lipgloss.Style
	sPathStyle        lipgloss.Style
	sModeStyle        lipgloss.Style
	sWriteModeStyle   lipgloss.Style
	sInfoStyle        lipgloss.Style
	sArrowStyle       lipgloss.Style
	sFileStyle        lipgloss.Style
	sStatusLabelStyle lipgloss.Style
	sRowIconStyle     lipgloss.Style
)

func initSpotifyStyles() {
	ld := func(dark, light lipgloss.Color) lipgloss.Color {
		if lipgloss.HasDarkBackground() {
			return dark
		}
		return light
	}

	// Spotify green palette
	spotifyGreen := lipgloss.Color("#1DB954")
	darkBg := lipgloss.Color("#191414")
	lightGreen := lipgloss.Color("#1ed760")
	mutedGreen := lipgloss.Color("#169c46")
	darkGray := lipgloss.Color("#535353")
	medGray := lipgloss.Color("#b3b3b3")
	white := lipgloss.Color("#ffffff")

	green := ld(spotifyGreen, lipgloss.Color("#006b3e"))
	borderC := ld(darkGray, medGray)
	sepC := ld(darkGray, medGray)

	sHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(spotifyGreen)
	sLabelStyle = lipgloss.NewStyle().Foreground(medGray)
	sSuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(lightGreen)
	sErrorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e74c3c"))
	sWarnStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f39c12"))
	sArtistStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(white, darkBg))
	sGenreStyle = lipgloss.NewStyle().Foreground(lightGreen)
	sDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#535353"))
	sCountStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(white, darkBg))
	sSeparatorStyle = lipgloss.NewStyle().Foreground(sepC)

	sBorderStyle = lipgloss.NewStyle().Foreground(borderC)
	sTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(green)
	sTitleIconStyle = lipgloss.NewStyle().Bold(true).Foreground(spotifyGreen)
	sDescStyle = lipgloss.NewStyle().Foreground(medGray)
	sSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(white, darkBg))
	sSectionIconStyle = lipgloss.NewStyle().Foreground(spotifyGreen)
	sKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(spotifyGreen)
	sRowIconStyle = lipgloss.NewStyle().Foreground(medGray)
	sActionLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(white, darkBg))
	sMenuDescStyle = lipgloss.NewStyle().Foreground(medGray)
	sSepStyle = lipgloss.NewStyle().Foreground(sepC)
	sPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lightGreen)
	sMutedStyle = lipgloss.NewStyle().Foreground(mutedGreen)
	sValueStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(white, darkBg))
	sPathStyle = lipgloss.NewStyle().Foreground(mutedGreen)
	sModeStyle = lipgloss.NewStyle().Bold(true).Foreground(medGray)
	sWriteModeStyle = lipgloss.NewStyle().Bold(true).Foreground(lightGreen)
	sInfoStyle = lipgloss.NewStyle().Foreground(medGray)
	sArrowStyle = lipgloss.NewStyle().Foreground(medGray)
	sFileStyle = lipgloss.NewStyle().Foreground(ld(white, darkBg))
	sStatusLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(medGray)
}

func init() {
	initSpotifyStyles()
}

// ─── Helpers ────────────────────────────────────────────────────

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func readLine() string {
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}

func readInput(prompt string) string {
	fmt.Print(sPromptStyle.Render("❯ " + prompt + ":"))
	fmt.Print(" ")
	return readLine()
}

func promptReturn() {
	fmt.Println()
	fmt.Println("  " + sMutedStyle.Render("Press Enter to return..."))
	readLine()
}

func centerText(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func renderSep(width int) string {
	return sSepStyle.Render(strings.Repeat("─", width))
}

func renderHR(width int) string {
	return sSepStyle.Render(strings.Repeat("═", width))
}

// ─── Spinner ────────────────────────────────────────────────────

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func startSpinner() *time.Ticker {
	tick := time.NewTicker(80 * time.Millisecond)
	i := 0
	go func() {
		for range tick.C {
			fmt.Printf("\r  %s ", spinFrames[i%len(spinFrames)])
			i++
		}
	}()
	return tick
}

func stopSpinner(t *time.Ticker) {
	t.Stop()
	fmt.Print("\r                                                                    \r")
}

// ─── Renderers ──────────────────────────────────────────────────

const appWidth = 63

func renderTitle(left, icon, right string) string {
	return sTitleStyle.Render(left) + "  " + sTitleIconStyle.Render(icon) + "  " + sTitleStyle.Render(right)
}

func renderPageHeader(titleLeft, icon, titleRight, desc string) string {
	innerWidth := 32
	plain := titleLeft + "  " + icon + "  " + titleRight
	pad := innerWidth - lipgloss.Width(plain)
	if pad < 0 {
		pad = 0
	}
	leftPad := pad / 2
	rightPad := pad - leftPad
	title := strings.Repeat(" ", leftPad) + sTitleStyle.Render(titleLeft) + "  " + sTitleIconStyle.Render(icon) + "  " + sTitleStyle.Render(titleRight) + strings.Repeat(" ", rightPad)

	return strings.Join([]string{
		sBorderStyle.Render("               ╭────────────────────────────────╮"),
		sBorderStyle.Render("╭──────────────┤") + title + sBorderStyle.Render("├─────────────╮"),
		sBorderStyle.Render("│              ╰────────────────────────────────╯             │"),
		sBorderStyle.Render("│") + sDescStyle.Render(centerText(desc, appWidth-2)) + sBorderStyle.Render("│"),
		sBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSection(icon, title string) string {
	sep := sSepStyle.Render(strings.Repeat("─", 56))
	return fmt.Sprintf("  %s %s\n  %s\n", sSectionIconStyle.Render(icon), sSectionStyle.Render(title), sep)
}

func renderMenuItem(key, icon, label, desc string) string {
	return fmt.Sprintf("   %s  %s  %s%s",
		sKeyStyle.Render(key),
		sRowIconStyle.Render(icon),
		sActionLabelStyle.Render(padRight(label, 22)),
		sMenuDescStyle.Render(desc))
}

func renderSettingItem(key, label, value string) string {
	return fmt.Sprintf("   %s  %s%s",
		sKeyStyle.Render(key),
		sActionLabelStyle.Render(padRight(label, 22)),
		sValueStyle.Render(value))
}

func renderMainHeader() string {
	title := renderTitle("S P O T I F Y", "", "G E N R E S")
	return strings.Join([]string{
		sBorderStyle.Render("               ╭───────────────────────────────╮"),
		sBorderStyle.Render("╭──────────────┤ ") + title + sBorderStyle.Render(" ├──────────────╮"),
		sBorderStyle.Render("│              ╰───────────────────────────────╯              │"),
		sBorderStyle.Render("│") + sDescStyle.Render(centerText("Fetch artist genres from the Spotify API", appWidth-2)) + sBorderStyle.Render("│"),
		sBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSettingsHeader() string {
	title := strings.Repeat(" ", 5) + sTitleIconStyle.Render("") + "  " + sTitleStyle.Render("S E T T I N G S") + strings.Repeat(" ", 7)
	return strings.Join([]string{
		sBorderStyle.Render("               ╭───────────────────────────────╮"),
		sBorderStyle.Render("╭──────────────┤ ") + title + sBorderStyle.Render("├──────────────╮"),
		sBorderStyle.Render("│              ╰───────────────────────────────╯              │"),
		sBorderStyle.Render("│") + sDescStyle.Render(centerText("Configure default Spotify scraping behavior", appWidth-2)) + sBorderStyle.Render("│"),
		sBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderStatusLine(s savedSettings) string {
	modeLabel := "Dry"
	modeStyler := sModeStyle
	if s.Write {
		modeLabel = "Write"
		modeStyler = sWriteModeStyle
	}
	return fmt.Sprintf("        %s %s: %s  %s  %s %s: %s  %s  %s %s: %s",
		sSectionIconStyle.Render(""),
		sStatusLabelStyle.Render("Genres"),
		sValueStyle.Render(fmt.Sprintf("%d", s.MaxGenres)),
		sDimStyle.Render("•"),
		sSectionIconStyle.Render(""),
		sStatusLabelStyle.Render("Mode"),
		modeStyler.Render(modeLabel),
		sDimStyle.Render("•"),
		sSectionIconStyle.Render("󱦞"),
		sStatusLabelStyle.Render("Delay"),
		sValueStyle.Render(fmt.Sprintf("%dms", s.DelayMs)),
	)
}

func renderMainMenu(s savedSettings) {
	globalSettings, _ := genrenorm.LoadGlobalSettings()

	fmt.Print(renderMainHeader())
	fmt.Println()
	fmt.Println(renderStatusLine(s))
	fmt.Println()
	fmt.Print(renderSection("", "Input"))
	fmt.Println(renderMenuItem("1", "\uf506", "Path", "Enter file/folder paths"))
	fmt.Println(renderMenuItem("2", "\U000f1266", "Clipboard", "Use path from clipboard"))
	if globalSettings.EnableSwinsian {
		fmt.Println(renderMenuItem("p", "▶", "Playing", "Tag currently playing in Swinsian"))
		fmt.Println(renderMenuItem("e", "", "Selected", "Tag selected tracks in Swinsian"))
	}
	fmt.Println()
	fmt.Print(renderSection("", "Search"))
	fmt.Println(renderMenuItem("3", "\U000f0036", "Finder", "Select folder with Finder"))
	fmt.Println(renderMenuItem("4", "", "FZF", "Search from audio library"))
	fmt.Println()
	fmt.Print(renderSection("", "Actions"))
	fmt.Println(renderMenuItem("5", "\uf0e2", "Run Last", "Re-run previous target"))
	fmt.Println(renderMenuItem("6", "\uf001", "Downloads", "Run on downloads folder"))
	fmt.Println(renderMenuItem("u", "󰕌", "Undo Last", "Restore files from last write"))
	fmt.Println()
	fmt.Print(renderSection("", "System"))
	fmt.Println(renderMenuItem("s", "󰆍", "Settings", "Change defaults"))
	fmt.Println(renderMenuItem("q", "", "Quit", "Exit"))
	fmt.Println()
}

func renderSettingsMenu(s savedSettings) {
	globalSettings, _ := genrenorm.LoadGlobalSettings()

	var modeStr string
	var modeStyler lipgloss.Style
	if s.Write {
		modeStr = "write"
		modeStyler = sWriteModeStyle
	} else {
		modeStr = "dry-run"
		modeStyler = sModeStyle
	}

	fmt.Print(renderSettingsHeader())
	fmt.Println()
	fmt.Print(renderSection("󰟓", "Options"))
	fmt.Println(renderSettingItem("w", "Default Mode", modeStyler.Render(modeStr)))
	fmt.Println(renderSettingItem("g", "Max Genres", fmt.Sprintf("%d", s.MaxGenres)))
	fmt.Println(renderSettingItem("d", "Delay", fmt.Sprintf("%d ms", s.DelayMs)))
	fmt.Println()
	fmt.Print(renderSection("", "Backup"))
	backupStatus := "disabled"
	if globalSettings.EnableBackup {
		backupStatus = "enabled"
	}
	fmt.Println(renderSettingItem("a", "Auto Backup", backupStatus))
	promptStatus := "off"
	if globalSettings.PromptForBackup {
		promptStatus = "on"
	}
	fmt.Println(renderSettingItem("p", "Prompt Before Write", promptStatus))
	fmt.Println(renderSettingItem("c", "Clear All Backups", "Remove all backup files"))
	fmt.Println()
	fmt.Print(renderSection("", "Swinsian"))
	swinsianStatus := "disabled"
	if globalSettings.EnableSwinsian {
		swinsianStatus = "enabled"
	}
	fmt.Println(renderSettingItem("n", "Swinsian Integration", swinsianStatus))
	fmt.Println()
	fmt.Print(renderSection("󱐋", "Actions"))
	fmt.Println(renderMenuItem("s", "\uf00c", "Save", "Write settings and return"))
	fmt.Println(renderMenuItem("x", "\uf00d", "Back", "Return without saving"))
	fmt.Println()
}

// ─── Constants & Settings ─────────────────────────────────────

type savedSettings struct {
	Write     bool `json:"write"`
	MaxGenres int  `json:"max_genres"`
	DelayMs   int  `json:"delay_ms"`
}

var defaultSettings = savedSettings{
	Write:     false,
	MaxGenres: 3,
	DelayMs:   500,
}

var reader = bufio.NewReader(os.Stdin)

func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".config", "genres")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "spotify.json")
}

func loadSettings() savedSettings {
	p := settingsPath()
	if p == "" {
		return defaultSettings
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return defaultSettings
	}
	var s savedSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultSettings
	}
	if s.MaxGenres < 1 {
		s.MaxGenres = defaultSettings.MaxGenres
	}
	if s.DelayMs < 0 {
		s.DelayMs = defaultSettings.DelayMs
	}
	return s
}

func saveSettings(s savedSettings) {
	p := settingsPath()
	if p == "" {
		return
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(p, data, 0644)
}

// ─── Input Screens ────────────────────────────────────────────

func readPathInputScreen() string {
	clearScreen()
	fmt.Println(renderPageHeader("P A T H", "\uf013", "I N P U T", "Enter a file or folder path"))
	fmt.Println()
	fmt.Println("                                    " + sMenuDescStyle.Render("b back • q quit • enter confirm"))
	fmt.Print(sPromptStyle.Render("❯ Path:"))
	fmt.Print(" ")
	input := readLine()
	if input == "" || strings.EqualFold(input, "b") || strings.EqualFold(input, "q") {
		return ""
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		fmt.Println("   " + sErrorStyle.Render("󰅏 Invalid path"))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + sWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + sPathStyle.Render(abs))
		promptReturn()
		return ""
	}
	return abs
}

func readClipboard() string {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readClipboardScreen() string {
	clearScreen()
	fmt.Println(renderPageHeader("", "󱉦", "C L I P B O A R D", "Read a file or folder path from clipboard"))
	fmt.Println()
	fmt.Print(renderSection("󱘟", "Status"))
	fmt.Println("   " + sMutedStyle.Render("󱘟 Reading clipboard..."))
	fmt.Println()

	content := readClipboard()
	if content == "" {
		fmt.Println("   " + sWarnStyle.Render("󰅏 Empty: Copy a file or folder path first."))
		promptReturn()
		return ""
	}
	abs, err := filepath.Abs(content)
	if err != nil {
		fmt.Println("   " + sWarnStyle.Render("󰅏 Invalid path: "+content))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + sWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + sPathStyle.Render(abs))
		promptReturn()
		return ""
	}
	return abs
}

func openFinderDialog() string {
	script := `tell application "Finder"
		set theDir to choose folder with prompt "Select folder with MP3 files"
		return POSIX path of theDir
	end tell`
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runFZFSearch() (string, error) {
	baseDir, _ := os.UserHomeDir()
	if st, err := os.Stat("/Volumes/Eksternal/Audio"); err == nil && st.IsDir() {
		baseDir = "/Volumes/Eksternal/Audio"
	}

	if _, err := exec.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("fzf not found. Install with: brew install fzf")
	}

	script := `set -euo pipefail
base="${1:-$HOME}"
find "$base" -type d 2>/dev/null | fzf --exact --prompt=' ❯ ' --header='Enter:Select  Esc:Cancel'`

	cmd := exec.Command("bash", "-lc", script, "fzf-albums", baseDir)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cancelled")
	}
	selected := strings.TrimSpace(string(out))
	if selected == "" {
		return "", fmt.Errorf("cancelled")
	}
	return filepath.Clean(selected), nil
}

func runFZFScreen() string {
	clearScreen()
	fmt.Println(renderPageHeader("F Z F", "", "S E A R C H", "Search album folders from the library"))
	fmt.Println()
	fmt.Print(renderSection("󰪵", "Opening FZF"))
	fmt.Println("   " + sMutedStyle.Render("") + " " + sStatusLabelStyle.Render("Mode") + ": " + sMutedStyle.Render("directory search with find"))
	fmt.Println()

	path, err := runFZFSearch()
	if err != nil {
		fmt.Println("   " + sWarnStyle.Render("󰜺 Cancelled") + "  " + sMutedStyle.Render(err.Error()))
		promptReturn()
		return ""
	}
	if path == "" {
		promptReturn()
		return ""
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("   " + sWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + sPathStyle.Render(path))
		promptReturn()
		return ""
	}
	return path
}

// ─── Settings Menu ─────────────────────────────────────────────

func runSettingsMenu() savedSettings {
	s := loadSettings()
	for {
		clearScreen()
		renderSettingsMenu(s)
		choice := readInput("Choice")

		switch choice {
		case "w":
			s.Write = !s.Write
		case "g":
			s.MaxGenres++
			if s.MaxGenres > 10 {
				s.MaxGenres = 1
			}
		case "d":
			delays := []int{0, 100, 250, 500, 750, 1000, 1500, 2000}
			idx := 0
			for i, v := range delays {
				if v == s.DelayMs {
					idx = i
					break
				}
			}
			s.DelayMs = delays[(idx+1)%len(delays)]
		case "a":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			globalSettings.EnableBackup = !globalSettings.EnableBackup
			genrenorm.SaveGlobalSettings(globalSettings)
		case "p":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			globalSettings.PromptForBackup = !globalSettings.PromptForBackup
			genrenorm.SaveGlobalSettings(globalSettings)
		case "c":
			clearScreen()
			fmt.Println(renderPageHeader("C L E A R", "󱘿", "B A C K U P S", "Remove all backup files for Spotify"))
			fmt.Println()
			fmt.Print(renderSection("", "Warning"))
			fmt.Println("   " + sWarnStyle.Render("This will permanently delete all backup files."))
			fmt.Println("   " + sMutedStyle.Render("This action cannot be undone."))
			fmt.Println()
			fmt.Print(sPromptStyle.Render(" Are you sure? [y/N]:"))
			fmt.Print(" ")
			answer := strings.ToLower(readLine())
			if answer == "y" || answer == "yes" {
				removed, err := genrenorm.ClearAllBackups("spotify")
				if err != nil {
					fmt.Println("   " + sErrorStyle.Render("✖ Error clearing backups: "+err.Error()))
				} else {
					fmt.Println("   " + sSuccessStyle.Render(fmt.Sprintf("◆ Cleared %d backup session(s)", removed)))
				}
				promptReturn()
			}
		case "n":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			globalSettings.EnableSwinsian = !globalSettings.EnableSwinsian
			genrenorm.SaveGlobalSettings(globalSettings)
		case "s":
			saveSettings(s)
			cfg.write = s.Write
			cfg.maxGenres = s.MaxGenres
			cfg.delayMs = s.DelayMs
			return s
		case "x", "q":
			return s
		}
	}
}

// ─── Spotify API ────────────────────────────────────────────────

func loadEnvVar(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	envPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".env"),
		filepath.Join(os.Getenv("HOME"), ".kenv", ".env"),
	}

	for _, p := range envPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, key+"=") {
				val := strings.TrimPrefix(line, key+"=")
				val = strings.Trim(val, `"'`)
				if val != "" {
					return val
				}
			}
		}
	}
	return ""
}

var (
	spotifyClientID     string
	spotifyClientSecret string
	spotifyToken        string
	spotifyTokenExpiry  time.Time
)

func getSpotifyToken() (string, error) {
	if spotifyToken != "" && time.Now().Before(spotifyTokenExpiry) {
		return spotifyToken, nil
	}

	spotifyClientID = loadEnvVar("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = loadEnvVar("SPOTIFY_CLIENT_SECRET")

	if spotifyClientID == "" || spotifyClientSecret == "" {
		return "", fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set in ~/.env")
	}

	creds := base64.StdEncoding.EncodeToString([]byte(spotifyClientID + ":" + spotifyClientSecret))
	body := url.Values{"grant_type": {"client_credentials"}}.Encode()

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("failed to parse token response: %s", string(raw))
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response: %s", string(raw))
	}

	spotifyToken = result.AccessToken
	spotifyTokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return spotifyToken, nil
}

func spotifyGet(urlPath string) (map[string]interface{}, error) {
	token, err := getSpotifyToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", "https://api.spotify.com"+urlPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "MP3GenreUpdater/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		retryAfter := 1
		if v := resp.Header.Get("Retry-After"); v != "" {
			fmt.Sscanf(v, "%d", &retryAfter)
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
		return spotifyGet(urlPath)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func fetchSpotifyArtistGenres(artist string) []string {
	query := url.QueryEscape(artist)
	data, err := spotifyGet("/v1/search?q=" + query + "&type=artist&limit=5")
	if err != nil || data == nil {
		return nil
	}

	artists, ok := data["artists"].(map[string]interface{})
	if !ok {
		return nil
	}
	items, ok := artists["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil
	}

	// Find exact match or most popular
	lowerArtist := strings.ToLower(strings.TrimSpace(artist))
	type candidate struct {
		name       string
		popularity int
		genres     []string
	}
	var candidates []candidate
	var exact *candidate

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := itemMap["name"].(string)
		pop, _ := itemMap["popularity"].(float64)
		genresRaw, _ := itemMap["genres"].([]interface{})

		var genres []string
		for _, g := range genresRaw {
			if s, ok := g.(string); ok {
				genres = append(genres, s)
			}
		}

		c := candidate{name: name, popularity: int(pop), genres: genres}
		if strings.ToLower(strings.TrimSpace(name)) == lowerArtist {
			exact = &c
		}
		candidates = append(candidates, c)
	}

	var best candidate
	if exact != nil {
		best = *exact
	} else {
		// Sort by popularity descending
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].popularity > candidates[j].popularity
		})
		if len(candidates) > 0 {
			best = candidates[0]
		}
	}

	if len(best.genres) == 0 {
		return nil
	}
	return best.genres
}

// ─── File Walking ──────────────────────────────────────────────

func findMP3Files(target string) ([]string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if info.Mode().IsRegular() && strings.HasSuffix(strings.ToLower(target), ".mp3") {
		return []string{target}, nil
	}
	if !info.IsDir() {
		return nil, nil
	}

	var files []string
	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && strings.HasSuffix(strings.ToLower(path), ".mp3") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ─── MP3 Tag Reading/Writing ──────────────────────────────────

func readTags(file string) (artist, album string) {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return "", ""
	}
	defer tag.Close()
	return strings.TrimSpace(tag.Artist()), strings.TrimSpace(tag.Album())
}

func writeGenre(file, genre string) bool {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return false
	}
	defer tag.Close()
	tag.SetGenre(genre)
	if err := tag.Save(); err != nil {
		return false
	}
	return true
}

// ─── Interactive Processing ───────────────────────────────────

func printlnStyle(st lipgloss.Style, format string, a ...interface{}) {
	fmt.Println(st.Render(fmt.Sprintf(format, a...)))
}

func printfStyle(st lipgloss.Style, format string, a ...interface{}) {
	fmt.Print(st.Render(fmt.Sprintf(format, a...)))
}

type fileInfo struct {
	path   string
	artist string
	album  string
}

func artistSetLookup(norm string, infos []fileInfo) string {
	for _, fi := range infos {
		if strings.ToLower(strings.TrimSpace(fi.artist)) == norm && fi.artist != "" {
			return fi.artist
		}
	}
	return norm
}

func validateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("No MP3 file/folder path provided.")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("Path does not exist: %s", abs)
	}
	cfg.target = abs
	return nil
}

type processingSummary struct {
	dryCandidates int
	updated       int
	skipped       int
	stillMissing  int
	failed        int
}

func processTarget(target string) (*processingSummary, error) {
	modeTag := sWarnStyle.Render("[DRY RUN]")
	if cfg.write {
		modeTag = sSuccessStyle.Render("[LIVE]")
	}

	sum := &processingSummary{}
	var undoSession *genrenorm.UndoSession
	if cfg.write {
		session, err := genrenorm.StartUndoSession("spotify")
		if err != nil {
			return sum, fmt.Errorf("could not create undo backup session: %w", err)
		}
		undoSession = session
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		sHeaderStyle.Render("Genres from Spotify"),
		"  ",
		modeTag,
	)
	fmt.Println("  " + header)
	fmt.Println("  " + renderSep(50))
	fmt.Printf("  %s %s\n", sLabelStyle.Render("Target:"), sCountStyle.Render(cfg.target))
	fmt.Printf("  %s %s\n", sLabelStyle.Render("Max genres:"), sCountStyle.Render(fmt.Sprintf("%d", cfg.maxGenres)))
	if !cfg.write {
		fmt.Println("  " + sDimStyle.Render("Pass --write to apply changes."))
	}
	fmt.Println()

	files, err := findMP3Files(cfg.target)
	if err != nil {
		printlnStyle(sErrorStyle, "✗ Error: %v", err)
		return sum, err
	}
	if len(files) == 0 {
		printlnStyle(sWarnStyle, "No MP3 files found.")
		return sum, nil
	}

	fmt.Printf("  %s %s %s\n",
		sLabelStyle.Render("Found"),
		sCountStyle.Render(fmt.Sprintf("%d", len(files))),
		sLabelStyle.Render("MP3 file(s) — reading artist tags..."),
	)

	var fileInfos []fileInfo
	for _, f := range files {
		artist, album := readTags(f)
		fileInfos = append(fileInfos, fileInfo{path: f, artist: artist, album: album})
	}

	type artistEntry struct {
		name   string
		albums []string
	}
	artistAlbums := make(map[string]map[string]bool)
	for _, fi := range fileInfos {
		if fi.artist == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(fi.artist))
		if artistAlbums[key] == nil {
			artistAlbums[key] = make(map[string]bool)
		}
		if fi.album != "" {
			artistAlbums[key][strings.ToLower(strings.TrimSpace(fi.album))] = true
		}
	}
	uniqueArtists := make([]artistEntry, 0, len(artistAlbums))
	for norm, orig := range artistAlbums {
		name := artistSetLookup(norm, fileInfos)
		albums := make([]string, 0, len(orig))
		for a := range orig {
			albums = append(albums, a)
		}
		sort.Strings(albums)
		uniqueArtists = append(uniqueArtists, artistEntry{name: name, albums: albums})
	}
	sort.Slice(uniqueArtists, func(i, j int) bool {
		return uniqueArtists[i].name < uniqueArtists[j].name
	})

	noArtistFiles := 0
	for _, fi := range fileInfos {
		if fi.artist == "" {
			noArtistFiles++
		}
	}

	fmt.Println()
	missingInfo := ""
	if noArtistFiles > 0 {
		missingInfo = sDimStyle.Render(fmt.Sprintf(" (%d files missing artist tag)", noArtistFiles))
	}
	fmt.Printf("  %s %s %s%s\n",
		sLabelStyle.Render("Fetching genres from Spotify for"),
		sCountStyle.Render(fmt.Sprintf("%d", len(uniqueArtists))),
		sLabelStyle.Render("unique artist(s)"),
		missingInfo,
	)
	fmt.Println()

	artistCache := make(map[string][]string)
	delay := time.Duration(cfg.delayMs) * time.Millisecond
	foundCount := 0

	for i, entry := range uniqueArtists {
		artist := entry.name
		counter := sDimStyle.Render(fmt.Sprintf("[%d/%d]", i+1, len(uniqueArtists)))
		artistName := sArtistStyle.Render(artist)
		fmt.Printf("  %s %s  ", counter, artistName)

		tick := startSpinner()
		time.Sleep(delay)
		genres := fetchSpotifyArtistGenres(artist)
		stopSpinner(tick)

		if len(genres) > 0 {
			normalized := genrenorm.ExpandGenres(genres)
			if len(normalized) > cfg.maxGenres {
				normalized = normalized[:cfg.maxGenres]
			}
			artistCache[strings.ToLower(strings.TrimSpace(artist))] = normalized
			foundCount++

			fmt.Printf("%s %s\n",
				sSuccessStyle.Render("✓"),
				sGenreStyle.Render(strings.Join(normalized, "; ")),
			)
		} else {
			artistCache[strings.ToLower(strings.TrimSpace(artist))] = []string{}
			fmt.Println(sErrorStyle.Render("✗  no genres found on Spotify"))
		}
	}

	fmt.Println()
	fmt.Println("  " + renderSep(50))
	fmt.Printf("  %s %s %s %s %s\n",
		sLabelStyle.Render("Found genres for"),
		sCountStyle.Render(fmt.Sprintf("%d", foundCount)),
		sLabelStyle.Render("/"),
		sCountStyle.Render(fmt.Sprintf("%d", len(uniqueArtists))),
		sLabelStyle.Render("artists"),
	)
	fmt.Println()

	fmt.Println("  " + sLabelStyle.Render("Processing files..."))
	fmt.Println()

	for _, fi := range fileInfos {
		base := filepath.Base(fi.path)

		if fi.artist == "" {
			sum.stillMissing++
			fmt.Printf("  %s %s\n", sDimStyle.Render("∘"), sDimStyle.Render(base+" — no artist tag"))
			continue
		}
		genres := artistCache[strings.ToLower(strings.TrimSpace(fi.artist))]
		if len(genres) == 0 {
			sum.skipped++
			fmt.Printf("  %s %s\n",
				sDimStyle.Render("∘"),
				sDimStyle.Render(fmt.Sprintf("%s — no Spotify genre for \"%s\"", base, fi.artist)),
			)
			continue
		}
		genreValue := strings.Join(genres, "; ")

		if !cfg.write {
			sum.dryCandidates++
			fileStyle := lipgloss.NewStyle().Bold(true)
			fmt.Printf("  %s %s %s %s\n",
				sWarnStyle.Render("◇"),
				fileStyle.Render(base),
				sLabelStyle.Render("→"),
				sGenreStyle.Render(genreValue),
			)
			continue
		}

		if undoSession != nil {
			if err := undoSession.Backup(fi.path); err != nil {
				sum.failed++
				fmt.Printf("  %s %s\n",
					sErrorStyle.Render("✗"),
					sErrorStyle.Render(base+" — backup failed: "+err.Error()),
				)
				continue
			}
		}

		if ok := writeGenre(fi.path, genreValue); ok {
			sum.updated++
			fileStyle := lipgloss.NewStyle().Bold(true)
			fmt.Printf("  %s %s %s %s\n",
				sSuccessStyle.Render("✓"),
				fileStyle.Render(base),
				sLabelStyle.Render("→"),
				sGenreStyle.Render(genreValue),
			)
		} else {
			sum.failed++
			fmt.Printf("  %s %s\n",
				sErrorStyle.Render("✗"),
				sErrorStyle.Render(base+" — write failed"),
			)
		}
	}

	fmt.Println()
	fmt.Println("  " + renderHR(50))
	fmt.Println("  " + sHeaderStyle.Render("Summary"))
	fmt.Println()

	if !cfg.write {
		fmt.Printf("  %-28s %s\n",
			sLabelStyle.Render("Dry-run candidates"),
			sCountStyle.Render(fmt.Sprintf("%d", sum.dryCandidates)),
		)
	} else {
		fmt.Printf("  %-28s %s\n",
			sLabelStyle.Render("Files updated"),
			sCountStyle.Render(fmt.Sprintf("%d", sum.updated)),
		)
	}
	if sum.skipped > 0 {
		fmt.Printf("  %-28s %s\n",
			sLabelStyle.Render("Skipped, no Spotify genre"),
			sDimStyle.Render(fmt.Sprintf("%d", sum.skipped)),
		)
	}
	if sum.stillMissing > 0 {
		fmt.Printf("  %-28s %s\n",
			sLabelStyle.Render("Skipped, no artist tag"),
			sDimStyle.Render(fmt.Sprintf("%d", sum.StillMissing)),
		)
	}
	if sum.failed > 0 {
		fmt.Printf("  %-28s %s\n",
			sLabelStyle.Render("Failed"),
			sErrorStyle.Render(fmt.Sprintf("%d", sum.failed)),
		)
	}
	if !cfg.write && sum.dryCandidates > 0 {
		fmt.Println()
		fmt.Println("  " + renderSep(50))
		fmt.Printf("  %s %s %s\n",
			sWarnStyle.Render("💡"),
			sWarnStyle.Render("Re-run with"),
			sCountStyle.Render("--write"),
		)
		fmt.Printf("  %s\n", sWarnStyle.Render("   to apply these changes."))
	}

	if sum.failed > 0 {
		return sum, fmt.Errorf("%d write failures", sum.failed)
	}
	return sum, nil
}

func processPathInteractive(path string, s savedSettings) bool {
	prevWrite := cfg.write
	prevMax := cfg.maxGenres
	prevDelay := cfg.delayMs
	cfg.write = s.Write
	cfg.maxGenres = s.MaxGenres
	cfg.delayMs = s.DelayMs
	defer func() {
		cfg.write = prevWrite
		cfg.maxGenres = prevMax
		cfg.delayMs = prevDelay
	}()

	if err := validateTarget(path); err != nil {
		clearScreen()
		fmt.Println(renderPageHeader("", "󰄉", "E R R O R", "Validation failed"))
		fmt.Println()
		fmt.Println("   " + sErrorStyle.Render("󰅏 "+err.Error()))
		return false
	}

	_, err := processTarget(path)

	if !cfg.write {
		fmt.Println()
		fmt.Print(sPromptStyle.Render("❯ Write these changes now? [y/N]:"))
		fmt.Print(" ")
		answer := strings.ToLower(readLine())
		if answer == "y" || answer == "yes" {
			clearScreen()
			cfg.write = true
			_, err = processTarget(path)
		}
	}

	if err != nil {
		fmt.Println()
		fmt.Println("   " + sErrorStyle.Render("✖ "+err.Error()))
	}

	if cfg.write {
		fmt.Println()
		fmt.Print(sPromptStyle.Render("❯ Open in Mp3tag? [y/N]:"))
		fmt.Print(" ")
		answer := strings.ToLower(readLine())
		if answer == "y" || answer == "yes" {
			tagPath := path
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				tagPath = filepath.Dir(path)
			}
			exec.Command("open", "-a", "Mp3tag", tagPath).Start()
		}
	}

	promptReturn()
	return true
}

// ─── Interactive Loop ─────────────────────────────────────────

func interactiveLoop() {
	s := loadSettings()
	lastTarget := ""

	for {
		clearScreen()
		renderMainMenu(s)
		choice := readInput("Choice")

		switch choice {
		case "1":
			path := readPathInputScreen()
			if path != "" {
				lastTarget = path
				processPathInteractive(path, s)
			}

		case "2":
			path := readClipboardScreen()
			if path != "" {
				lastTarget = path
				processPathInteractive(path, s)
			}

		case "p", "P":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			if !globalSettings.EnableSwinsian {
				fmt.Println()
				fmt.Println("  " + sWarnStyle.Render("Swinsian integration is disabled") + "  " + sMutedStyle.Render("Enable it in Settings."))
				promptReturn()
				continue
			}
			path, err := genrenorm.CurrentAlbumDir()
			if err != nil {
				fmt.Println()
				fmt.Println("  " + sErrorStyle.Render("✖ "+err.Error()))
				promptReturn()
				continue
			}
			lastTarget = path
			processPathInteractive(path, s)

		case "e", "E":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			if !globalSettings.EnableSwinsian {
				fmt.Println()
				fmt.Println("  " + sWarnStyle.Render("Swinsian integration is disabled") + "  " + sMutedStyle.Render("Enable it in Settings."))
				promptReturn()
				continue
			}
			paths, err := genrenorm.SelectedTrackPaths()
			if err != nil {
				fmt.Println()
				fmt.Println("  " + sErrorStyle.Render("✖ "+err.Error()))
				promptReturn()
				continue
			}
			lastTarget = paths[0]
			processPathsInteractive(paths, s)

		case "3":
			clearScreen()
			fmt.Println(renderPageHeader("", "󰀶", "F I N D E R", "Select a folder with the macOS folder picker"))
			fmt.Println()
			fmt.Print(renderSection("󰄉", "Status"))
			fmt.Println("   " + sMutedStyle.Render("󰪵 Opening Finder dialog..."))
			path := openFinderDialog()
			if path == "" {
				fmt.Println()
				fmt.Println("   " + sWarnStyle.Render("󰜺 Cancelled") + "  " + sMutedStyle.Render("Finder dialog was cancelled."))
				promptReturn()
				continue
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Println()
				fmt.Println("   " + sWarnStyle.Render("󰅏 Path does not exist"))
				fmt.Println("      " + sPathStyle.Render(path))
				promptReturn()
				continue
			}
			lastTarget = path
			processPathInteractive(path, s)

		case "4":
			path := runFZFScreen()
			if path != "" {
				lastTarget = path
				processPathInteractive(path, s)
			}

		case "5":
			if lastTarget == "" {
				fmt.Println()
				fmt.Println("  " + sInfoStyle.Render("No previous target") + "  " + sMutedStyle.Render("Run Path, Clipboard, Finder, fzf, or Downloads first."))
				promptReturn()
				continue
			}
			processPathInteractive(lastTarget, s)

		case "6":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			downloadsPath := globalSettings.DownloadsPath
			if downloadsPath == "" {
				downloadsPath = "/Volumes/Eksternal/Music/Downloads"
			}
			lastTarget = downloadsPath
			processPathInteractive(downloadsPath, s)

		case "u", "U":
			undoMenu()

		case "s", "S":
			s = runSettingsMenu()

		case "q", "Q":
			return
		}
	}
}

// processPathsInteractive processes multiple paths by processing each one
func processPathsInteractive(paths []string, s savedSettings) bool {
	for i, path := range paths {
		if i > 0 {
			fmt.Println()
			fmt.Println("   " + sMutedStyle.Render(fmt.Sprintf("Processing track %d of %d...", i+1, len(paths))))
			fmt.Println()
		}
		processPathInteractive(path, s)
	}
	return true
}
func undoMenu() {
	for {
		clearScreen()
		fmt.Println(renderPageHeader("U N D O", "󰕌", "M A N A G E", "Restore or clean Spotify backup sessions"))
		fmt.Println()
		fmt.Println("  " + renderMenuItem("1", "󰌋", "Undo Last", "Restore latest Spotify write"))
		fmt.Println("  " + renderMenuItem("2", "󰏃", "Backups", "List or clean old undo sessions"))
		fmt.Println("  " + renderMenuItem("b", "", "Back", "Return to main menu"))
		fmt.Println()
		fmt.Print(sPromptStyle.Render("❯ Choice:"))
		fmt.Print(" ")
		switch strings.ToLower(readLine()) {
		case "1":
			clearScreen()
			fmt.Println(renderPageHeader("U N D O", "󰕌", "L A S T", "Restore files from the latest Spotify write"))
			fmt.Println()
			count, err := genrenorm.RestoreLatestUndo("spotify")
			if err != nil {
				fmt.Println("   " + sErrorStyle.Render("󰅏 "+err.Error()))
			} else {
				fmt.Printf("   %s %s\n", sSuccessStyle.Render("◆"), sSuccessStyle.Render(fmt.Sprintf("Restored %d file(s)", count)))
			}
			promptReturn()
		case "2":
			manageBackups("spotify")
		case "b", "q":
			return
		}
	}
}

func manageBackups(tool string) {
	sessions, err := genrenorm.ListUndoSessions(tool)
	if err != nil {
		clearScreen()
		fmt.Println(renderPageHeader("B A C K U P S", "󰏃", "E R R O R", "Failed to list backup sessions"))
		fmt.Println()
		fmt.Println("   " + sErrorStyle.Render("󰅏 "+err.Error()))
		promptReturn()
		return
	}
	for {
		clearScreen()
		fmt.Println(renderPageHeader("B A C K U P S", "󰏃", strings.ToUpper(tool), "Manage undo backup sessions"))
		fmt.Println()
		if len(sessions) == 0 {
			fmt.Println("   " + sMutedStyle.Render("No backup sessions found."))
		} else {
			fmt.Println("  " + renderSection("󰏃", "Sessions"))
			totalSize := int64(0)
			for i, s := range sessions {
				totalSize += s.SizeBytes
				fmt.Printf("   %s  %s  %s  %s  %s\n",
					sKeyStyle.Render(fmt.Sprintf("%d", i+1)),
					sRowIconStyle.Render("󰏋"),
					sActionLabelStyle.Render(s.Timestamp.Format("2006-01-02 15:04:05")),
					sCountStyle.Render(fmt.Sprintf("%d files", s.FileCount)),
					sPathStyle.Render(genrenorm.FormatBytes(s.SizeBytes)),
				)
			}
			fmt.Println()
			fmt.Printf("   %s  %s\n", sActionLabelStyle.Render("Total:"), sCountStyle.Render(fmt.Sprintf("%d sessions, %s", len(sessions), genrenorm.FormatBytes(totalSize))))
		}
		fmt.Println()
		fmt.Println("  " + sMutedStyle.Render("c clean old sessions  b back"))
		fmt.Println()
		fmt.Print(sPromptStyle.Render("❯ Choice:"))
		fmt.Print(" ")
		switch strings.ToLower(readLine()) {
		case "c":
			cleanBackups(tool, sessions)
		case "b", "q":
			return
		}
	}
}

func cleanBackups(tool string, sessions []genrenorm.UndoSessionInfo) {
	if len(sessions) == 0 {
		return
	}
	clearScreen()
	fmt.Println(renderPageHeader("B A C K U P S", "󰏃", "C L E A N", "Remove old backup sessions"))
	fmt.Println()
	fmt.Println("  " + renderSection("󰏃", "Keep"))
	fmt.Println("   " + sActionLabelStyle.Render("The most recent session is always kept."))
	fmt.Println()
	fmt.Print(sPromptStyle.Render("❯ Sessions to keep (default 3):"))
	fmt.Print(" ")
	input := readLine()
	keep := 3
	if input != "" {
		if k, err := strconv.Atoi(input); err == nil && k > 0 {
			keep = k
		}
	}
	if keep >= len(sessions) {
		fmt.Println()
		fmt.Println("   " + sMutedStyle.Render("Nothing to clean."))
		promptReturn()
		return
	}
	fmt.Println()
	fmt.Printf("   %s  %s\n", sWarnStyle.Render("⚠"), sWarnStyle.Render(fmt.Sprintf("This will remove %d old session(s).", len(sessions)-keep)))
	fmt.Print(sPromptStyle.Render("❯ Confirm? [y/N]:"))
	fmt.Print(" ")
	if !strings.EqualFold(readLine(), "y") {
		return
	}
	removed, err := genrenorm.CleanOldUndoSessions(tool, keep)
	if err != nil {
		fmt.Println()
		fmt.Println("   " + sErrorStyle.Render("󰅏 "+err.Error()))
	} else {
		fmt.Println()
		fmt.Printf("   %s %s\n", sSuccessStyle.Render("◆"), sSuccessStyle.Render(fmt.Sprintf("Removed %d session(s).", removed)))
	}
	promptReturn()
}

// ─── Main ───────────────────────────────────────────────────────

func main() {
	defer func() {
		if r := recover(); r != nil {
			clearScreen()
			fmt.Println(renderPageHeader("E R R O R", "󰅏", "P A N I C", "An unexpected error occurred"))
			fmt.Println()
			fmt.Printf("   %s %v\n", sErrorStyle.Render("✖"), r)
			fmt.Println()
			fmt.Println("  " + sMutedStyle.Render("Press Enter to exit..."))
			readLine()
		}
	}()

	flag.BoolVar(&cfg.write, "write", false, "Actually update MP3 tags (default: dry-run)")
	flag.BoolVar(&cfg.undo, "undo", false, "Restore files from the latest write")
	flag.IntVar(&cfg.maxGenres, "max-genres", 3, "Number of final genres to write")
	flag.IntVar(&cfg.delayMs, "delay-ms", 500, "Delay between Spotify API requests in ms")
	flag.Parse()

	if cfg.undo {
		count, err := genrenorm.RestoreLatestUndo("spotify")
		if err != nil {
			printlnStyle(sErrorStyle, "✗ %v", err)
			os.Exit(1)
		}
		printlnStyle(sSuccessStyle, "Restored %d file(s)", count)
		return
	}

	s := loadSettings()

	isFlagPassed := func(name string) bool {
		found := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == name {
				found = true
			}
		})
		return found
	}

	if !isFlagPassed("write") {
		cfg.write = s.Write
	}
	if !isFlagPassed("max-genres") {
		cfg.maxGenres = s.MaxGenres
	}
	if !isFlagPassed("delay-ms") {
		cfg.delayMs = s.DelayMs
	}

	if flag.NArg() > 0 {
		cfg.target = flag.Arg(0)
		if err := validateTarget(cfg.target); err != nil {
			printlnStyle(sErrorStyle, "✗ Error: %v", err)
			os.Exit(1)
		}
		if _, err := processTarget(cfg.target); err != nil {
			printlnStyle(sErrorStyle, "✗ %v", err)
			os.Exit(1)
		}
		return
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		input, _ := reader.ReadString('\n')
		path := strings.TrimSpace(input)
		if path != "" {
			cfg.target = path
			if err := validateTarget(cfg.target); err != nil {
				printlnStyle(sErrorStyle, "✗ Error: %v", err)
				os.Exit(1)
			}
			if _, err := processTarget(cfg.target); err != nil {
				printlnStyle(sErrorStyle, "✗ %v", err)
				os.Exit(1)
			}
			return
		}
	}

	interactiveLoop()
}
