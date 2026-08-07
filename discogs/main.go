package main

import (
	"bufio"
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

// ─── Lip Gloss Styles (Discogs Teal/Blue) ───────────────────────

var (
	dHeaderStyle      lipgloss.Style
	dLabelStyle       lipgloss.Style
	dSuccessStyle     lipgloss.Style
	dErrorStyle       lipgloss.Style
	dWarnStyle        lipgloss.Style
	dArtistStyle      lipgloss.Style
	dAlbumStyle       lipgloss.Style
	dGenreStyle       lipgloss.Style
	dDimStyle         lipgloss.Style
	dCountStyle       lipgloss.Style
	dSeparatorStyle   lipgloss.Style
	dBorderStyle      lipgloss.Style
	dTitleStyle       lipgloss.Style
	dTitleIconStyle   lipgloss.Style
	dDescStyle        lipgloss.Style
	dSectionStyle     lipgloss.Style
	dSectionIconStyle lipgloss.Style
	dKeyStyle         lipgloss.Style
	dActionLabelStyle lipgloss.Style
	dMenuDescStyle    lipgloss.Style
	dSepStyle         lipgloss.Style
	dPromptStyle      lipgloss.Style
	dMutedStyle       lipgloss.Style
	dValueStyle       lipgloss.Style
	dPathStyle        lipgloss.Style
	dModeStyle        lipgloss.Style
	dWriteModeStyle   lipgloss.Style
	dInfoStyle        lipgloss.Style
	dArrowStyle       lipgloss.Style
	dFileStyle        lipgloss.Style
	dStatusLabelStyle lipgloss.Style
	dRowIconStyle     lipgloss.Style
)

func initDiscogsStyles() {
	ld := func(dark, light lipgloss.Color) lipgloss.Color {
		if lipgloss.HasDarkBackground() {
			return dark
		}
		return light
	}

	// Discogs teal/blue palette
	teal := lipgloss.Color("#2eb8b8")
	blue := lipgloss.Color("#1a9eff")
	darkTeal := lipgloss.Color("#1a6666")
	darkBg := lipgloss.Color("#1a1a2e")
	lightText := lipgloss.Color("#e0f0ff")
	mutedText := lipgloss.Color("#7ab8b8")
	medGray := lipgloss.Color("#88a0a0")
	darkGray := lipgloss.Color("#4a5a5a")

	primary := ld(teal, darkTeal)
	accent := ld(blue, lipgloss.Color("#0055aa"))
	borderC := ld(darkGray, medGray)
	sepC := ld(darkGray, medGray)

	dHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	dLabelStyle = lipgloss.NewStyle().Foreground(medGray)
	dSuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	dErrorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e74c3c"))
	dWarnStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e6a817"))
	dArtistStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(lightText, darkBg))
	dAlbumStyle = lipgloss.NewStyle().Foreground(accent)
	dGenreStyle = lipgloss.NewStyle().Foreground(teal)
	dDimStyle = lipgloss.NewStyle().Foreground(darkGray)
	dCountStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(lightText, darkBg))
	dSeparatorStyle = lipgloss.NewStyle().Foreground(sepC)

	dBorderStyle = lipgloss.NewStyle().Foreground(borderC)
	dTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(primary)
	dTitleIconStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	dDescStyle = lipgloss.NewStyle().Foreground(medGray)
	dSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(lightText, darkBg))
	dSectionIconStyle = lipgloss.NewStyle().Foreground(teal)
	dKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	dRowIconStyle = lipgloss.NewStyle().Foreground(medGray)
	dActionLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(lightText, darkBg))
	dMenuDescStyle = lipgloss.NewStyle().Foreground(medGray)
	dSepStyle = lipgloss.NewStyle().Foreground(sepC)
	dPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	dMutedStyle = lipgloss.NewStyle().Foreground(mutedText)
	dValueStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(lightText, darkBg))
	dPathStyle = lipgloss.NewStyle().Foreground(mutedText)
	dModeStyle = lipgloss.NewStyle().Bold(true).Foreground(medGray)
	dWriteModeStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	dInfoStyle = lipgloss.NewStyle().Foreground(medGray)
	dArrowStyle = lipgloss.NewStyle().Foreground(medGray)
	dFileStyle = lipgloss.NewStyle().Foreground(ld(lightText, darkBg))
	dStatusLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(medGray)
}

func init() {
	initDiscogsStyles()
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
	fmt.Print(dPromptStyle.Render("❯ " + prompt + ":"))
	fmt.Print(" ")
	return readLine()
}

func promptReturn() {
	fmt.Println()
	fmt.Println("  " + dMutedStyle.Render("Press Enter to return..."))
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
	return dSepStyle.Render(strings.Repeat("─", width))
}

func renderHR(width int) string {
	return dSepStyle.Render(strings.Repeat("═", width))
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
	return dTitleStyle.Render(left) + "  " + dTitleIconStyle.Render(icon) + "  " + dTitleStyle.Render(right)
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
	title := strings.Repeat(" ", leftPad) + dTitleStyle.Render(titleLeft) + "  " + dTitleIconStyle.Render(icon) + "  " + dTitleStyle.Render(titleRight) + strings.Repeat(" ", rightPad)

	return strings.Join([]string{
		dBorderStyle.Render("               ╭────────────────────────────────╮"),
		dBorderStyle.Render("╭──────────────┤") + title + dBorderStyle.Render("├─────────────╮"),
		dBorderStyle.Render("│              ╰────────────────────────────────╯             │"),
		dBorderStyle.Render("│") + dDescStyle.Render(centerText(desc, appWidth-2)) + dBorderStyle.Render("│"),
		dBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSection(icon, title string) string {
	sep := dSepStyle.Render(strings.Repeat("─", 56))
	return fmt.Sprintf("  %s %s\n  %s\n", dSectionIconStyle.Render(icon), dSectionStyle.Render(title), sep)
}

func renderMenuItem(key, icon, label, desc string) string {
	return fmt.Sprintf("   %s  %s  %s%s",
		dKeyStyle.Render(key),
		dRowIconStyle.Render(icon),
		dActionLabelStyle.Render(padRight(label, 22)),
		dMenuDescStyle.Render(desc))
}

func renderSettingItem(key, label, value string) string {
	return fmt.Sprintf("   %s  %s%s",
		dKeyStyle.Render(key),
		dActionLabelStyle.Render(padRight(label, 22)),
		dValueStyle.Render(value))
}

func renderMainHeader() string {
	title := renderTitle("D I S C O G S", "󰋙", "G E N R E S")
	return strings.Join([]string{
		dBorderStyle.Render("               ╭───────────────────────────────╮"),
		dBorderStyle.Render("╭──────────────┤ ") + title + dBorderStyle.Render(" ├──────────────╮"),
		dBorderStyle.Render("│              ╰───────────────────────────────╯              │"),
		dBorderStyle.Render("│") + dDescStyle.Render(centerText("Fetch album genres from the Discogs API", appWidth-2)) + dBorderStyle.Render("│"),
		dBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSettingsHeader() string {
	title := strings.Repeat(" ", 5) + dTitleIconStyle.Render("") + "  " + dTitleStyle.Render("S E T T I N G S") + strings.Repeat(" ", 7)
	return strings.Join([]string{
		dBorderStyle.Render("               ╭───────────────────────────────╮"),
		dBorderStyle.Render("╭──────────────┤ ") + title + dBorderStyle.Render("├──────────────╮"),
		dBorderStyle.Render("│              ╰───────────────────────────────╯              │"),
		dBorderStyle.Render("│") + dDescStyle.Render(centerText("Configure default Discogs scraping behavior", appWidth-2)) + dBorderStyle.Render("│"),
		dBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderStatusLine(s savedSettings) string {
	modeLabel := "Dry"
	modeStyler := dModeStyle
	if s.Write {
		modeLabel = "Write"
		modeStyler = dWriteModeStyle
	}
	return fmt.Sprintf("        %s %s: %s  %s  %s %s: %s  %s  %s %s: %s",
		dSectionIconStyle.Render(""),
		dStatusLabelStyle.Render("Genres"),
		dValueStyle.Render(fmt.Sprintf("%d", s.MaxGenres)),
		dDimStyle.Render("•"),
		dSectionIconStyle.Render(""),
		dStatusLabelStyle.Render("Mode"),
		modeStyler.Render(modeLabel),
		dDimStyle.Render("•"),
		dSectionIconStyle.Render("󱦞"),
		dStatusLabelStyle.Render("Delay"),
		dValueStyle.Render(fmt.Sprintf("%dms", s.DelayMs)),
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
	fmt.Print(renderSection("", "Actions"))
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
		modeStyler = dWriteModeStyle
	} else {
		modeStr = "dry-run"
		modeStyler = dModeStyle
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
	fmt.Print(renderSection("", "Actions"))
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
	DelayMs:   600,
}

var reader = bufio.NewReader(os.Stdin)

// ─── Settings Persistence ─────────────────────────────────────

func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".config", "genres")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "discogs.json")
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
	fmt.Println("                                    " + dMenuDescStyle.Render("b back • q quit • enter confirm"))
	fmt.Print(dPromptStyle.Render("❯ Path:"))
	fmt.Print(" ")
	input := readLine()
	if input == "" || strings.EqualFold(input, "b") || strings.EqualFold(input, "q") {
		return ""
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		fmt.Println("   " + dErrorStyle.Render("󰅏 Invalid path"))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + dWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + dPathStyle.Render(abs))
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
	fmt.Println("   " + dMutedStyle.Render("󱘟 Reading clipboard..."))
	fmt.Println()

	content := readClipboard()
	if content == "" {
		fmt.Println("   " + dWarnStyle.Render("󰅏 Empty: Copy a file or folder path first."))
		promptReturn()
		return ""
	}
	abs, err := filepath.Abs(content)
	if err != nil {
		fmt.Println("   " + dWarnStyle.Render("󰅏 Invalid path: "+content))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + dWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + dPathStyle.Render(abs))
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
	fmt.Println("   " + dMutedStyle.Render("") + " " + dStatusLabelStyle.Render("Mode") + ": " + dMutedStyle.Render("directory search with find"))
	fmt.Println()

	path, err := runFZFSearch()
	if err != nil {
		fmt.Println("   " + dWarnStyle.Render("󰜺 Cancelled") + "  " + dMutedStyle.Render(err.Error()))
		promptReturn()
		return ""
	}
	if path == "" {
		promptReturn()
		return ""
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("   " + dWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + dPathStyle.Render(path))
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
			delays := []int{0, 100, 250, 600, 1000, 1500, 2000, 3000}
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
			fmt.Println(renderPageHeader("C L E A R", "", "B A C K U P S", "Remove all backup files for Discogs"))
			fmt.Println()
			fmt.Print(renderSection("", "Warning"))
			fmt.Println("   " + dWarnStyle.Render("This will permanently delete all backup files."))
			fmt.Println("   " + dMutedStyle.Render("This action cannot be undone."))
			fmt.Println()
			fmt.Print(dPromptStyle.Render(" Are you sure? [y/N]:"))
			fmt.Print(" ")
			answer := strings.ToLower(readLine())
			if answer == "y" || answer == "yes" {
				removed, err := genrenorm.ClearAllBackups("discogs")
				if err != nil {
					fmt.Println("   " + dErrorStyle.Render("✖ Error clearing backups: "+err.Error()))
				} else {
					fmt.Println("   " + dSuccessStyle.Render(fmt.Sprintf("◆ Cleared %d backup session(s)", removed)))
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

// ─── Discogs API ────────────────────────────────────────────────

func loadEnvVar(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".env"))
	if err != nil {
		return ""
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
	return ""
}

var discogsToken string

func getDiscogsToken() string {
	if discogsToken == "" {
		discogsToken = loadEnvVar("DISCOGS_API_TOKEN")
	}
	return discogsToken
}

type discogsSearchResult struct {
	Results []struct {
		ID    int      `json:"id"`
		Title string   `json:"title"`
		Type  string   `json:"type"`
		Year  string   `json:"year"`
		Style []string `json:"style"`
		Genre []string `json:"genre"`
	} `json:"results"`
}

type discogsMaster struct {
	ID     int      `json:"id"`
	Title  string   `json:"title"`
	Year   int      `json:"year"`
	Styles []string `json:"styles"`
	Genres []string `json:"genres"`
}

func discogsGet(urlStr string) ([]byte, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MP3GenreUpdater/1.0")
	req.Header.Set("Authorization", "Discogs token="+getDiscogsToken())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		time.Sleep(2 * time.Second)
		return discogsGet(urlStr)
	}

	return io.ReadAll(resp.Body)
}

func cleanAlbumName(album string) string {
	// Strip parenthetical edition/reissue qualifiers for Discogs search
	// e.g. "Load (Remastered)" → "Load"
	editions := []string{
		"(Remastered)", "(Remaster)",
		"(Reissue)", "(Reissued)",
		"(Deluxe Edition)", "(Expanded Edition)",
		"(Bonus Track Version)", "(Bonus Tracks)",
		"(Limited Edition)", "(Special Edition)",
		"(Collector's Edition)", "(Collectors Edition)",
		"(LP)", "(CD)", "(Vinyl)",
	}
	cleaned := strings.TrimSpace(album)
	for _, suffix := range editions {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	// Also strip trailing parenthetical without a specific match
	// e.g. "Something (2023 Remaster)" → "Something"
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

func searchMasterRelease(artist, album string) *discogsSearchResult {
	cleanedAlbum := cleanAlbumName(album)
	query := url.QueryEscape(artist + " " + cleanedAlbum)
	urlStr := "https://api.discogs.com/database/search?q=" + query + "&type=master&token=" + getDiscogsToken()

	raw, err := discogsGet(urlStr)
	if err != nil {
		return nil
	}

	var result discogsSearchResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}

	if len(result.Results) == 0 {
		return nil
	}
	return &result
}

func getMasterRelease(masterID int) *discogsMaster {
	urlStr := fmt.Sprintf("https://api.discogs.com/masters/%d?token=%s", masterID, getDiscogsToken())

	raw, err := discogsGet(urlStr)
	if err != nil {
		return nil
	}

	var master discogsMaster
	if err := json.Unmarshal(raw, &master); err != nil {
		return nil
	}
	return &master
}

type discogsAlbumData struct {
	Genres []string
	Year   int
}

func fetchDiscogsAlbumGenres(artist, album string) *discogsAlbumData {
	searchResult := searchMasterRelease(artist, album)
	if searchResult == nil || len(searchResult.Results) == 0 {
		return nil
	}

	best := searchResult.Results[0]
	master := getMasterRelease(best.ID)
	if master == nil {
		// Use search result styles as fallback
		if len(best.Style) > 0 {
			return &discogsAlbumData{
				Genres: best.Style,
				Year:   parseYear(best.Year),
			}
		}
		return nil
	}

	return &discogsAlbumData{
		Genres: master.Styles,
		Year:   master.Year,
	}
}

func parseYear(s string) int {
	if s == "" {
		return 0
	}
	y, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return y
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

func readTags(file string) (artist, album, year string) {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return "", "", ""
	}
	defer tag.Close()
	return strings.TrimSpace(tag.Artist()),
		strings.TrimSpace(tag.Album()),
		strings.TrimSpace(tag.Genre()) // Note: id3v2 stores year differently; read from custom frame
}

func readMP3Year(file string) string {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return ""
	}
	defer tag.Close()

	// Try TYER (v2.3) frame first
	if frame := tag.GetFrames("TYER"); len(frame) > 0 {
		if tf, ok := frame[0].(*id3v2.TextFrame); ok && tf.Text != "" {
			return strings.TrimSpace(tf.Text)
		}
	}

	// Try TDRC (v2.4) frame
	if frame := tag.GetFrames("TDRC"); len(frame) > 0 {
		if tf, ok := frame[0].(*id3v2.TextFrame); ok && tf.Text != "" {
			// TDRC may be YYYY-MM-DD; take just the year
			return strings.TrimSpace(strings.Split(tf.Text, "-")[0])
		}
	}

	return ""
}

func writeGenreAndYear(file, genre, year string) bool {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return false
	}
	defer tag.Close()

	tag.SetGenre(genre)

	// Write year as text frame TYER (v2.3)
	if year != "" {
		tag.AddTextFrame("TYER", tag.DefaultEncoding(), year)
		// Also write TDRC for v2.4 compatibility
		tag.AddTextFrame("TDRC", tag.DefaultEncoding(), year)
	}

	if err := tag.Save(); err != nil {
		return false
	}
	return true
}

// ─── Genre Normalization (Discogs-specific combos) ──────────────

// normalizeDiscogsStyles converts Discogs style strings into cleaned genre strings.
// This mirrors the Discogs JS logic but outputs values ready for genrenorm.ExpandGenres().
func normalizeDiscogsStyles(styles []string, maxGenres int) []string {
	var all []string
	for _, s := range styles {
		// Some styles are comma-separated
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				all = append(all, part)
			}
		}
	}

	if len(all) == 0 {
		return nil
	}

	// Map abbreviations to full genre names (Discogs uses concise labels)
	expansions := map[string]string{
		"thrash":            "Thrash Metal",
		"death":             "Death Metal",
		"black":             "Black Metal",
		"doom":              "Doom Metal",
		"heavy":             "Heavy Metal",
		"speed":             "Speed Metal",
		"power":             "Power Metal",
		"progressive":       "Progressive Metal",
		"symphonic":         "Symphonic Metal",
		"folk":              "Folk Metal",
		"gothic":            "Gothic Metal",
		"groove":            "Groove Metal",
		"industrial":        "Industrial Metal",
		"nu":                "Nu Metal",
		"stoner":            "Stoner Metal",
		"sludge":            "Sludge Metal",
		"hardcore":          "Hardcore",
		"punk":              "Punk",
		"hardcore punk":     "Hardcore Punk",
		"experimental":      "Experimental",
		"avantgarde":        "Avantgarde",
		"atmospheric":       "Atmospheric",
		"melodic":           "Melodic",
		"technical":         "Technical",
		"neoclassical":      "Neoclassical",
		"post-metal":        "Post-Metal",
		"post-hardcore":     "Post-Hardcore",
		"mathcore":          "Mathcore",
		"metalcore":         "Metalcore",
		"deathcore":         "Deathcore",
		"djent":             "Djent",
		"ambient":           "Ambient",
		"drone":             "Drone",
		"noise":             "Noise",
		"avant-garde":       "Avantgarde",
		"post-rock":         "Post-Rock",
		"sludge metal":      "Sludge Metal",
		"stoner rock":       "Stoner Rock",
		"stoner metal":      "Stoner Metal",
		"atmospheric black": "Atmospheric Black Metal",
		"depressive black":  "Depressive Black Metal",
		"melodic death":     "Melodic Death Metal",
		"technical death":   "Technical Death Metal",
		"brutal death":      "Brutal Death Metal",
		"slam":              "Slam Death Metal",
		"grindcore":         "Grindcore",
		"deathgrind":        "Deathgrind",
		"goregrind":         "Goregrind",
		"pornogrind":        "Pornogrind",
		"crust":             "Crust Punk",
		"d-beat":            "D-Beat",
		"anarcho":           "Anarcho Punk",
		"oi":                "Oi!",
		"street punk":       "Street Punk",
		"skate punk":        "Skate Punk",
		"melodic hardcore":  "Melodic Hardcore",
		"metal":             "Heavy Metal",
		"rock":              "Rock",
		"hard rock":         "Hard Rock",
		"alternative":       "Alternative",
		"alternative rock":  "Alternative Rock",
		"indie":             "Indie",
		"indie rock":        "Indie Rock",
		"downtempo":         "Downtempo",
		"electronic":        "Electronic",
		"industrial metal":  "Industrial Metal",
	}

	normalized := make([]string, 0, len(all))
	for _, s := range all {
		lower := strings.ToLower(strings.TrimSpace(s))
		if expanded, ok := expansions[lower]; ok {
			normalized = append(normalized, expanded)
		} else {
			// Title-case it
			normalized = append(normalized, toMixedCase(s))
		}
	}

	// Handle Crossover: if both Thrash Metal and Hardcore exist, combine
	lowerNorm := make([]string, len(normalized))
	for i, g := range normalized {
		lowerNorm[i] = strings.ToLower(g)
	}

	hasThrash := containsOneOf(lowerNorm, "thrash metal", "thrash")
	hasHardcore := containsOneOf(lowerNorm, "hardcore", "hardcore punk")

	if hasThrash && hasHardcore {
		var filtered []string
		for _, g := range normalized {
			lower := strings.ToLower(g)
			if lower != "hardcore" && lower != "hardcore punk" {
				filtered = append(filtered, g)
			}
		}
		normalized = filtered
		normalized = append([]string{"Crossover"}, normalized...)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var result []string
	for _, g := range normalized {
		key := strings.ToLower(g)
		if !seen[key] {
			seen[key] = true
			result = append(result, g)
		}
	}

	// Now pipe through shared genre normalization
	result = genrenorm.ExpandGenres(result)

	// Apply max limit
	if len(result) > maxGenres {
		result = result[:maxGenres]
	}

	return result
}

func containsOneOf(slice []string, values ...string) bool {
	for _, s := range slice {
		for _, v := range values {
			if s == v {
				return true
			}
		}
	}
	return false
}

func toMixedCase(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return s
	}
	words := strings.Fields(lower)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
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
	year   string
}

func albumKey(artist, album string) string {
	return strings.ToLower(strings.TrimSpace(artist)) + "|||" + strings.ToLower(strings.TrimSpace(album))
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
	yearUpdated   int
}

func processTarget(target string) (*processingSummary, error) {
	modeTag := dWarnStyle.Render("[DRY RUN]")
	if cfg.write {
		modeTag = dSuccessStyle.Render("[LIVE]")
	}

	sum := &processingSummary{}
	var undoSession *genrenorm.UndoSession
	if cfg.write {
		session, err := genrenorm.StartUndoSession("discogs")
		if err != nil {
			return sum, fmt.Errorf("could not create undo backup session: %w", err)
		}
		undoSession = session
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		dHeaderStyle.Render("Genres from Discogs"),
		"  ",
		modeTag,
	)
	fmt.Println("  " + header)
	fmt.Println("  " + renderSep(50))
	fmt.Printf("  %s %s\n", dLabelStyle.Render("Target:"), dCountStyle.Render(cfg.target))
	fmt.Printf("  %s %s\n", dLabelStyle.Render("Max genres:"), dCountStyle.Render(fmt.Sprintf("%d", cfg.maxGenres)))
	if !cfg.write {
		fmt.Println("  " + dDimStyle.Render("Pass --write to apply changes."))
	}
	fmt.Println()

	files, err := findMP3Files(cfg.target)
	if err != nil {
		printlnStyle(dErrorStyle, "✗ Error: %v", err)
		return sum, err
	}
	if len(files) == 0 {
		printlnStyle(dWarnStyle, "No MP3 files found.")
		return sum, nil
	}

	fmt.Printf("  %s %s %s\n",
		dLabelStyle.Render("Found"),
		dCountStyle.Render(fmt.Sprintf("%d", len(files))),
		dLabelStyle.Render("MP3 file(s) — reading artist/album tags..."),
	)

	var fileInfos []fileInfo
	for _, f := range files {
		artist, album, _ := readTags(f)
		year := readMP3Year(f)
		fileInfos = append(fileInfos, fileInfo{path: f, artist: artist, album: album, year: year})
	}

	// Group by artist+album
	albumGroups := make(map[string][]fileInfo)
	for _, fi := range fileInfos {
		if fi.artist == "" || fi.album == "" {
			continue
		}
		key := albumKey(fi.artist, fi.album)
		albumGroups[key] = append(albumGroups[key], fi)
	}

	if len(albumGroups) == 0 {
		printlnStyle(dWarnStyle, "No files with both artist and album tags found.")
		return sum, nil
	}

	noTagsFileCount := 0
	for _, fi := range fileInfos {
		if fi.artist == "" || fi.album == "" {
			noTagsFileCount++
		}
	}

	fmt.Println()
	missingInfo := ""
	if noTagsFileCount > 0 {
		missingInfo = dDimStyle.Render(fmt.Sprintf(" (%d files missing artist or album tag)", noTagsFileCount))
	}
	fmt.Printf("  %s %s %s%s\n",
		dLabelStyle.Render("Fetching genres from Discogs for"),
		dCountStyle.Render(fmt.Sprintf("%d", len(albumGroups))),
		dLabelStyle.Render("album(s)"),
		missingInfo,
	)
	fmt.Println()

	albumCache := make(map[string]*discogsAlbumData)
	delay := time.Duration(cfg.delayMs) * time.Millisecond
	foundCount := 0

	// Sort album groups for deterministic processing
	var sortedKeys []string
	for key := range albumGroups {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for i, key := range sortedKeys {
		entries := albumGroups[key]
		if len(entries) == 0 {
			continue
		}
		artist := entries[0].artist
		album := entries[0].album

		counter := dDimStyle.Render(fmt.Sprintf("[%d/%d]", i+1, len(sortedKeys)))
		dispDisp := dArtistStyle.Render(artist) + "  " + dAlbumStyle.Render("•") + "  " + dAlbumStyle.Render(album)
		fmt.Printf("  %s %s  ", counter, dispDisp)

		tick := startSpinner()
		time.Sleep(delay)
		albumData := fetchDiscogsAlbumGenres(artist, album)
		stopSpinner(tick)

		if albumData != nil && len(albumData.Genres) > 0 {
			normalized := normalizeDiscogsStyles(albumData.Genres, cfg.maxGenres)
			albumCache[key] = &discogsAlbumData{
				Genres: normalized,
				Year:   albumData.Year,
			}
			foundCount++

			yearInfo := ""
			if albumData.Year > 0 {
				yearInfo = dDimStyle.Render(fmt.Sprintf(" [%d]", albumData.Year))
			}
			fmt.Printf("%s %s%s\n",
				dSuccessStyle.Render("✓"),
				dGenreStyle.Render(strings.Join(normalized, "; ")),
				yearInfo,
			)
		} else {
			albumCache[key] = nil
			fmt.Println(dErrorStyle.Render("✗  not found on Discogs"))
		}
	}

	fmt.Println()
	fmt.Println("  " + renderSep(50))
	fmt.Printf("  %s %s %s %s %s\n",
		dLabelStyle.Render("Found genres for"),
		dCountStyle.Render(fmt.Sprintf("%d", foundCount)),
		dLabelStyle.Render("/"),
		dCountStyle.Render(fmt.Sprintf("%d", len(sortedKeys))),
		dLabelStyle.Render("albums"),
	)
	fmt.Println()

	fmt.Println("  " + dLabelStyle.Render("Processing files..."))
	fmt.Println()

	for _, fi := range fileInfos {
		base := filepath.Base(fi.path)

		if fi.artist == "" || fi.album == "" {
			sum.stillMissing++
			fmt.Printf("  %s %s\n", dDimStyle.Render("∘"), dDimStyle.Render(base+" — missing artist or album tag"))
			continue
		}

		key := albumKey(fi.artist, fi.album)
		albumData := albumCache[key]
		if albumData == nil || len(albumData.Genres) == 0 {
			sum.skipped++
			fmt.Printf("  %s %s\n",
				dDimStyle.Render("∘"),
				dDimStyle.Render(fmt.Sprintf("%s — no Discogs genre for \"%s\"", base, fi.artist)),
			)
			continue
		}

		genreValue := strings.Join(albumData.Genres, "; ")

		// Handle year logic: only update if existing year > original year (remaster/reissue)
		yearToWrite := fi.year
		yearChanged := false
		if albumData.Year > 0 {
			existingYearNum := 0
			if fi.year != "" {
				existingYearNum, _ = strconv.Atoi(strings.TrimSpace(fi.year))
			}

			if existingYearNum > 0 && existingYearNum > albumData.Year {
				yearToWrite = strconv.Itoa(albumData.Year)
				yearChanged = true
			} else if existingYearNum == 0 {
				yearToWrite = strconv.Itoa(albumData.Year)
				yearChanged = true
			}
		}

		if !cfg.write {
			sum.dryCandidates++
			fileStyle := lipgloss.NewStyle().Bold(true)
			yearStr := ""
			if yearChanged {
				yearStr = dDimStyle.Render(fmt.Sprintf(" year: %s → %s", fi.year, yearToWrite))
			}
			fmt.Printf("  %s %s %s %s%s\n",
				dWarnStyle.Render("◇"),
				fileStyle.Render(base),
				dLabelStyle.Render("→"),
				dGenreStyle.Render(genreValue),
				yearStr,
			)
			continue
		}

		if undoSession != nil {
			if err := undoSession.Backup(fi.path); err != nil {
				sum.failed++
				fmt.Printf("  %s %s\n",
					dErrorStyle.Render("✗"),
					dErrorStyle.Render(base+" — backup failed: "+err.Error()),
				)
				continue
			}
		}

		if ok := writeGenreAndYear(fi.path, genreValue, yearToWrite); ok {
			sum.updated++
			if yearChanged {
				sum.yearUpdated++
			}
			fileStyle := lipgloss.NewStyle().Bold(true)
			yearStr := ""
			if yearChanged {
				yearStr = dDimStyle.Render(fmt.Sprintf(" year: %s → %s", fi.year, yearToWrite))
			}
			fmt.Printf("  %s %s %s %s%s\n",
				dSuccessStyle.Render("✓"),
				fileStyle.Render(base),
				dLabelStyle.Render("→"),
				dGenreStyle.Render(genreValue),
				yearStr,
			)
		} else {
			sum.failed++
			fmt.Printf("  %s %s\n",
				dErrorStyle.Render("✗"),
				dErrorStyle.Render(base+" — write failed"),
			)
		}
	}

	fmt.Println()
	fmt.Println("  " + renderHR(50))
	fmt.Println("  " + dHeaderStyle.Render("Summary"))
	fmt.Println()

	if !cfg.write {
		fmt.Printf("  %-28s %s\n",
			dLabelStyle.Render("Dry-run candidates"),
			dCountStyle.Render(fmt.Sprintf("%d", sum.dryCandidates)),
		)
	} else {
		fmt.Printf("  %-28s %s\n",
			dLabelStyle.Render("Files updated"),
			dCountStyle.Render(fmt.Sprintf("%d", sum.updated)),
		)
	}
	if sum.yearUpdated > 0 {
		fmt.Printf("  %-28s %s\n",
			dLabelStyle.Render("Years updated"),
			dCountStyle.Render(fmt.Sprintf("%d", sum.yearUpdated)),
		)
	}
	if sum.skipped > 0 {
		fmt.Printf("  %-28s %s\n",
			dLabelStyle.Render("Skipped, no Discogs style"),
			dDimStyle.Render(fmt.Sprintf("%d", sum.skipped)),
		)
	}
	if sum.stillMissing > 0 {
		fmt.Printf("  %-28s %s\n",
			dLabelStyle.Render("Skipped, no artist/album tag"),
			dDimStyle.Render(fmt.Sprintf("%d", sum.stillMissing)),
		)
	}
	if sum.failed > 0 {
		fmt.Printf("  %-28s %s\n",
			dLabelStyle.Render("Failed"),
			dErrorStyle.Render(fmt.Sprintf("%d", sum.failed)),
		)
	}
	if !cfg.write && sum.dryCandidates > 0 {
		fmt.Println()
		fmt.Println("  " + renderSep(50))
		fmt.Printf("  %s %s %s\n",
			dWarnStyle.Render("💡"),
			dWarnStyle.Render("Re-run with"),
			dCountStyle.Render("--write"),
		)
		fmt.Printf("  %s\n", dWarnStyle.Render("   to apply these changes."))
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
		fmt.Println("   " + dErrorStyle.Render("󰅏 "+err.Error()))
		return false
	}

	_, err := processTarget(path)

	if !cfg.write {
		fmt.Println()
		fmt.Print(dPromptStyle.Render("❯ Write these changes now? [y/N]:"))
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
		fmt.Println("   " + dErrorStyle.Render("✖ "+err.Error()))
	}

	if cfg.write {
		fmt.Println()
		fmt.Print(dPromptStyle.Render("❯ Open in Mp3tag? [y/N]:"))
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
				fmt.Println("  " + dWarnStyle.Render("Swinsian integration is disabled") + "  " + dMutedStyle.Render("Enable it in Settings."))
				promptReturn()
				continue
			}
			path, err := genrenorm.CurrentAlbumDir()
			if err != nil {
				fmt.Println()
				fmt.Println("  " + dErrorStyle.Render("✖ "+err.Error()))
				promptReturn()
				continue
			}
			lastTarget = path
			processPathInteractive(path, s)

		case "e", "E":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			if !globalSettings.EnableSwinsian {
				fmt.Println()
				fmt.Println("  " + dWarnStyle.Render("Swinsian integration is disabled") + "  " + dMutedStyle.Render("Enable it in Settings."))
				promptReturn()
				continue
			}
			paths, err := genrenorm.SelectedTrackPaths()
			if err != nil {
				fmt.Println()
				fmt.Println("  " + dErrorStyle.Render("✖ "+err.Error()))
				promptReturn()
				continue
			}
			// Process each selected track's album folder
			seen := make(map[string]bool)
			for _, p := range paths {
				dir := filepath.Dir(p)
				if !seen[dir] {
					seen[dir] = true
					lastTarget = dir
					processPathInteractive(dir, s)
				}
			}

		case "3":
			clearScreen()
			fmt.Println(renderPageHeader("", "󰀶", "F I N D E R", "Select a folder with the macOS folder picker"))
			fmt.Println()
			fmt.Print(renderSection("󰄉", "Status"))
			fmt.Println("   " + dMutedStyle.Render("󰪵 Opening Finder dialog..."))
			path := openFinderDialog()
			if path == "" {
				fmt.Println()
				fmt.Println("   " + dWarnStyle.Render("󰜺 Cancelled") + "  " + dMutedStyle.Render("Finder dialog was cancelled."))
				promptReturn()
				continue
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Println()
				fmt.Println("   " + dWarnStyle.Render("󰅏 Path does not exist"))
				fmt.Println("      " + dPathStyle.Render(path))
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
				fmt.Println("  " + dInfoStyle.Render("No previous target") + "  " + dMutedStyle.Render("Run Path, Clipboard, Finder, fzf, or Downloads first."))
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

func undoMenu() {
	for {
		clearScreen()
		fmt.Println(renderPageHeader("U N D O", "󰕌", "M A N A G E", "Restore or clean Discogs backup sessions"))
		fmt.Println()
		fmt.Println("  " + renderMenuItem("1", "󰌋", "Undo Last", "Restore latest Discogs write"))
		fmt.Println("  " + renderMenuItem("2", "󰏃", "Backups", "List or clean old undo sessions"))
		fmt.Println("  " + renderMenuItem("b", "", "Back", "Return to main menu"))
		fmt.Println()
		fmt.Print(dPromptStyle.Render("❯ Choice:"))
		fmt.Print(" ")
		switch strings.ToLower(readLine()) {
		case "1":
			clearScreen()
			fmt.Println(renderPageHeader("U N D O", "󰕌", "L A S T", "Restore files from the latest Discogs write"))
			fmt.Println()
			count, err := genrenorm.RestoreLatestUndo("discogs")
			if err != nil {
				fmt.Println("   " + dErrorStyle.Render("󰅏 "+err.Error()))
			} else {
				fmt.Printf("   %s %s\n", dSuccessStyle.Render("◆"), dSuccessStyle.Render(fmt.Sprintf("Restored %d file(s)", count)))
			}
			promptReturn()
		case "2":
			manageBackups("discogs")
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
		fmt.Println("   " + dErrorStyle.Render("󰅏 "+err.Error()))
		promptReturn()
		return
	}
	for {
		clearScreen()
		fmt.Println(renderPageHeader("B A C K U P S", "󰏃", strings.ToUpper(tool), "Manage undo backup sessions"))
		fmt.Println()
		if len(sessions) == 0 {
			fmt.Println("   " + dMutedStyle.Render("No backup sessions found."))
		} else {
			fmt.Println("  " + renderSection("󰏃", "Sessions"))
			totalSize := int64(0)
			for i, s := range sessions {
				totalSize += s.SizeBytes
				fmt.Printf("   %s  %s  %s  %s  %s\n",
					dKeyStyle.Render(fmt.Sprintf("%d", i+1)),
					dRowIconStyle.Render("󰏋"),
					dActionLabelStyle.Render(s.Timestamp.Format("2006-01-02 15:04:05")),
					dCountStyle.Render(fmt.Sprintf("%d files", s.FileCount)),
					dPathStyle.Render(genrenorm.FormatBytes(s.SizeBytes)),
				)
			}
			fmt.Println()
			fmt.Printf("   %s  %s\n", dActionLabelStyle.Render("Total:"), dCountStyle.Render(fmt.Sprintf("%d sessions, %s", len(sessions), genrenorm.FormatBytes(totalSize))))
		}
		fmt.Println()
		fmt.Println("  " + dMutedStyle.Render("c clean old sessions  b back"))
		fmt.Println()
		fmt.Print(dPromptStyle.Render("❯ Choice:"))
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
	fmt.Println("   " + dActionLabelStyle.Render("The most recent session is always kept."))
	fmt.Println()
	fmt.Print(dPromptStyle.Render("❯ Sessions to keep (default 3):"))
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
		fmt.Println("   " + dMutedStyle.Render("Nothing to clean."))
		promptReturn()
		return
	}
	fmt.Println()
	fmt.Printf("   %s  %s\n", dWarnStyle.Render("⚠"), dWarnStyle.Render(fmt.Sprintf("This will remove %d old session(s).", len(sessions)-keep)))
	fmt.Print(dPromptStyle.Render("❯ Confirm? [y/N]:"))
	fmt.Print(" ")
	if !strings.EqualFold(readLine(), "y") {
		return
	}
	removed, err := genrenorm.CleanOldUndoSessions(tool, keep)
	if err != nil {
		fmt.Println()
		fmt.Println("   " + dErrorStyle.Render("󰅏 "+err.Error()))
	} else {
		fmt.Println()
		fmt.Printf("   %s %s\n", dSuccessStyle.Render("◆"), dSuccessStyle.Render(fmt.Sprintf("Removed %d session(s).", removed)))
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
			fmt.Printf("   %s %v\n", dErrorStyle.Render("✖"), r)
			fmt.Println()
			fmt.Println("  " + dMutedStyle.Render("Press Enter to exit..."))
			readLine()
		}
	}()

	flag.BoolVar(&cfg.write, "write", false, "Actually update MP3 tags (default: dry-run)")
	flag.BoolVar(&cfg.undo, "undo", false, "Restore files from the latest write")
	flag.IntVar(&cfg.maxGenres, "max-genres", 3, "Number of final genres to write")
	flag.IntVar(&cfg.delayMs, "delay-ms", 600, "Delay between Discogs API requests in ms")
	flag.Parse()

	if cfg.undo {
		count, err := genrenorm.RestoreLatestUndo("discogs")
		if err != nil {
			printlnStyle(dErrorStyle, "✗ %v", err)
			os.Exit(1)
		}
		printlnStyle(dSuccessStyle, "Restored %d file(s)", count)
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
			printlnStyle(dErrorStyle, "✗ Error: %v", err)
			os.Exit(1)
		}
		if _, err := processTarget(cfg.target); err != nil {
			printlnStyle(dErrorStyle, "✗ %v", err)
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
				printlnStyle(dErrorStyle, "✗ Error: %v", err)
				os.Exit(1)
			}
			if _, err := processTarget(cfg.target); err != nil {
				printlnStyle(dErrorStyle, "✗ %v", err)
				os.Exit(1)
			}
			return
		}
	}

	interactiveLoop()
}
