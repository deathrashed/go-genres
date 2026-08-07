package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	target    string
}

var cfg config

type settings struct {
	MaxGenres int    `json:"max_genres"`
	DryRun    bool   `json:"dry_run"`
	BaseDir   string `json:"base_dir"`
}

var defaultSettings = settings{
	MaxGenres: 3,
	DryRun:    true,
	BaseDir:   "/Volumes/Eksternal/Audio",
}

var ErrCancelled = errors.New("cancelled")


var lastfmAPIKey string
var (
	borderColor    lipgloss.Color
	sepColor       lipgloss.Color
	titleColor     lipgloss.Color
	titleIconColor lipgloss.Color
	sectionColor   lipgloss.Color
	sectionIcon    lipgloss.Color
	keyColor       lipgloss.Color
	actionColor    lipgloss.Color
	actionIcon     lipgloss.Color
	textColor      lipgloss.Color
	mutedColor     lipgloss.Color
	pathColor      lipgloss.Color
	modeColor      lipgloss.Color
	statusKeyColor lipgloss.Color
	warnColor      lipgloss.Color
	errorColor     lipgloss.Color
	successColor   lipgloss.Color
	promptColor    lipgloss.Color
	artistColor    lipgloss.Color
	albumColor     lipgloss.Color
	skyColor       lipgloss.Color
	peachColor     lipgloss.Color

	borderStyle      lipgloss.Style
	titleStyle       lipgloss.Style
	titleIconStyle   lipgloss.Style
	descStyle        lipgloss.Style
	sectionStyle     lipgloss.Style
	sectionIconStyle lipgloss.Style
	keyStyle         lipgloss.Style
	rowIconStyle     lipgloss.Style
	iconStyle        lipgloss.Style
	labelStyle       lipgloss.Style
	mutedStyle       lipgloss.Style
	pathStyle        lipgloss.Style
	modeStyle        lipgloss.Style
	writeModeStyle   lipgloss.Style
	promptStyle      lipgloss.Style
	warnStyle        lipgloss.Style
	errorStyle       lipgloss.Style
	successStyle     lipgloss.Style
	infoStyle        lipgloss.Style
	genreStyle       lipgloss.Style
	countStyle       lipgloss.Style
	artistStyle      lipgloss.Style
	albumStyle       lipgloss.Style
	sepStyle         lipgloss.Style
	statusLabelStyle lipgloss.Style
	actionLabelStyle lipgloss.Style
	descAccentStyle  lipgloss.Style
	menuDescStyle    lipgloss.Style
	valueStyle       lipgloss.Style
	dotStyle         lipgloss.Style
	arrowStyle       lipgloss.Style
	counterStyle     lipgloss.Style
	fileStyle        lipgloss.Style
	warnTagStyle     lipgloss.Style
	skyStyle         lipgloss.Style
)

func initTheme() {
	ld := func(dark, light lipgloss.Color) lipgloss.Color {
		if lipgloss.HasDarkBackground() {
			return dark
		}
		return light
	}

	white := lipgloss.Color("#ffffff")
	torchRed := lipgloss.Color("#f8211c")
	alizarin := lipgloss.Color("#e11e19")
	thunderbird := lipgloss.Color("#cc1b17")
	carnation := lipgloss.Color("#fa645d")
	chestnutRose := lipgloss.Color("#d0524a")
	burntSienna := lipgloss.Color("#f06059")
	oldRose := lipgloss.Color("#be7776")
	sandstone := lipgloss.Color("#736765")
	chicago := lipgloss.Color("#565554")

	borderColor = ld(thunderbird, alizarin)
	sepColor = ld(sandstone, oldRose)
	titleColor = ld(white, chicago)
	titleIconColor = ld(torchRed, alizarin)
	sectionColor = ld(carnation, thunderbird)
	sectionIcon = ld(white, alizarin)
	keyColor = ld(torchRed, alizarin)
	actionColor = ld(white, chicago)
	actionIcon = ld(carnation, thunderbird)
	textColor = ld(white, chicago)
	mutedColor = ld(sandstone, oldRose)
	pathColor = ld(burntSienna, chestnutRose)
	modeColor = ld(white, chicago)
	statusKeyColor = ld(sandstone, oldRose)
	warnColor = ld(burntSienna, chestnutRose)
	errorColor = ld(torchRed, alizarin)
	successColor = ld(white, chicago)
	promptColor = ld(torchRed, alizarin)
	artistColor = ld(carnation, thunderbird)
	albumColor = ld(white, chicago)
	skyColor = ld(burntSienna, chestnutRose)
	peachColor = ld(burntSienna, chestnutRose)

	borderStyle = lipgloss.NewStyle().Foreground(borderColor)
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(titleColor)
	titleIconStyle = lipgloss.NewStyle().Bold(true).Foreground(titleIconColor)
	descStyle = lipgloss.NewStyle().Foreground(mutedColor)
	descAccentStyle = lipgloss.NewStyle().Foreground(mutedColor)
	menuDescStyle = lipgloss.NewStyle().Foreground(mutedColor)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(sectionColor)
	sectionIconStyle = lipgloss.NewStyle().Foreground(sectionIcon)
	keyStyle = lipgloss.NewStyle().Bold(true).Foreground(keyColor)
	rowIconStyle = lipgloss.NewStyle().Foreground(actionIcon)
	iconStyle = lipgloss.NewStyle().Foreground(sectionIcon)
	labelStyle = lipgloss.NewStyle().Foreground(textColor)
	mutedStyle = lipgloss.NewStyle().Foreground(mutedColor)
	pathStyle = lipgloss.NewStyle().Foreground(pathColor)
	modeStyle = lipgloss.NewStyle().Bold(true).Foreground(modeColor)
	writeModeStyle = lipgloss.NewStyle().Bold(true).Foreground(warnColor)
	promptStyle = lipgloss.NewStyle().Bold(true).Foreground(promptColor)
	warnStyle = lipgloss.NewStyle().Foreground(warnColor)
	errorStyle = lipgloss.NewStyle().Foreground(errorColor)
	successStyle = lipgloss.NewStyle().Foreground(successColor)
	infoStyle = lipgloss.NewStyle().Foreground(skyColor)
	genreStyle = lipgloss.NewStyle().Bold(true).Foreground(white)
	countStyle = lipgloss.NewStyle().Bold(true).Foreground(white)
	valueStyle = lipgloss.NewStyle().Bold(true).Foreground(textColor)
	dotStyle = lipgloss.NewStyle().Foreground(mutedColor)
	arrowStyle = lipgloss.NewStyle().Foreground(mutedColor)
	counterStyle = lipgloss.NewStyle().Foreground(peachColor)
	fileStyle = lipgloss.NewStyle().Foreground(textColor)
	warnTagStyle = lipgloss.NewStyle().Foreground(warnColor)
	skyStyle = lipgloss.NewStyle().Bold(true).Foreground(skyColor)
	artistStyle = lipgloss.NewStyle().Bold(true).Foreground(artistColor)
	albumStyle = lipgloss.NewStyle().Bold(true).Foreground(albumColor)
	sepStyle = lipgloss.NewStyle().Foreground(sepColor)
	statusLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(statusKeyColor)
	actionLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(actionColor)
}

func init() {
	initTheme()
}

const appWidth = 63

// ─── Reader Helpers ─────────────────────────────────────────────

var reader = bufio.NewReader(os.Stdin)

func readLine() string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func readLineRaw() string {
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\n\r")
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func promptReturn() {
	fmt.Println()
	fmt.Println("  " + mutedStyle.Render("Press Enter to return..."))
	readLine()
}

func readInput(prompt string) string {
	fmt.Print(promptStyle.Render("❯ " + prompt + ":"))
	fmt.Print(" ")
	input := readLine()
	return input
}

// ─── Render Helpers ─────────────────────────────────────────────

func renderTitle(left, icon, right string, leftPad, rightPad int) string {
	return strings.Repeat(" ", leftPad) + titleStyle.Render(left) + "  " + titleIconStyle.Render(icon) + "  " + titleStyle.Render(right) + strings.Repeat(" ", rightPad)
}

func renderMainHeader() string {
	title := renderTitle("L A S T . F M", "", "G E N R E S", 0, 0)
	return strings.Join([]string{
		borderStyle.Render("               ╭───────────────────────────────╮"),
		borderStyle.Render("╭──────────────┤ ") + title + borderStyle.Render(" ├──────────────╮"),
		borderStyle.Render("│              ╰───────────────────────────────╯              │"),
		borderStyle.Render("│") + descAccentStyle.Render("      Replace MP3 genre tags using Last.fm artist tags       ") + borderStyle.Render("│"),
		borderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSettingsHeader() string {
	title := strings.Repeat(" ", 5) + titleIconStyle.Render("") + "  " + titleStyle.Render("S E T T I N G S") + strings.Repeat(" ", 7)
	return strings.Join([]string{
		borderStyle.Render("               ╭───────────────────────────────╮"),
		borderStyle.Render("╭──────────────┤ ") + title + borderStyle.Render("├──────────────╮"),
		borderStyle.Render("│              ╰───────────────────────────────╯              │"),
		borderStyle.Render("│") + descAccentStyle.Render("      Configure default Last.fm genre tagging behavior       ") + borderStyle.Render("│"),
		borderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
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
	title := renderTitle(titleLeft, icon, titleRight, leftPad, rightPad)
	return strings.Join([]string{
		borderStyle.Render("               ╭────────────────────────────────╮"),
		borderStyle.Render("╭──────────────┤") + title + borderStyle.Render("├─────────────╮"),
		borderStyle.Render("│              ╰────────────────────────────────╯             │"),
		borderStyle.Render("│") + descAccentStyle.Render(centerText(desc, appWidth-2)) + borderStyle.Render("│"),
		borderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderStatusLine(s settings) string {
	modeLabel := "Dry"
	modeStyler := modeStyle
	if !s.DryRun {
		modeLabel = "Write"
		modeStyler = writeModeStyle
	}

	return fmt.Sprintf("        %s %s: %s  %s  %s %s: %s  %s  %s %s: %s",
		sectionIconStyle.Render(""),
		statusLabelStyle.Render("Genres"),
		valueStyle.Render(fmt.Sprintf("%d", s.MaxGenres)),
		dotStyle.Render("•"),
		sectionIconStyle.Render(""),
		statusLabelStyle.Render("Mode"),
		modeStyler.Render(modeLabel),
		dotStyle.Render("•"),
		sectionIconStyle.Render(""),
		statusLabelStyle.Render("Scope"),
		pathStyle.Render(shortScope(s.BaseDir)),
	)
}

func renderStatus(s settings) string {
	return renderStatusLine(s)
}

func shortScope(s string) string {
	if strings.TrimRight(s, "/") == "/Volumes/Eksternal/Audio" {
		return "Eksternal"
	}
	if s == "" {
		return "(none)"
	}
	return s
}

func renderSection(icon, title string) string {
	sep := sepStyle.Render("────────────────────────────────────────────────────────────")
	return fmt.Sprintf("  %s %s\n  %s\n", sectionIconStyle.Render(icon), sectionStyle.Render(title), sep)
}

func renderMenuItem(key, icon, label, desc string) string {
	return fmt.Sprintf("   %s  %s  %s%s",
		keyStyle.Render(key),
		rowIconStyle.Render(icon),
		actionLabelStyle.Render(padRight(label, 22)),
		menuDescStyle.Render(desc))
}

func renderSettingItem(key, icon, label, value string, valStyler lipgloss.Style) string {
	return fmt.Sprintf("   %s  %s  %s%s",
		keyStyle.Render(key),
		rowIconStyle.Render(icon),
		actionLabelStyle.Render(padRight(label, 22)),
		valStyler.Render(value))
}

func renderKeyboardHint(hint string) string {
	return descAccentStyle.Render(hint)
}

func renderInputFooter() {
	fmt.Println()
	fmt.Println("                                    " + renderKeyboardHint("b back • q quit • enter confirm"))
}

func renderInfoLine(icon, label, value string, styler lipgloss.Style) string {
	return fmt.Sprintf("  %s %s: %s", rowIconStyle.Render(icon), statusLabelStyle.Render(label), styler.Render(value))
}

func renderErrorScreen(titleLeft, icon, titleRight, desc, message, detail string) {
	clearScreen()
	fmt.Println(renderPageHeader(titleLeft, icon, titleRight, desc))
	fmt.Println()
	fmt.Print(renderSection("󰄉", "Status"))
	fmt.Println("   " + errorStyle.Render("󰅏 "+message))
	if strings.TrimSpace(detail) != "" {
		fmt.Println("   " + mutedStyle.Render(detail))
	}
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

func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// ─── Main Menu ─────────────────────────────────────────────────

func renderMainMenu(s settings) {
	globalSettings, _ := genrenorm.LoadGlobalSettings()

	fmt.Print(renderMainHeader())
	fmt.Println()
	fmt.Println(renderStatus(s))
	fmt.Println()
	fmt.Print(renderSection("", "Input"))
	fmt.Println(renderMenuItem("1", "\uf506", "Path", "Enter file/folder paths"))
	fmt.Println(renderMenuItem("2", "\U000f1266", "Clipboard", "Use path from clipboard"))
	if globalSettings.EnableSwinsian {
		fmt.Println(renderMenuItem("p", "▶", "Playing", "Tag currently playing in Swinsian"))
		fmt.Println(renderMenuItem("e", "󰎈", "Selected", "Tag selected tracks in Swinsian"))
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

// ─── Settings Menu ──────────────────────────────────────────────

func renderSettingsMenu(s settings) {
	globalSettings, _ := genrenorm.LoadGlobalSettings()

	var modeStr string
	var modeStyler lipgloss.Style
	if s.DryRun {
		modeStr = "dry-run"
		modeStyler = modeStyle
	} else {
		modeStr = "write"
		modeStyler = writeModeStyle
	}

	fmt.Print(renderSettingsHeader())
	fmt.Println()
	fmt.Print(renderSection("󰟓", "Options"))
	fmt.Println(renderSettingItem("g", "󰲹", "Max Genres", fmt.Sprintf("%d", s.MaxGenres), labelStyle))
	fmt.Println(renderSettingItem("m", "󱃯", "Default Mode", modeStr, modeStyler))
	fmt.Println(renderSettingItem("b", "󰡦", "Base Dir", s.BaseDir, pathStyle))
	fmt.Println()
	fmt.Print(renderSection("󰗼", "Backup"))
	backupStatus := "disabled"
	if globalSettings.EnableBackup {
		backupStatus = "enabled"
	}
	fmt.Println(renderSettingItem("a", "󰄬", "Auto Backup", backupStatus, labelStyle))
	promptStatus := "off"
	if globalSettings.PromptForBackup {
		promptStatus = "on"
	}
	fmt.Println(renderSettingItem("p", "󰋧", "Prompt Before Write", promptStatus, labelStyle))
	fmt.Println(renderSettingItem("c", "󰆴", "Clear All Backups", "Remove all backup files", warnStyle))
	fmt.Println()
	fmt.Print(renderSection("", "Swinsian"))
	swinsianStatus := "disabled"
	if globalSettings.EnableSwinsian {
		swinsianStatus = "enabled"
	}
	fmt.Println(renderSettingItem("n", "▶", "Swinsian Integration", swinsianStatus, labelStyle))
	fmt.Println()
	fmt.Print(renderSection("", "Actions"))
	fmt.Println(renderMenuItem("s", "", "Save", "Write settings and return"))
	fmt.Println(renderMenuItem("x", "", "Back", "Return without saving"))
	fmt.Println()
}

func runSettingsMenu() settings {
	s := loadSettings()

	for {
		clearScreen()
		renderSettingsMenu(s)
		choice := readInput("Choice")

		switch choice {
		case "g":
			s.MaxGenres++
			if s.MaxGenres > 10 {
				s.MaxGenres = 1
			}
		case "m":
			s.DryRun = !s.DryRun
		case "b":
			newDir := readBaseDirInput(s.BaseDir)
			if newDir != "" {
				s.BaseDir = newDir
			}
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
			fmt.Println(renderPageHeader("C L E A R", "󰴀", "B A C K U P S", "Remove all backup files for Last.fm"))
			fmt.Println()
			fmt.Print(renderSection("⚠", "Warning"))
			fmt.Println("   " + warnStyle.Render("This will permanently delete all backup files."))
			fmt.Println("   " + mutedStyle.Render("This action cannot be undone."))
			fmt.Println()
			fmt.Print(promptStyle.Render(" Are you sure? [y/N]:"))
			fmt.Print(" ")
			answer := strings.ToLower(readLine())
			if answer == "y" || answer == "yes" {
				removed, err := genrenorm.ClearAllBackups("lastfm")
				if err != nil {
					fmt.Println("   " + errorStyle.Render("✖ Error clearing backups: "+err.Error()))
				} else {
					fmt.Println("   " + successStyle.Render(fmt.Sprintf("◆ Cleared %d backup session(s)", removed)))
				}
				promptReturn()
			}
		case "n":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			globalSettings.EnableSwinsian = !globalSettings.EnableSwinsian
			genrenorm.SaveGlobalSettings(globalSettings)
		case "s":
			cfg.write = !s.DryRun
			cfg.maxGenres = s.MaxGenres
			saveSettings(s)
			return s
		case "x", "q":
			return s
		}
	}
}

func readBaseDirInput(current string) string {
	clearScreen()
	fmt.Println(renderPageHeader("B A S E", "󱍚", "D I R E C T O R Y", "Change the default library search root"))
	fmt.Println()
	fmt.Println(renderInfoLine("󱍚", "Current", current, pathStyle))
	fmt.Println("  " + sepStyle.Render("────────────────────────────────────────────────────────────"))
	fmt.Println()
	fmt.Println("                                    " + renderKeyboardHint("b back • q quit • enter confirm"))
	fmt.Print(promptStyle.Render("❯ Path:"))
	fmt.Print(" ")
	input := readLineRaw()
	if input == "" || strings.EqualFold(input, "b") || strings.EqualFold(input, "q") {
		return ""
	}
	resolved := resolvePath(input, current)
	if resolved == "" {
		resolved = input
	}
	return filepath.Clean(resolved)
}

// ─── I/O Helpers ────────────────────────────────────────────────

func readPathInputScreen(baseDir string) string {
	clearScreen()
	fmt.Println(renderPageHeader("P A T H", "", "I N P U T", "Enter one or more files or folders"))
	fmt.Println()
	fmt.Println(renderInfoLine("󱍚", "Base", baseDir, pathStyle))
	fmt.Println("  " + sepStyle.Render("────────────────────────────────────────────────────────────"))
	fmt.Println()
	fmt.Println("                                    " + renderKeyboardHint("b back • q quit • enter confirm"))
	fmt.Print(promptStyle.Render("❯ Path:"))
	fmt.Print(" ")
	input := readLineRaw()
	if input == "" || strings.EqualFold(input, "b") || strings.EqualFold(input, "q") {
		return ""
	}
	paths := strings.Fields(input)
	var resolved []string
	for _, p := range paths {
		r := resolvePath(p, baseDir)
		if r != "" {
			resolved = append(resolved, r)
		}
	}
	if len(resolved) == 0 {
		renderErrorScreen("P A T H", "", "I N P U T", "Enter one or more files or folders", "No valid paths", "Could not resolve any of the provided paths.")
		promptReturn()
		return ""
	}
	abs := filepath.Clean(resolved[0])
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		renderErrorScreen("P A T H", "", "I N P U T", "Enter one or more files or folders", "Path does not exist", abs)
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

func readClipboardScreen(baseDir string) string {
	clearScreen()
	fmt.Println(renderPageHeader("", "󱉦", "C L I P B O A R D", "Read a file or folder path from clipboard"))
	fmt.Println()
	fmt.Print(renderSection("󱘟", "Status"))
	fmt.Println("   " + mutedStyle.Render("󱘟 Reading clipboard..."))
	fmt.Println()

	content := readClipboard()
	if content == "" {
		fmt.Println("   " + warnStyle.Render("󰅏 Empty: Copy a file or folder path first."))
		promptReturn()
		return ""
	}

	paths := strings.Fields(content)
	var resolved []string
	for _, p := range paths {
		r := resolvePath(p, baseDir)
		if r != "" {
			resolved = append(resolved, r)
		}
	}
	if len(resolved) == 0 {
		fmt.Println("   " + warnStyle.Render("󰅏 No valid paths: Clipboard content did not resolve."))
		promptReturn()
		return ""
	}
	return filepath.Clean(resolved[0])
}

func renderFinderScreen() {
	clearScreen()
	fmt.Println(renderPageHeader("", "󰀶", "F I N D E R", "Select a folder with the macOS folder picker"))
	fmt.Println()
	fmt.Print(renderSection("󰄉", "Status"))
	fmt.Println("   " + mutedStyle.Render("󰪵 Opening Finder dialog..."))
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

func runFZFSearch(baseDir string) (string, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = defaultSettings.BaseDir
	}

	baseDir = filepath.Clean(baseDir)

	if st, err := os.Stat(baseDir); err != nil || !st.IsDir() {
		return "", fmt.Errorf("fzf base directory not found: %s", baseDir)
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("fzf not found. Install with: brew install fzf")
	}
	if _, err := exec.LookPath("fd"); err != nil {
		return "", fmt.Errorf("fd not found. Install with: brew install fd")
	}

	script := `
set -euo pipefail
base=$1

fd --type d . "$base" 2>/dev/null |
  fzf \
    --exact \
    --prompt=' ❯ ' \
    --header='Enter:Select  Esc:Cancel  Ctrl-R:Reveal  Ctrl-Y:Copy' \
    --preview='eza --tree --color=always {} 2>/dev/null | head -80 || ls -la {} 2>/dev/null | head -80' \
    --preview-window='top:40%:wrap' \
    --bind='ctrl-r:execute(open -R {})' \
    --bind='ctrl-y:execute-silent(echo {} | pbcopy)'
`

	cmd := exec.Command("bash", "-lc", script, "fzf-albums", baseDir)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", ErrCancelled
	}

	selected := strings.TrimSpace(string(out))
	if selected == "" {
		return "", ErrCancelled
	}

	if !filepath.IsAbs(selected) {
		selected = filepath.Join(baseDir, selected)
	}

	return filepath.Clean(selected), nil
}

func runFZFScreen(baseDir string) string {
	clearScreen()
	fmt.Println(renderPageHeader("F Z F", "", "S E A R C H", "Search album folders from the library"))
	fmt.Println()
	fmt.Print(renderSection("󰪵", "Opening FZF"))
	fmt.Println("   " + renderInfoLine("󱍚", "Root", baseDir, pathStyle)[2:])
	fmt.Println("   " + rowIconStyle.Render("") + " " + statusLabelStyle.Render("Mode") + ": " + mutedStyle.Render("directory search with fd"))
	fmt.Println()

	path, err := runFZFSearch(baseDir)
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			fmt.Println("   " + warnStyle.Render("󰜺 Cancelled") + "  " + mutedStyle.Render("No folder selected."))
		} else {
			fmt.Println("   " + errorStyle.Render("󰅏 Error") + "  " + mutedStyle.Render(err.Error()))
		}
		promptReturn()
		return ""
	}
	if path == "" {
		promptReturn()
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		fmt.Println("   " + errorStyle.Render("󰅏 Error") + "  " + mutedStyle.Render(err.Error()))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + warnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + pathStyle.Render(abs))
		promptReturn()
		return ""
	}
	resetHTTPClient()
	return abs
}

// ─── Interactive Processing ─────────────────────────────────────

func processPathInteractive(path string, s settings) bool {
	clearScreen()
	resetHTTPClient()
	previousWrite := cfg.write
	cfg.write = !s.DryRun
	cfg.maxGenres = s.MaxGenres
	defer func() { cfg.write = previousWrite }()

	if !loadAPIKey() {
		renderErrorScreen("A P I", "󰌋", "K E Y", "LASTFM_API_KEY is required before processing", "LASTFM_API_KEY not found", "Set LASTFM_API_KEY in your environment or ~/.env.")
		promptReturn()
		return false
	}

	err := validateTarget(path)
	if err != nil {
		renderErrorScreen("P A T H", "󱍚", "W A R N I N G", "Validate the selected file or folder", "Invalid target", err.Error())
		promptReturn()
		return false
	}

	files, err := collectFiles([]string{path})
	if err != nil {
		renderErrorScreen("P A T H", "󱍚", "W A R N I N G", "Collect MP3 files from the selected target", "Collect failed", err.Error())
		promptReturn()
		return false
	}
	if len(files) == 0 {
		renderErrorScreen("N O", "", "M P 3", "No matching audio files were found", "No MP3 files found", path)
		promptReturn()
		return false
	}

	_, procErr := runProcessingCLI(files)

	if !cfg.write {
		fmt.Println()
		fmt.Print(promptStyle.Render("❯ Write these changes now? [y/N]:"))
		fmt.Print(" ")
		answer := strings.ToLower(readLine())
		if answer == "y" || answer == "yes" {
			clearScreen()
			cfg.write = true
			_, procErr = runProcessingCLI(files)
		}
	}

	if procErr != nil {
		fmt.Println()
		fmt.Println("   " + errorStyle.Render("✖ "+procErr.Error()))
	}

	if cfg.write {
		fmt.Println()
		fmt.Print(promptStyle.Render("❯ Open in Mp3tag? [y/N]:"))
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

func countDryCandidates(files []string) int {
	dirGroups := make(map[string][]string)
	for _, f := range files {
		d := filepath.Dir(f)
		dirGroups[d] = append(dirGroups[d], f)
	}
	var dirKeys []string
	for d := range dirGroups {
		dirKeys = append(dirKeys, d)
	}
	sort.Strings(dirKeys)

	dirArtist := make(map[string]string, len(dirKeys))
	for _, dir := range dirKeys {
		group := dirGroups[dir]
		for _, f := range group {
			if a := readArtist(f); a != "" {
				dirArtist[dir] = a
				break
			}
		}
	}

	uniqueArtists := make(map[string]string)
	for _, a := range dirArtist {
		first := firstArtist(a)
		if first == "" {
			continue
		}
		key := strings.ToLower(first)
		if _, ok := uniqueArtists[key]; ok {
			continue
		}
		uniqueArtists[key] = first
	}

	cache := make(map[string][]string)
	for _, name := range uniqueArtists {
		key := strings.ToLower(strings.TrimSpace(name))
		rawTags, err := fetchTopTagsWithError(name, cfg.maxGenres)
		if err != nil {
			continue
		}
		if len(rawTags) > 0 {
			genres := genrenorm.ExpandGenres(rawTags)
			if len(genres) > cfg.maxGenres {
				genres = genres[:cfg.maxGenres]
			}
			cache[key] = genres
		}
	}

	candidates := 0
	for _, dir := range dirKeys {
		group := dirGroups[dir]
		artistName := dirArtist[dir]
		first := firstArtist(artistName)
		if first == "" {
			continue
		}
		key := strings.ToLower(first)
		if genres, ok := cache[key]; ok && len(genres) > 0 {
			candidates += len(group)
		}
	}
	return candidates
}

// ─── Main Interactive Loop ──────────────────────────────────────

func interactiveLoop() {
	s := loadSettings()
	lastTarget := ""

	for {
		clearScreen()
		renderMainMenu(s)
		choice := readInput("Choice")

		switch choice {
		case "1":
			path := readPathInputScreen(s.BaseDir)
			if path != "" {
				lastTarget = path
				processPathInteractive(path, s)
			}

		case "2":
			path := readClipboardScreen(s.BaseDir)
			if path != "" {
				lastTarget = path
				processPathInteractive(path, s)
			}

		case "p", "P":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			if !globalSettings.EnableSwinsian {
				fmt.Println()
				fmt.Println("  " + warnStyle.Render("Swinsian integration is disabled") + "  " + mutedStyle.Render("Enable it in Settings."))
				promptReturn()
				continue
			}
			path, err := genrenorm.CurrentAlbumDir()
			if err != nil {
				fmt.Println()
				fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
				promptReturn()
				continue
			}
			lastTarget = path
			processPathInteractive(path, s)

		case "e", "E":
			globalSettings, _ := genrenorm.LoadGlobalSettings()
			if !globalSettings.EnableSwinsian {
				fmt.Println()
				fmt.Println("  " + warnStyle.Render("Swinsian integration is disabled") + "  " + mutedStyle.Render("Enable it in Settings."))
				promptReturn()
				continue
			}
			paths, err := genrenorm.SelectedTrackPaths()
			if err != nil {
				fmt.Println()
				fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
				promptReturn()
				continue
			}
			// Use first track as target for processing
			lastTarget = paths[0]
			processPathsInteractive(paths, s)

		case "3":
			renderFinderScreen()
			path := openFinderDialog()
			if path == "" {
				fmt.Println()
				fmt.Println("   " + warnStyle.Render("󰜺 Cancelled") + "  " + mutedStyle.Render("Finder dialog was cancelled."))
				promptReturn()
				continue
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Println()
				fmt.Println("   " + warnStyle.Render("󰅏 Path does not exist"))
				fmt.Println("      " + pathStyle.Render(path))
				promptReturn()
				continue
			}
			resetHTTPClient()
			lastTarget = path
			processPathInteractive(path, s)

		case "4":
			base := s.BaseDir
			if base == "" {
				base = defaultSettings.BaseDir
			}
			path := runFZFScreen(filepath.Clean(base))
			if path != "" {
				lastTarget = path
				processPathInteractive(path, s)
			}

		case "5":
			if lastTarget == "" {
				fmt.Println()
				fmt.Println("  " + infoStyle.Render("No previous target") + "  " + mutedStyle.Render("Run Path, Clipboard, Finder, fzf, or Downloads first."))
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

// processPathsInteractive processes multiple paths interactively
func processPathsInteractive(paths []string, s settings) bool {
	clearScreen()
	resetHTTPClient()
	previousWrite := cfg.write
	cfg.write = !s.DryRun
	cfg.maxGenres = s.MaxGenres
	defer func() { cfg.write = previousWrite }()

	if !loadAPIKey() {
		renderErrorScreen("A P I", "", "K E Y", "LASTFM_API_KEY is required before processing", "LASTFM_API_KEY not found", "Set LASTFM_API_KEY in your environment or ~/.env.")
		promptReturn()
		return false
	}

	var allFiles []string
	for _, path := range paths {
		err := validateTarget(path)
		if err != nil {
			renderErrorScreen("P A T H", "", "W A R N I N G", "Validate the selected file or folder", "Invalid target", err.Error())
			promptReturn()
			return false
		}

		files, err := collectFiles([]string{path})
		if err != nil {
			renderErrorScreen("P A T H", "", "W A R N I N G", "Collect MP3 files from the selected target", "Collect failed", err.Error())
			promptReturn()
			return false
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		renderErrorScreen("N O", "󰽳", "M P 3", "No matching audio files were found", "No MP3 files found", strings.Join(paths, ", "))
		promptReturn()
		return false
	}

	_, procErr := runProcessingCLI(allFiles)

	if !cfg.write {
		fmt.Println()
		fmt.Print(promptStyle.Render("❯ Write these changes now? [y/N]:"))
		fmt.Print(" ")
		answer := strings.ToLower(readLine())
		if answer == "y" || answer == "yes" {
			clearScreen()
			cfg.write = true
			_, procErr = runProcessingCLI(allFiles)
		}
	}

	if procErr != nil {
		fmt.Println()
		fmt.Println("   " + errorStyle.Render("✖ "+procErr.Error()))
	}

	if cfg.write {
		fmt.Println()
		fmt.Print(promptStyle.Render(" Open in Mp3tag? [y/N]:"))
		fmt.Print(" ")
		answer := strings.ToLower(readLine())
		if answer == "y" || answer == "yes" {
			tagPath := paths[0]
			if info, err := os.Stat(paths[0]); err == nil && !info.IsDir() {
				tagPath = filepath.Dir(paths[0])
			}
			exec.Command("open", "-a", "Mp3tag", tagPath).Start()
		}
	}

	promptReturn()
	return true
}

// ─── HTTP ───────────────────────────────────────────────────────

var httpClient = &http.Client{
	Timeout:   20 * time.Second,
	Transport: newLastFMTransport(),
}

func resetHTTPClient() {
	httpClient = &http.Client{
		Timeout:   20 * time.Second,
		Transport: newLastFMTransport(),
	}
}

func newLastFMTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp4", address)
	}
	return transport
}

func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bad file descriptor") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "network is down") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "no such file or directory") ||
		strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "operation not permitted") ||
		strings.Contains(msg, "kqueue")
}

type lastfmTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type lastfmTopTags struct {
	Tag  []lastfmTag `json:"tag"`
	Attr lastfmAttr  `json:"@attr"`
}

type lastfmAttr struct {
	Artist string `json:"artist"`
}

type lastfmResponse struct {
	TopTags *lastfmTopTags `json:"toptags"`
	Error   *int           `json:"error"`
	Message string         `json:"message"`
}

var exiftoolPath = "exiftool"
var curlPath = "curl"

func doLastFMRequest(req *http.Request) (*http.Response, error) {
	resp, err := httpClient.Do(req)
	if err == nil {
		return resp, nil
	}

	// Retry once with a fresh client and a cloned request. Terminal subprocesses
	// (fzf, Finder) can leave the default transport in a bad state on macOS.
	if isTransientNetErr(err) {
		resetHTTPClient()
		retryReq := req.Clone(req.Context())
		return httpClient.Do(retryReq)
	}

	return nil, err
}

func fetchTopTags(artist string, max int) []string {
	tags, _ := fetchTopTagsWithError(artist, max)
	return tags
}

func fetchTopTagsWithError(artist string, max int) ([]string, error) {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil, fmt.Errorf("empty artist")
	}
	if strings.TrimSpace(lastfmAPIKey) == "" {
		return nil, fmt.Errorf("LASTFM_API_KEY is empty")
	}

	endpoint := "https://ws.audioscrobbler.com/2.0/"
	q := url.Values{}
	q.Set("method", "artist.gettoptags")
	q.Set("artist", artist)
	q.Set("api_key", lastfmAPIKey)
	q.Set("format", "json")

	req, err := http.NewRequest("GET", endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GenresFromLastfm/1.0 (+https://last.fm)")
	req.Header.Set("Accept", "application/json")

	body, err := readLastFMBody(req)
	if err != nil {
		curlBody, curlErr := readLastFMBodyWithCurl(endpoint + "?" + q.Encode())
		if curlErr != nil {
			return nil, err
		}
		body = curlBody
	}

	return parseLastFMResponse(body, max)
}

func readLastFMBody(req *http.Request) ([]byte, error) {
	resp, err := doLastFMRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Last.fm HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func readLastFMBodyWithCurl(rawURL string) ([]byte, error) {
	cmd := exec.Command(curlPath, "-4", "--fail", "--silent", "--show-error", "--max-time", "20", rawURL)
	return cmd.Output()
}

func parseLastFMResponse(body []byte, max int) ([]string, error) {
	var data lastfmResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.Error != nil && *data.Error != 0 {
		msg := strings.TrimSpace(data.Message)
		if msg == "" {
			msg = "unknown Last.fm API error"
		}
		return nil, fmt.Errorf("Last.fm API error %d: %s", *data.Error, msg)
	}

	if data.TopTags == nil || len(data.TopTags.Tag) == 0 {
		return nil, nil
	}

	return filterLastFMTags(data.TopTags.Tag, max), nil
}

func filterLastFMTags(tags []lastfmTag, max int) []string {
	nonGenres := map[string]bool{
		// Keep this blacklist conservative. Last.fm tags such as ambient,
		// lo-fi, instrumental, and singer-songwriter are valid genres/styles
		// for this tool and should not be filtered out.
		"seen live": true, "under 2000 listeners": true,
		"favorites": true, "favorite": true, "to tag": true,
		"all": true, "my library": true, "library": true,
		"tag": true, "tags": true, "listened": true,
		"guitar": true, "modern": true, "alabama": true, "rickroll": true,
		"pete waterman": true, "pete": true, "waterman": true,
		"my top songs": true, "my": true, "top": true, "songs": true,
		"cute": true, "animals": true, "charity": true, "charitable": true,
		"80s": true, "90s": true, "00s": true,
		"2000s": true, "2010s": true, "2020s": true,
		"80's": true, "90's": true, "00's": true,
		"2000's": true, "2010's": true, "2020's": true,
		"1980s": true, "1990s": true,
		"1980's": true, "1990's": true,
	}

	countries := map[string]bool{
		"india": true, "usa": true, "united states": true, "swedish": true,
		"sweden": true, "canada": true, "canadian": true, "german": true,
		"germany": true, "american": true, "british": true, "uk": true,
		"united kingdom": true, "england": true, "english": true,
		"scottish": true, "welsh": true, "french": true, "france": true,
		"spanish": true, "spain": true, "italian": true, "italy": true,
		"norwegian": true, "norway": true, "finnish": true, "finland": true,
		"danish": true, "denmark": true, "polish": true, "poland": true,
		"russian": true, "japan": true, "japanese": true, "chinese": true,
		"china": true, "australia": true, "australian": true, "brazil": true,
		"brazilian": true, "mexico": true, "mexican": true, "argentina": true,
		"argentine": true, "argentinian": true, "chile": true, "chilean": true,
		"greece": true, "greek": true, "netherlands": true, "dutch": true,
		"belgium": true, "belgian": true, "austria": true, "austrian": true,
		"switzerland": true, "swiss": true, "portugal": true, "portuguese": true,
		"ireland": true, "irish": true, "iceland": true, "icelandic": true,
		"czech": true, "czech republic": true, "czechia": true, "slovakia": true,
		"slovak": true, "slovenia": true, "slovenian": true, "croatia": true,
		"croatian": true, "serbia": true, "serbian": true, "romania": true,
		"romanian": true, "hungary": true, "hungarian": true, "bulgaria": true,
		"bulgarian": true, "turkey": true, "turkish": true, "israel": true,
		"israeli": true, "south africa": true, "south african": true,
		"new zealand": true, "indonesia": true, "indonesian": true,
		"thailand": true, "thai": true, "south korea": true, "south korean": true,
		"taiwan": true, "taiwanese": true, "philippines": true, "filipino": true,
		"singapore": true, "singaporean": true, "malaysia": true, "malaysian": true,
		"vietnam": true, "vietnamese": true, "pakistan": true, "pakistani": true,
		"bangladesh": true, "bangladeshi": true, "sri lanka": true, "sri lankan": true,
	}

	decadePattern := regexp.MustCompile(`^(19|20)?\d0s?$`)

	var filtered []lastfmTag
	for _, t := range tags {
		name := strings.ToLower(strings.TrimSpace(t.Name))
		if name == "" {
			continue
		}
		if nonGenres[name] {
			continue
		}
		if countries[name] {
			continue
		}
		if decadePattern.MatchString(name) {
			continue
		}
		filtered = append(filtered, t)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Count > filtered[j].Count
	})

	if len(filtered) > max {
		filtered = filtered[:max]
	}

	var names []string
	for _, t := range filtered {
		name := strings.TrimSpace(t.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

// ─── File Walking ──────────────────────────────────────────────

func collectFiles(paths []string) ([]string, error) {
	var allFiles []string
	seen := make(map[string]bool)
	for _, target := range paths {
		abs, err := filepath.Abs(target)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			continue
		}
		files, err := walkMP3Files(abs)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				allFiles = append(allFiles, f)
			}
		}
	}
	return allFiles, nil
}

func walkMP3Files(target string) ([]string, error) {
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

func readArtist(file string) string {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err == nil {
		defer tag.Close()
		if artist := firstTextValue(
			tag.Artist(),
			tag.GetTextFrame("TPE2").Text,
			tag.GetTextFrame("TOPE").Text,
		); artist != "" {
			return artist
		}
	}

	metadata := readExiftoolMetadata(file)
	return firstTextValue(
		metadata.Artist,
		metadata.AlbumArtist,
		metadata.Band,
		metadata.OriginalArtist,
	)
}

func firstTextValue(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstArtist(raw string) string {
	idx := strings.Index(raw, "; ")
	if idx < 0 {
		return raw
	}
	return strings.TrimSpace(raw[:idx])
}

func readAlbum(file string) string {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err == nil {
		defer tag.Close()
		if album := strings.TrimSpace(tag.Album()); album != "" {
			return album
		}
	}

	return strings.TrimSpace(readExiftoolMetadata(file).Album)
}

type exiftoolMetadata struct {
	Artist         string `json:"Artist"`
	AlbumArtist    string `json:"AlbumArtist"`
	Band           string `json:"Band"`
	OriginalArtist string `json:"OriginalArtist"`
	Album          string `json:"Album"`
}

func readExiftoolMetadata(file string) exiftoolMetadata {
	cmd := exec.Command(exiftoolPath, "-j", "-Artist", "-AlbumArtist", "-Band", "-OriginalArtist", "-Album", file)
	out, err := cmd.Output()
	if err != nil {
		return exiftoolMetadata{}
	}

	var entries []exiftoolMetadata
	if err := json.Unmarshal(out, &entries); err != nil || len(entries) == 0 {
		return exiftoolMetadata{}
	}
	return entries[0]
}

func writeGenre(file, genre string) bool {
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return writeGenreWithExiftool(file, genre)
	}
	defer tag.Close()
	tag.SetGenre(genre)
	if err := tag.Save(); err != nil {
		return writeGenreWithExiftool(file, genre)
	}
	return true
}

func writeGenreWithExiftool(file, genre string) bool {
	cmd := exec.Command(exiftoolPath, "-overwrite_original", "-Genre="+genre, file)
	return cmd.Run() == nil
}

// ─── Settings Persistence ──────────────────────────────────────

func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".config", "genres")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "lastfm.json")
}

func loadSettings() settings {
	p := settingsPath()
	if p == "" {
		return defaultSettings
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return defaultSettings
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultSettings
	}
	if s.MaxGenres < 1 {
		s.MaxGenres = defaultSettings.MaxGenres
	}
	if s.BaseDir == "" {
		s.BaseDir = defaultSettings.BaseDir
	} else {
		s.BaseDir = filepath.Clean(s.BaseDir)
	}
	return s
}

func saveSettings(s settings) {
	p := settingsPath()
	if p == "" {
		return
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(p, data, 0644)
}

// ─── Path Resolution ──────────────────────────────────────────

func resolvePath(input string, baseDir string) string {
	input = strings.Trim(strings.TrimSpace(input), `"'`)
	if input == "" {
		return ""
	}
	if strings.HasPrefix(input, "/") {
		return input
	}
	if baseDir != "" {
		candidate := filepath.Join(baseDir, input)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return candidate
	}
	return input
}

// ─── API Key Loading ────────────────────────────────────────────

func loadAPIKey() bool {
	if key := strings.TrimSpace(os.Getenv("LASTFM_API_KEY")); key != "" {
		lastfmAPIKey = strings.Trim(key, `"'`)
		return true
	}

	for _, p := range candidateEnvFiles() {
		if key := readLastFMKeyFromFile(p); key != "" {
			lastfmAPIKey = key
			return true
		}
	}
	return false
}

func candidateEnvFiles() []string {
	var paths []string
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".env"))
		paths = append(paths, filepath.Join(filepath.Dir(filepath.Dir(exe)), ".env"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".env"),
			filepath.Join(home, ".config", "genres", ".env"),
		)
	}

	seen := make(map[string]bool, len(paths))
	out := paths[:0]
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func readLastFMKeyFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "LASTFM_API_KEY" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if value != "" {
			return value
		}
	}
	return ""
}

// ─── Validation ─────────────────────────────────────────────────

func validateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("No MP3 file/folder path provided.")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("Path does not exist: %s", target)
	}
	return nil
}

// ─── Main ───────────────────────────────────────────────────────

func undoMenu() {
	for {
		clearScreen()
		fmt.Println(renderPageHeader("U N D O", "󰕌", "M A N A G E", "Restore or clean Last.fm backup sessions"))
		fmt.Println()
		fmt.Println("   " + rowIconStyle.Render("󰌋") + "  " + keyStyle.Render("1") + actionLabelStyle.Render("  Undo Last") + "  " + menuDescStyle.Render("Restore latest Last.fm write"))
		fmt.Println("   " + rowIconStyle.Render("󰏃") + "  " + keyStyle.Render("2") + actionLabelStyle.Render("  Backups") + "  " + menuDescStyle.Render("List or clean old undo sessions"))
		fmt.Println("   " + rowIconStyle.Render("") + "  " + keyStyle.Render("b") + actionLabelStyle.Render("  Back") + "  " + menuDescStyle.Render("Return to main menu"))
		fmt.Println()
		fmt.Print(promptStyle.Render("❯ Choice:"))
		fmt.Print(" ")
		switch strings.ToLower(readLine()) {
		case "1":
			clearScreen()
			fmt.Println(renderPageHeader("U N D O", "󰕌", "L A S T", "Restore files from the latest Last.fm write"))
			fmt.Println()
			count, err := genrenorm.RestoreLatestUndo("lastfm")
			if err != nil {
				fmt.Println("   " + errorStyle.Render("󰅏 "+err.Error()))
			} else {
				fmt.Printf("   %s %s\n", successStyle.Render("◆"), successStyle.Render(fmt.Sprintf("Restored %d file(s)", count)))
			}
			promptReturn()
		case "2":
			manageBackups("lastfm")
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
		fmt.Println("   " + errorStyle.Render("󰅏 "+err.Error()))
		promptReturn()
		return
	}
	for {
		clearScreen()
		fmt.Println(renderPageHeader("B A C K U P S", "󰏃", strings.ToUpper(tool), "Manage undo backup sessions"))
		fmt.Println()
		if len(sessions) == 0 {
			fmt.Println("   " + mutedStyle.Render("No backup sessions found."))
		} else {
			fmt.Println("  " + renderSection("󰏃", "Sessions"))
			totalSize := int64(0)
			for i, s := range sessions {
				totalSize += s.SizeBytes
				fmt.Printf("   %s  %s  %s  %s  %s\n",
					keyStyle.Render(fmt.Sprintf("%d", i+1)),
					rowIconStyle.Render("󰏋"),
					actionLabelStyle.Render(s.Timestamp.Format("2006-01-02 15:04:05")),
					countStyle.Render(fmt.Sprintf("%d files", s.FileCount)),
					pathStyle.Render(genrenorm.FormatBytes(s.SizeBytes)),
				)
			}
			fmt.Println()
			fmt.Printf("   %s  %s\n", actionLabelStyle.Render("Total:"), countStyle.Render(fmt.Sprintf("%d sessions, %s", len(sessions), genrenorm.FormatBytes(totalSize))))
		}
		fmt.Println()
		fmt.Println("  " + mutedStyle.Render("c clean old sessions  b back"))
		fmt.Println()
		fmt.Print(promptStyle.Render("❯ Choice:"))
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
	fmt.Println("   " + actionLabelStyle.Render("The most recent session is always kept."))
	fmt.Println()
	fmt.Print(promptStyle.Render("❯ Sessions to keep (default 3):"))
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
		fmt.Println("   " + mutedStyle.Render("Nothing to clean."))
		promptReturn()
		return
	}
	fmt.Println()
	fmt.Printf("   %s  %s\n", warnStyle.Render("⚠"), warnStyle.Render(fmt.Sprintf("This will remove %d old session(s).", len(sessions)-keep)))
	fmt.Print(promptStyle.Render("❯ Confirm? [y/N]:"))
	fmt.Print(" ")
	if !strings.EqualFold(readLine(), "y") {
		return
	}
	removed, err := genrenorm.CleanOldUndoSessions(tool, keep)
	if err != nil {
		fmt.Println()
		fmt.Println("   " + errorStyle.Render("󰅏 "+err.Error()))
	} else {
		fmt.Println()
		fmt.Printf("   %s %s\n", successStyle.Render("◆"), successStyle.Render(fmt.Sprintf("Removed %d session(s).", removed)))
	}
	promptReturn()
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			clearScreen()
			fmt.Println(renderPageHeader("E R R O R", "󰅏", "P A N I C", "An unexpected error occurred"))
			fmt.Println()
			fmt.Printf("   %s %v\n", errorStyle.Render("✖"), r)
			fmt.Println()
			fmt.Println("  " + mutedStyle.Render("Press Enter to exit..."))
			readLine()
		}
	}()

	writeFlag := flag.Bool("write", false, "Actually update MP3 tags (default: dry-run)")
	undoFlag := flag.Bool("undo", false, "Restore files from the latest write")
	maxGenresFlag := flag.Int("max-genres", 3, "Number of final genres to write")
	flag.Parse()

	if *undoFlag {
		count, err := genrenorm.RestoreLatestUndo("lastfm")
		if err != nil {
			fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
			os.Exit(1)
		}
		fmt.Printf("  %s\n", successStyle.Render(fmt.Sprintf("Restored %d file(s)", count)))
		return
	}

	s := loadSettings()

	cfg.write = *writeFlag
	cfg.maxGenres = *maxGenresFlag

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
		cfg.write = !s.DryRun
	}
	if !isFlagPassed("max-genres") {
		cfg.maxGenres = s.MaxGenres
	}

	if flag.NArg() > 0 {
		cfg.target = flag.Arg(0)
		if err := validateTarget(cfg.target); err != nil {
			fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
			os.Exit(1)
		}
		processPathsCli([]string{cfg.target})
		return
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		path := strings.TrimSpace(input)
		if path != "" {
			cfg.target = resolvePath(path, s.BaseDir)
			if err := validateTarget(cfg.target); err != nil {
				fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
				os.Exit(1)
			}
			processPathsCli([]string{cfg.target})
			return
		}
	}

	interactiveLoop()
}

// ─── CLI Processing (non-interactive mode) ──────────────────────

type processingSummary struct {
	Targets       int
	Files         int
	Directories   int
	Artists       int
	FoundArtists  int
	APIErrors     int
	Updated       int
	DryCandidates int
	Skipped       int
	NoArtist      int
	Failed        int
}

func renderProcessingHeader(targetCount, fileCount, dirCount, artistCount, maxGenres int, write bool) {
	mode := "Dry"
	modeStyler := modeStyle
	if write {
		mode = "Write"
		modeStyler = writeModeStyle
	}
	fmt.Println(renderPageHeader("", "", "P R O C E S S", "Fetch and normalize artist genres"))
	fmt.Println()
	fmt.Printf("          %s %s: %s  %s  %s %s: %s  %s  %s %s: %s\n",
		sectionIconStyle.Render("󰚔"), statusLabelStyle.Render("Targets"), valueStyle.Render(fmt.Sprintf("%d", targetCount)),
		dotStyle.Render("•"),
		sectionIconStyle.Render(""), statusLabelStyle.Render("Mode"), modeStyler.Render(mode),
		dotStyle.Render("•"),
		sectionIconStyle.Render(""), statusLabelStyle.Render("Genres"), valueStyle.Render(fmt.Sprintf("%d", maxGenres)),
	)
	fmt.Println()
	fmt.Print(renderSection("󰄉", "Status"))
	fmt.Printf("   %s  %-24s %s\n", successStyle.Render("◆"), actionLabelStyle.Render("Found"), countStyle.Render(fmt.Sprintf("%d MP3s across %d directories", fileCount, dirCount)))
	fmt.Printf("   %s  %-24s %s\n", infoStyle.Render("◌"), actionLabelStyle.Render("Fetching"), pathStyle.Render(fmt.Sprintf("%d unique artists from Last.fm", artistCount)))
	fmt.Println()
}

func renderSummary(summary processingSummary) {
	mode := "Dry"
	modeStyler := modeStyle
	if cfg.write {
		mode = "Write"
		modeStyler = writeModeStyle
	}
	fmt.Println()
	fmt.Println(renderPageHeader("", "", "S U M M A R Y", "Results for this session's processed files"))
	fmt.Println()
	fmt.Printf("        %s %s: %s  %s  %s %s: %s  %s  %s %s: %s\n",
		sectionIconStyle.Render("󰚔"), statusLabelStyle.Render("Targets"), valueStyle.Render(fmt.Sprintf("%d", summary.Targets)),
		dotStyle.Render("•"),
		sectionIconStyle.Render(""), statusLabelStyle.Render("Mode"), modeStyler.Render(mode),
		dotStyle.Render("•"),
		sectionIconStyle.Render(""), statusLabelStyle.Render("Genres"), valueStyle.Render(fmt.Sprintf("%d", cfg.maxGenres)),
	)
	fmt.Println()
	fmt.Print(renderSection("󰱽", "Results"))
	if cfg.write {
		fmt.Printf("   %s  %-24s %s\n", successStyle.Render("󰼄"), actionLabelStyle.Render("Files Updated"), successStyle.Render(fmt.Sprintf("%d", summary.Updated)))
	} else {
		fmt.Printf("   %s  %-24s %s\n", successStyle.Render("󰼄"), actionLabelStyle.Render("Candidates"), successStyle.Render(fmt.Sprintf("%d", summary.DryCandidates)))
	}
	fmt.Printf("   %s  %-24s %s\n", infoStyle.Render("󱈡"), actionLabelStyle.Render("No Tags"), skyStyle.Render(fmt.Sprintf("%d", summary.Skipped)))
	if summary.APIErrors > 0 {
		fmt.Printf("   %s  %-24s %s\n", errorStyle.Render("󰅏"), actionLabelStyle.Render("API Errors"), errorStyle.Render(fmt.Sprintf("%d", summary.APIErrors)))
	}
	fmt.Printf("   %s  %-24s %s\n", warnStyle.Render("󰳩"), actionLabelStyle.Render("No Artist"), warnStyle.Render(fmt.Sprintf("%d", summary.NoArtist)))
	if summary.Failed > 0 {
		fmt.Printf("   %s  %-24s %s\n", errorStyle.Render("󰅏"), actionLabelStyle.Render("Failed"), errorStyle.Render(fmt.Sprintf("%d", summary.Failed)))
	}
}

func processPathsCli(paths []string) {
	if !loadAPIKey() {
		fmt.Println("  " + errorStyle.Render("✖ Error: LASTFM_API_KEY not found."))
		os.Exit(1)
	}

	files, err := collectFiles(paths)
	if err != nil {
		fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Println("  " + warnStyle.Render("No MP3 files found."))
		return
	}

	_, err = runProcessingCLI(files)
	if err != nil {
		fmt.Println("  " + errorStyle.Render("✖ "+err.Error()))
		os.Exit(1)
	}
}

func runProcessingCLI(files []string) (processingSummary, error) {
	summary := processingSummary{Targets: 1, Files: len(files)}
	var undoSession *genrenorm.UndoSession

	if cfg.write {
		globalSettings, _ := genrenorm.LoadGlobalSettings()

		// Check if backup is enabled
		if globalSettings.EnableBackup {
			// Prompt for backup if enabled
			if globalSettings.PromptForBackup {
				fmt.Println()
				fmt.Print(promptStyle.Render("❯ Save backup before writing? [Y/n]:"))
				fmt.Print(" ")
				answer := strings.ToLower(readLine())
				if answer == "" || answer == "y" || answer == "yes" {
					session, err := genrenorm.StartUndoSession("lastfm")
					if err != nil {
						return summary, fmt.Errorf("could not create undo backup session: %w", err)
					}
					undoSession = session
				}
			} else {
				// Auto backup without prompting
				session, err := genrenorm.StartUndoSession("lastfm")
				if err != nil {
					return summary, fmt.Errorf("could not create undo backup session: %w", err)
				}
				undoSession = session
			}
		}
	}

	dirGroups := make(map[string][]string)
	for _, f := range files {
		d := filepath.Dir(f)
		dirGroups[d] = append(dirGroups[d], f)
	}
	var dirKeys []string
	for d := range dirGroups {
		dirKeys = append(dirKeys, d)
	}
	sort.Strings(dirKeys)
	summary.Directories = len(dirKeys)

	dirArtist := make(map[string]string, len(dirKeys))
	dirAlbum := make(map[string]string, len(dirKeys))
	for _, dir := range dirKeys {
		group := dirGroups[dir]
		for _, f := range group {
			if a := readArtist(f); a != "" {
				dirArtist[dir] = a
				break
			}
		}
		if a := readAlbum(group[0]); a != "" {
			dirAlbum[dir] = a
		}
	}

	uniqueArtists := make(map[string]string)
	for _, a := range dirArtist {
		first := firstArtist(a)
		if first == "" {
			continue
		}
		key := strings.ToLower(first)
		if _, ok := uniqueArtists[key]; ok {
			continue
		}
		uniqueArtists[key] = first
	}
	artistList := make([]string, 0, len(uniqueArtists))
	for _, name := range uniqueArtists {
		artistList = append(artistList, name)
	}
	sort.Strings(artistList)
	summary.Artists = len(artistList)

	renderProcessingHeader(1, len(files), len(dirKeys), len(artistList), cfg.maxGenres, cfg.write)

	artistCache := make(map[string][]string)
	foundCount := 0

	for i, artist := range artistList {
		counter := counterStyle.Render(fmt.Sprintf("[%d/%d]", i+1, len(artistList)))
		artistName := artistStyle.Render(artist)
		fmt.Printf("   %s  %s  %s", infoStyle.Render("◌"), counter, artistName)

		rawTags, fetchErr := fetchTopTagsWithError(artist, cfg.maxGenres)
		key := strings.ToLower(strings.TrimSpace(artist))

		if fetchErr != nil {
			summary.APIErrors++
			artistCache[key] = []string{}
			fmt.Printf("\r\033[2K   %s  %s  %s  %s\n",
				warnStyle.Render("󰅏"), counter, artistName, warnStyle.Render(fetchErr.Error()))
			continue
		}

		if len(rawTags) > 0 {
			genres := genrenorm.ExpandGenres(rawTags)
			if len(genres) > cfg.maxGenres {
				genres = genres[:cfg.maxGenres]
			}
			artistCache[key] = genres
			foundCount++
			fmt.Printf("\r\033[2K   %s  %s  %s  %s\n",
				successStyle.Render("◆"), counter, artistName, genreStyle.Render(strings.Join(genres, "; ")))
		} else {
			artistCache[key] = []string{}
			fmt.Printf("\r\033[2K   %s  %s  %s  %s\n",
				mutedStyle.Render("∿"), counter, artistName, mutedStyle.Render("no tags found"))
		}
	}
	summary.FoundArtists = foundCount

	fmt.Println()
	fmt.Printf("   %s  %-24s %s\n", mutedStyle.Render("◆"), actionLabelStyle.Render("Genres Found"), countStyle.Render(fmt.Sprintf("%d / %d artists", foundCount, len(artistList))))
	fmt.Println()
	fmt.Print(renderSection("", "Files"))

	for _, dir := range dirKeys {
		group := dirGroups[dir]
		album := filepath.Base(dir)
		if a := dirAlbum[dir]; a != "" {
			album = a
		}

		artistName := dirArtist[dir]
		first := firstArtist(artistName)

		albumLine := albumStyle.Render(album)
		if first != "" {
			albumLine = albumStyle.Render(album) + "  " + arrowStyle.Render("—") + "  " + artistStyle.Render(first)
		}
		fmt.Printf("\n  %s\n", albumLine)
		fmt.Println("  " + sepStyle.Render("────────────────────────────────────────────────────────────"))

		for _, f := range group {
			base := filepath.Base(f)

			if first == "" {
				summary.NoArtist++
				fmt.Printf("   %s  %s\n", warnTagStyle.Render("○"), warnTagStyle.Render(base+" — no artist tag"))
				continue
			}
			key := strings.ToLower(first)
			genres := artistCache[key]
			if len(genres) == 0 {
				summary.Skipped++
				fmt.Printf("   %s  %s\n", infoStyle.Render("∿"), fmt.Sprintf("%s  %s  %s", fileStyle.Render(base), arrowStyle.Render("—"), warnTagStyle.Render(fmt.Sprintf("no Last.fm tags for %q", first))))
				continue
			}
			genreValue := strings.Join(genres, "; ")

			if !cfg.write {
				summary.DryCandidates++
				fmt.Printf("   %s  %s\n", modeStyle.Render("◈"), fmt.Sprintf("%s  %s  %s", fileStyle.Render(base), arrowStyle.Render("→"), genreStyle.Render(genreValue)))
				continue
			}

			if undoSession != nil {
				if err := undoSession.Backup(f); err != nil {
					summary.Failed++
					fmt.Printf("   %s  %s\n", errorStyle.Render("✖"), errorStyle.Render(base+"  —  backup failed: "+err.Error()))
					continue
				}
			}

			if ok := writeGenre(f, genreValue); ok {
				summary.Updated++
				fmt.Printf("   %s  %s\n", successStyle.Render("◆"), fmt.Sprintf("%s  %s  %s", fileStyle.Render(base), arrowStyle.Render("→"), genreStyle.Render(genreValue)))
			} else {
				summary.Failed++
				fmt.Printf("   %s  %s\n", errorStyle.Render("✖"), errorStyle.Render(base+"  —  write failed"))
			}
		}
	}

	renderSummary(summary)

	if cfg.write && summary.Updated > 0 {
		fmt.Println()
		fmt.Printf("   %s  %s\n", successStyle.Render("◆"), successStyle.Render("Done"))
	}

	if summary.Failed > 0 {
		return summary, fmt.Errorf("%d write failures", summary.Failed)
	}
	return summary, nil
}
