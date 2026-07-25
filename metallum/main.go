package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-genres/shared"
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

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// ─── Lip Gloss Styles ───────────────────────────────────────────

var (
	styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("222"))
	styleSuccess   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	styleError     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleArtist    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	styleGenre     = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	styleCount     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
)

func renderSep(width int) string {
	return styleSeparator.Render(strings.Repeat("─", width))
}

func renderHR(width int) string {
	return styleSeparator.Render(strings.Repeat("═", width))
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
	fmt.Print("\r     ")
}

// ─── Helpers ────────────────────────────────────────────────────

type bandResult struct {
	Name    string
	Genre   string
	Country string
	URL     string
}

// ─── HTTP ───────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 20 * time.Second}

func curlGet(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ─── Metal Archives Search ─────────────────────────────────────

type searchResponse struct {
	AAttributes any         `json:"aaAttributes"`
	AAiColumn   any         `json:"aaAiColumn"`
	AData       [][3]string `json:"aaData"`
	Err         string      `json:"error"`
}

type albumSearchResponse struct {
	AAttributes any         `json:"aaAttributes"`
	AAiColumn   any         `json:"aaAiColumn"`
	AData       [][4]string `json:"aaData"`
	Err         string      `json:"error"`
}

func normalized(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

var htmlTag = regexp.MustCompile(`<a[^>]*>([^<]+)</a>`)
var linkTag = regexp.MustCompile(`<a\s+href=['"]([^'"]+)['"][^>]*>([^<]+)</a>`)

func searchBandCandidates(artist string) []bandResult {
	url := fmt.Sprintf("https://www.metal-archives.com/search/ajax-band-search/?field=name&query=%s", urlQueryEscape(artist))
	raw, err := curlGet(url)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}

	var resp searchResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil
	}

	lower := normalized(artist)
	var candidates []bandResult

	for _, row := range resp.AData {
		nameHTML := row[0]
		genre := strings.TrimSpace(row[1])
		country := strings.TrimSpace(row[2])

		m := linkTag.FindStringSubmatch(nameHTML)
		if len(m) < 3 {
			// fallback to plain text
			m2 := htmlTag.FindStringSubmatch(nameHTML)
			if len(m2) < 2 {
				continue
			}
			bandName := strings.TrimSpace(m2[1])
			bandLower := normalized(bandName)
			if bandLower == lower || strings.Contains(lower, bandLower) || strings.Contains(bandLower, lower) {
				candidates = append(candidates, bandResult{Name: bandName, Genre: genre, Country: country})
			}
			continue
		}
		bandURL := strings.TrimSpace(m[1])
		bandName := strings.TrimSpace(m[2])

		bandLower := normalized(bandName)
		if bandLower == lower || strings.Contains(lower, bandLower) || strings.Contains(bandLower, lower) {
			candidates = append(candidates, bandResult{Name: bandName, Genre: genre, Country: country, URL: bandURL})
		}
	}
	return candidates
}

func searchAlbumBands(album string) map[string]string {
	url := fmt.Sprintf("https://www.metal-archives.com/search/ajax-album-search/?field=album&query=%s", urlQueryEscape(album))
	raw, err := curlGet(url)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}

	var resp albumSearchResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil
	}

	// Row format: [band_html, album_html, year, type]
	bands := make(map[string]string)
	for _, row := range resp.AData {
		bandHTML := row[0]
		m := linkTag.FindStringSubmatch(bandHTML)
		if len(m) >= 3 {
			bandURL := strings.TrimSpace(m[1])
			bandName := strings.TrimSpace(m[2])
			bands[bandURL] = bandName
		}
	}
	return bands
}

func searchBand(artist, album string) *bandResult {
	candidates := searchBandCandidates(artist)
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		best := candidates[0]
		return &best
	}

	// Multiple candidates — match by unique band URL via album search
	if album != "" {
		albumBands := searchAlbumBands(album)
		if len(albumBands) > 0 {
			for _, c := range candidates {
				if c.URL != "" && albumBands[c.URL] != "" {
					best := c
					return &best
				}
			}
		}
	}

	// Fallback: prefer exact name match
	lower := normalized(artist)
	for _, c := range candidates {
		if normalized(c.Name) == lower {
			best := c
			return &best
		}
	}

	// Last resort: first candidate
	best := candidates[0]
	return &best
}

func urlQueryEscape(s string) string {
	// Manual encoding preserving URL structure
	var out strings.Builder
	for _, r := range s {
		if r == ' ' {
			out.WriteString("%20")
		} else if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == '~' {
			out.WriteRune(r)
		} else {
			out.WriteString(fmt.Sprintf("%%%02X", r))
		}
	}
	return out.String()
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

// ─── Constants & Settings ─────────────────────────────────────

const appWidth = 63
const deemixQuickPath = "/Volumes/Eksternal/Music/Downloads/Deemix"
const soulseekQuickPath = "/Volumes/Eksternal/Music/Downloads/Soulseek"

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

// ─── Theme (Crimson / Amber) ────────────────────────────────────

var (
	mBorderStyle      lipgloss.Style
	mTitleStyle       lipgloss.Style
	mTitleIconStyle   lipgloss.Style
	mDescStyle        lipgloss.Style
	mSectionStyle     lipgloss.Style
	mSectionIconStyle lipgloss.Style
	mKeyStyle         lipgloss.Style
	mRowIconStyle     lipgloss.Style
	mActionLabelStyle lipgloss.Style
	mMenuDescStyle    lipgloss.Style
	mSepStyle         lipgloss.Style
	mPromptStyle      lipgloss.Style
	mErrorStyle       lipgloss.Style
	mMutedStyle       lipgloss.Style
	mValueStyle       lipgloss.Style
	mPathStyle        lipgloss.Style
	mModeStyle        lipgloss.Style
	mWriteModeStyle   lipgloss.Style
	mWarnStyle        lipgloss.Style
	mSuccessStyle     lipgloss.Style
	mInfoStyle        lipgloss.Style
	mCountStyle       lipgloss.Style
	mArtistStyle      lipgloss.Style
	mGenreStyle       lipgloss.Style
	mDimStyle         lipgloss.Style
	mArrowStyle       lipgloss.Style
	mFileStyle        lipgloss.Style
	mStatusLabelStyle lipgloss.Style
)

func initMetallumMenuStyles() {
	ld := func(dark, light lipgloss.Color) lipgloss.Color {
		if lipgloss.HasDarkBackground() {
			return dark
		}
		return light
	}

	// Warm brown / pink palette (Metal Archives)
	dawnPink := lipgloss.Color("#f3e5e5")
	swissCoffee := lipgloss.Color("#e1d8d6")
	coldTurkey := lipgloss.Color("#6e4747")
	delRio := lipgloss.Color("#af9999")
	auChico := lipgloss.Color("#8c5f5f")
	ferra := lipgloss.Color("#6e4747")
	asphalt := lipgloss.Color("#0c0303")
	hurricane := lipgloss.Color("#837b78")
	cocoaBean := lipgloss.Color("#e79a9a")
	borderC := ld(ferra, delRio)
	sepC := ld(coldTurkey, delRio)
	titleC := ld(dawnPink, asphalt)
	sectionC := ld(delRio, auChico)
	textC := ld(dawnPink, asphalt)
	mutedC := ld(coldTurkey, hurricane)
	dimC := ld(coldTurkey, hurricane)

	mBorderStyle = lipgloss.NewStyle().Foreground(borderC)
	mTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(titleC)
	mTitleIconStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(auChico, cocoaBean))
	mDescStyle = lipgloss.NewStyle().Foreground(delRio)
	mSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(sectionC)
	mSectionIconStyle = lipgloss.NewStyle().Foreground(ld(swissCoffee, ferra))
	mKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(dawnPink, cocoaBean))
	mRowIconStyle = lipgloss.NewStyle().Foreground(ld(cocoaBean, ferra))
	mActionLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(textC)
	mMenuDescStyle = lipgloss.NewStyle().Foreground(ld(delRio, hurricane))
	mSepStyle = lipgloss.NewStyle().Foreground(sepC)
	mPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(dawnPink, cocoaBean))
	mErrorStyle = lipgloss.NewStyle().Foreground(ld(cocoaBean, asphalt))
	mMutedStyle = lipgloss.NewStyle().Foreground(mutedC)
	mValueStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(dawnPink, asphalt))
	mPathStyle = lipgloss.NewStyle().Foreground(ld(delRio, ferra))
	mModeStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(swissCoffee, asphalt))
	mWriteModeStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(auChico, cocoaBean))
	mWarnStyle = lipgloss.NewStyle().Foreground(ld(auChico, cocoaBean))
	mSuccessStyle = lipgloss.NewStyle().Foreground(ld(dawnPink, asphalt))
	mInfoStyle = lipgloss.NewStyle().Foreground(ld(delRio, ferra))
	mCountStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(dawnPink, asphalt))
	mArtistStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(swissCoffee, ferra))
	mGenreStyle = lipgloss.NewStyle().Foreground(ld(dawnPink, asphalt))
	mDimStyle = lipgloss.NewStyle().Foreground(dimC)
	mArrowStyle = lipgloss.NewStyle().Foreground(ld(swissCoffee, ferra))
	mFileStyle = lipgloss.NewStyle().Foreground(textC)
	mStatusLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(ld(swissCoffee, ferra))
}

func init() {
	initMetallumMenuStyles()
}

// ─── Screen Helpers ─────────────────────────────────────────────

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func readLine() string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func readInput(prompt string) string {
	fmt.Print(mPromptStyle.Render("❯ " + prompt + ":"))
	fmt.Print(" ")
	return readLine()
}

func promptReturn() {
	fmt.Println()
	fmt.Println("  " + mMutedStyle.Render("Press Enter to return..."))
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

// ─── Renderers ──────────────────────────────────────────────────

func renderTitle(left, icon, right string) string {
	return mTitleStyle.Render(left) + "  " + mTitleIconStyle.Render(icon) + "  " + mTitleStyle.Render(right)
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
	title := strings.Repeat(" ", leftPad) + mTitleStyle.Render(titleLeft) + "  " + mTitleIconStyle.Render(icon) + "  " + mTitleStyle.Render(titleRight) + strings.Repeat(" ", rightPad)

	return strings.Join([]string{
		mBorderStyle.Render("               ╭────────────────────────────────╮"),
		mBorderStyle.Render("╭──────────────┤") + title + mBorderStyle.Render("├─────────────╮"),
		mBorderStyle.Render("│              ╰────────────────────────────────╯             │"),
		mBorderStyle.Render("│") + mDescStyle.Render(centerText(desc, appWidth-2)) + mBorderStyle.Render("│"),
		mBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSection(icon, title string) string {
	sep := mSepStyle.Render(strings.Repeat("─", 56))
	return fmt.Sprintf("  %s %s\n  %s\n", mSectionIconStyle.Render(icon), mSectionStyle.Render(title), sep)
}

func renderMenuItem(key, icon, label, desc string) string {
	return fmt.Sprintf("   %s  %s  %s%s",
		mKeyStyle.Render(key),
		mRowIconStyle.Render(icon),
		mActionLabelStyle.Render(padRight(label, 22)),
		mMenuDescStyle.Render(desc))
}

func renderSettingItem(key, label, value string) string {
	return fmt.Sprintf("   %s  %s%s",
		mKeyStyle.Render(key),
		mActionLabelStyle.Render(padRight(label, 22)),
		mValueStyle.Render(value))
}

func renderMainHeader() string {
	title := renderTitle("M E T A L", "󰪦", "A R C H I V E S")
	return strings.Join([]string{
		mBorderStyle.Render("               ╭───────────────────────────────╮"),
		mBorderStyle.Render("╭──────────────┤ ") + title + mBorderStyle.Render(" ├──────────────╮"),
		mBorderStyle.Render("│              ╰───────────────────────────────╯              │"),
		mBorderStyle.Render("│") + mDescStyle.Render(centerText("Scrape genres from Encyclopaedia Metallum", appWidth-2)) + mBorderStyle.Render("│"),
		mBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderSettingsHeader() string {
	title := strings.Repeat(" ", 5) + mTitleIconStyle.Render("") + "  " + mTitleStyle.Render("S E T T I N G S") + strings.Repeat(" ", 7)
	return strings.Join([]string{
		mBorderStyle.Render("               ╭───────────────────────────────╮"),
		mBorderStyle.Render("╭──────────────┤ ") + title + mBorderStyle.Render("├──────────────╮"),
		mBorderStyle.Render("│              ╰───────────────────────────────╯              │"),
		mBorderStyle.Render("│") + mDescStyle.Render(centerText("Configure default Metallum scraping behavior", appWidth-2)) + mBorderStyle.Render("│"),
		mBorderStyle.Render("╰─────────────────────────────────────────────────────────────╯"),
	}, "\n") + "\n"
}

func renderStatusLine(s savedSettings) string {
	modeLabel := "Dry"
	modeStyler := mModeStyle
	if s.Write {
		modeLabel = "Write"
		modeStyler = mWriteModeStyle
	}
	return fmt.Sprintf("        %s %s: %s  %s  %s %s: %s  %s  %s %s: %s",
		mSectionIconStyle.Render(""),
		mStatusLabelStyle.Render("Genres"),
		mValueStyle.Render(fmt.Sprintf("%d", s.MaxGenres)),
		mDimStyle.Render("•"),
		mSectionIconStyle.Render(""),
		mStatusLabelStyle.Render("Mode"),
		modeStyler.Render(modeLabel),
		mDimStyle.Render("•"),
		mSectionIconStyle.Render("󱦞"),
		mStatusLabelStyle.Render("Delay"),
		mValueStyle.Render(fmt.Sprintf("%dms", s.DelayMs)),
	)
}

func renderMainMenu(s savedSettings) {
	fmt.Print(renderMainHeader())
	fmt.Println()
	fmt.Println(renderStatusLine(s))
	fmt.Println()
	fmt.Print(renderSection("", "Input"))
	fmt.Println(renderMenuItem("1", "\uf506", "Path", "Enter file/folder paths"))
	fmt.Println(renderMenuItem("2", "\U000f1266", "Clipboard", "Use path from clipboard"))
	fmt.Println()
	fmt.Print(renderSection("", "Search"))
	fmt.Println(renderMenuItem("3", "\U000f0036", "Finder", "Select folder with Finder"))
	fmt.Println(renderMenuItem("4", "", "FZF", "Search from audio library"))
	fmt.Println()
	fmt.Print(renderSection("", "Actions"))
	fmt.Println(renderMenuItem("5", "\uf0e2", "Run Last", "Re-run previous target"))
	fmt.Println(renderMenuItem("6", "\uf001", "Deemix", "Run on download folder"))
	fmt.Println(renderMenuItem("7", "\uf001", "Soulseek", "Run on Soulseek folder"))
	fmt.Println(renderMenuItem("u", "󰕌", "Undo Last", "Restore files from last write"))
	fmt.Println()
	fmt.Print(renderSection("", "System"))
	fmt.Println(renderMenuItem("s", "󰆍", "Settings", "Change defaults"))
	fmt.Println(renderMenuItem("q", "", "Quit", "Exit"))
	fmt.Println()
}

func renderSettingsMenu(s savedSettings) {
	var modeStr string
	var modeStyler lipgloss.Style
	if s.Write {
		modeStr = "write"
		modeStyler = mWriteModeStyle
	} else {
		modeStr = "dry-run"
		modeStyler = mModeStyle
	}

	fmt.Print(renderSettingsHeader())
	fmt.Println()
	fmt.Print(renderSection("󰟓", "Options"))
	fmt.Println(renderSettingItem("w", "Default Mode", modeStyler.Render(modeStr)))
	fmt.Println(renderSettingItem("g", "Max Genres", fmt.Sprintf("%d", s.MaxGenres)))
	fmt.Println(renderSettingItem("d", "Delay", fmt.Sprintf("%d ms", s.DelayMs)))
	fmt.Println()
	fmt.Print(renderSection("", "Actions"))
	fmt.Println(renderMenuItem("s", "\uf00c", "Save", "Write settings and return"))
	fmt.Println(renderMenuItem("x", "\uf00d", "Back", "Return without saving"))
	fmt.Println()
}

// ─── Settings Persistence ─────────────────────────────────────

func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".config", "genres")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "metallum.json")
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
	fmt.Println("                                    " + mMenuDescStyle.Render("b back • q quit • enter confirm"))
	fmt.Print(mPromptStyle.Render("❯ Path:"))
	fmt.Print(" ")
	input := readLine()
	if input == "" || strings.EqualFold(input, "b") || strings.EqualFold(input, "q") {
		return ""
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		fmt.Println("   " + mErrorStyle.Render("󰅏 Invalid path"))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + mWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + mPathStyle.Render(abs))
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
	fmt.Println("   " + mMutedStyle.Render("󱘟 Reading clipboard..."))
	fmt.Println()

	content := readClipboard()
	if content == "" {
		fmt.Println("   " + mWarnStyle.Render("󰅏 Empty: Copy a file or folder path first."))
		promptReturn()
		return ""
	}
	abs, err := filepath.Abs(content)
	if err != nil {
		fmt.Println("   " + mWarnStyle.Render("󰅏 Invalid path: "+content))
		promptReturn()
		return ""
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Println("   " + mWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + mPathStyle.Render(abs))
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
	fmt.Println("   " + mMutedStyle.Render("") + " " + mStatusLabelStyle.Render("Mode") + ": " + mMutedStyle.Render("directory search with find"))
	fmt.Println()

	path, err := runFZFSearch()
	if err != nil {
		fmt.Println("   " + mWarnStyle.Render("󰜺 Cancelled") + "  " + mMutedStyle.Render(err.Error()))
		promptReturn()
		return ""
	}
	if path == "" {
		promptReturn()
		return ""
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("   " + mWarnStyle.Render("󰅏 Path does not exist"))
		fmt.Println("      " + mPathStyle.Render(path))
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

// ─── Interactive Processing ───────────────────────────────────

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
		fmt.Println("   " + mErrorStyle.Render("󰅏 "+err.Error()))
		return false
	}

	_, err := processTarget(path)

	if !cfg.write {
		fmt.Println()
		fmt.Print(mPromptStyle.Render("❯ Write these changes now? [y/N]:"))
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
		fmt.Println("   " + mErrorStyle.Render("✖ "+err.Error()))
	}

	if cfg.write {
		fmt.Println()
		fmt.Print(mPromptStyle.Render("❯ Open in Mp3tag? [y/N]:"))
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

		case "3":
			clearScreen()
			fmt.Println(renderPageHeader("", "󰀶", "F I N D E R", "Select a folder with the macOS folder picker"))
			fmt.Println()
			fmt.Print(renderSection("󰄉", "Status"))
			fmt.Println("   " + mMutedStyle.Render("󰪵 Opening Finder dialog..."))
			path := openFinderDialog()
			if path == "" {
				fmt.Println()
				fmt.Println("   " + mWarnStyle.Render("󰜺 Cancelled") + "  " + mMutedStyle.Render("Finder dialog was cancelled."))
				promptReturn()
				continue
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Println()
				fmt.Println("   " + mWarnStyle.Render("󰅏 Path does not exist"))
				fmt.Println("      " + mPathStyle.Render(path))
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
				fmt.Println("  " + mInfoStyle.Render("No previous target") + "  " + mMutedStyle.Render("Run Path, Clipboard, Finder, fzf, or Deemix first."))
				promptReturn()
				continue
			}
			processPathInteractive(lastTarget, s)

		case "6":
			lastTarget = deemixQuickPath
			processPathInteractive(lastTarget, s)

		case "7":
			lastTarget = soulseekQuickPath
			processPathInteractive(lastTarget, s)

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
		fmt.Println(renderPageHeader("U N D O", "󰕌", "M A N A G E", "Restore or clean Metallum backup sessions"))
		fmt.Println()
		fmt.Println("  " + renderMenuItem("1", "󰌋", "Undo Last", "Restore latest Metallum write"))
		fmt.Println("  " + renderMenuItem("2", "󰏃", "Backups", "List or clean old undo sessions"))
		fmt.Println("  " + renderMenuItem("b", "", "Back", "Return to main menu"))
		fmt.Println()
		fmt.Print(mPromptStyle.Render("❯ Choice:"))
		fmt.Print(" ")
		switch strings.ToLower(readLine()) {
		case "1":
			clearScreen()
			fmt.Println(renderPageHeader("U N D O", "󰕌", "L A S T", "Restore files from the latest Metallum write"))
			fmt.Println()
			count, err := genrenorm.RestoreLatestUndo("metallum")
			if err != nil {
				fmt.Println("   " + mErrorStyle.Render("󰅏 "+err.Error()))
			} else {
				fmt.Printf("   %s %s\n", mSuccessStyle.Render("◆"), mSuccessStyle.Render(fmt.Sprintf("Restored %d file(s)", count)))
			}
			promptReturn()
		case "2":
			manageBackups("metallum")
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
		fmt.Println("   " + mErrorStyle.Render("󰅏 "+err.Error()))
		promptReturn()
		return
	}
	for {
		clearScreen()
		fmt.Println(renderPageHeader("B A C K U P S", "󰏃", strings.ToUpper(tool), "Manage undo backup sessions"))
		fmt.Println()
		if len(sessions) == 0 {
			fmt.Println("   " + mMutedStyle.Render("No backup sessions found."))
		} else {
			fmt.Println("  " + renderSection("󰏃", "Sessions"))
			totalSize := int64(0)
			for i, s := range sessions {
				totalSize += s.SizeBytes
				fmt.Printf("   %s  %s  %s  %s  %s\n",
					mKeyStyle.Render(fmt.Sprintf("%d", i+1)),
					mRowIconStyle.Render("󰏋"),
					mActionLabelStyle.Render(s.Timestamp.Format("2006-01-02 15:04:05")),
					mCountStyle.Render(fmt.Sprintf("%d files", s.FileCount)),
					mPathStyle.Render(genrenorm.FormatBytes(s.SizeBytes)),
				)
			}
			fmt.Println()
			fmt.Printf("   %s  %s\n", mActionLabelStyle.Render("Total:"), mCountStyle.Render(fmt.Sprintf("%d sessions, %s", len(sessions), genrenorm.FormatBytes(totalSize))))
		}
		fmt.Println()
		fmt.Println("  " + mMutedStyle.Render("c clean old sessions  b back"))
		fmt.Println()
		fmt.Print(mPromptStyle.Render("❯ Choice:"))
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
	fmt.Println("   " + mActionLabelStyle.Render("The most recent session is always kept."))
	fmt.Println()
	fmt.Print(mPromptStyle.Render("❯ Sessions to keep (default 3):"))
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
		fmt.Println("   " + mMutedStyle.Render("Nothing to clean."))
		promptReturn()
		return
	}
	fmt.Println()
	fmt.Printf("   %s  %s\n", mWarnStyle.Render("⚠"), mWarnStyle.Render(fmt.Sprintf("This will remove %d old session(s).", len(sessions)-keep)))
	fmt.Print(mPromptStyle.Render("❯ Confirm? [y/N]:"))
	fmt.Print(" ")
	if !strings.EqualFold(readLine(), "y") {
		return
	}
	removed, err := genrenorm.CleanOldUndoSessions(tool, keep)
	if err != nil {
		fmt.Println()
		fmt.Println("   " + mErrorStyle.Render("󰅏 "+err.Error()))
	} else {
		fmt.Println()
		fmt.Printf("   %s %s\n", mSuccessStyle.Render("◆"), mSuccessStyle.Render(fmt.Sprintf("Removed %d session(s).", removed)))
	}
	promptReturn()
}

// ─── Main ───────────────────────────────────────────────────────

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
		if normalized(fi.artist) == norm && fi.artist != "" {
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
	modeTag := styleWarn.Render("[DRY RUN]")
	if cfg.write {
		modeTag = styleSuccess.Render("[LIVE]")
	}

	sum := &processingSummary{}
	var undoSession *genrenorm.UndoSession
	if cfg.write {
		session, err := genrenorm.StartUndoSession("metallum")
		if err != nil {
			return sum, fmt.Errorf("could not create undo backup session: %w", err)
		}
		undoSession = session
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		styleHeader.Render("Genres from Metallum"),
		"  ",
		modeTag,
	)
	fmt.Println("  " + header)
	fmt.Println("  " + renderSep(50))
	fmt.Printf("  %s %s\n", styleLabel.Render("Target:"), styleCount.Render(cfg.target))
	fmt.Printf("  %s %s\n", styleLabel.Render("Max genres:"), styleCount.Render(fmt.Sprintf("%d", cfg.maxGenres)))
	if !cfg.write {
		fmt.Println("  " + styleDim.Render("Pass --write to apply changes."))
	}
	fmt.Println()

	files, err := findMP3Files(cfg.target)
	if err != nil {
		printlnStyle(styleError, "✗ Error: %v", err)
		return sum, err
	}
	if len(files) == 0 {
		printlnStyle(styleWarn, "No MP3 files found.")
		return sum, nil
	}

	fmt.Printf("  %s %s %s\n",
		styleLabel.Render("Found"),
		styleCount.Render(fmt.Sprintf("%d", len(files))),
		styleLabel.Render("MP3 file(s) — reading artist tags..."),
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
		key := normalized(fi.artist)
		if artistAlbums[key] == nil {
			artistAlbums[key] = make(map[string]bool)
		}
		if fi.album != "" {
			artistAlbums[key][normalized(fi.album)] = true
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
		missingInfo = styleDim.Render(fmt.Sprintf(" (%d files missing artist tag)", noArtistFiles))
	}
	fmt.Printf("  %s %s %s%s\n",
		styleLabel.Render("Fetching genres from Metal Archives for"),
		styleCount.Render(fmt.Sprintf("%d", len(uniqueArtists))),
		styleLabel.Render("unique artist(s)"),
		missingInfo,
	)
	fmt.Println()

	artistCache := make(map[string][]string)
	delay := time.Duration(cfg.delayMs) * time.Millisecond
	foundCount := 0

	for i, entry := range uniqueArtists {
		artist := entry.name
		counter := styleDim.Render(fmt.Sprintf("[%d/%d]", i+1, len(uniqueArtists)))
		artistName := styleArtist.Render(artist)
		fmt.Printf("  %s %s  ", counter, artistName)

		tick := startSpinner()
		time.Sleep(delay)
		album := ""
		if len(entry.albums) > 0 {
			album = entry.albums[0]
		}
		result := searchBand(artist, album)
		stopSpinner(tick)

		if result != nil {
			genres := genrenorm.NormalizeGenres(result.Genre, cfg.maxGenres)
			artistCache[normalized(artist)] = genres
			foundCount++

			meta := ""
			if result.Country != "" {
				meta = styleDim.Render(fmt.Sprintf(" [%s]", result.Country))
			}
			fmt.Printf("%s %s%s\n",
				styleSuccess.Render("✓"),
				styleGenre.Render(strings.Join(genres, "; ")),
				meta,
			)
		} else {
			artistCache[normalized(artist)] = []string{}
			fmt.Println(styleError.Render("✗  not found on Metal Archives"))
		}
	}

	fmt.Println()
	fmt.Println("  " + renderSep(50))
	fmt.Printf("  %s %s %s %s %s\n",
		styleLabel.Render("Found genres for"),
		styleCount.Render(fmt.Sprintf("%d", foundCount)),
		styleLabel.Render("/"),
		styleCount.Render(fmt.Sprintf("%d", len(uniqueArtists))),
		styleLabel.Render("artists"),
	)
	fmt.Println()

	fmt.Println("  " + styleLabel.Render("Processing files..."))
	fmt.Println()

	for _, fi := range fileInfos {
		base := filepath.Base(fi.path)

		if fi.artist == "" {
			sum.stillMissing++
			fmt.Printf("  %s %s\n", styleDim.Render("∘"), styleDim.Render(base+" — no artist tag"))
			continue
		}
		genres := artistCache[normalized(fi.artist)]
		if len(genres) == 0 {
			sum.skipped++
			fmt.Printf("  %s %s\n",
				styleDim.Render("∘"),
				styleDim.Render(fmt.Sprintf("%s — no Metallum genre for \"%s\"", base, fi.artist)),
			)
			continue
		}
		genreValue := strings.Join(genres, "; ")

		if !cfg.write {
			sum.dryCandidates++
			fileStyle := lipgloss.NewStyle().Bold(true)
			fmt.Printf("  %s %s %s %s\n",
				styleWarn.Render("◇"),
				fileStyle.Render(base),
				styleLabel.Render("→"),
				styleGenre.Render(genreValue),
			)
			continue
		}

		if undoSession != nil {
			if err := undoSession.Backup(fi.path); err != nil {
				sum.failed++
				fmt.Printf("  %s %s\n",
					styleError.Render("✗"),
					styleError.Render(base+" — backup failed: "+err.Error()),
				)
				continue
			}
		}

		if ok := writeGenre(fi.path, genreValue); ok {
			sum.updated++
			fileStyle := lipgloss.NewStyle().Bold(true)
			fmt.Printf("  %s %s %s %s\n",
				styleSuccess.Render("✓"),
				fileStyle.Render(base),
				styleLabel.Render("→"),
				styleGenre.Render(genreValue),
			)
		} else {
			sum.failed++
			fmt.Printf("  %s %s\n",
				styleError.Render("✗"),
				styleError.Render(base+" — write failed"),
			)
		}
	}

	fmt.Println()
	fmt.Println("  " + renderHR(50))
	fmt.Println("  " + styleHeader.Render("Summary"))
	fmt.Println()

	if !cfg.write {
		fmt.Printf("  %-28s %s\n",
			styleLabel.Render("Dry-run candidates"),
			styleCount.Render(fmt.Sprintf("%d", sum.dryCandidates)),
		)
	} else {
		fmt.Printf("  %-28s %s\n",
			styleLabel.Render("Files updated"),
			styleCount.Render(fmt.Sprintf("%d", sum.updated)),
		)
	}
	if sum.skipped > 0 {
		fmt.Printf("  %-28s %s\n",
			styleLabel.Render("Skipped, no MA genre"),
			styleDim.Render(fmt.Sprintf("%d", sum.skipped)),
		)
	}
	if sum.stillMissing > 0 {
		fmt.Printf("  %-28s %s\n",
			styleLabel.Render("Skipped, no artist tag"),
			styleDim.Render(fmt.Sprintf("%d", sum.stillMissing)),
		)
	}
	if sum.failed > 0 {
		fmt.Printf("  %-28s %s\n",
			styleLabel.Render("Failed"),
			styleError.Render(fmt.Sprintf("%d", sum.failed)),
		)
	}
	if !cfg.write && sum.dryCandidates > 0 {
		fmt.Println()
		fmt.Println("  " + renderSep(50))
		fmt.Printf("  %s %s %s\n",
			styleWarn.Render("💡"),
			styleWarn.Render("Re-run with"),
			styleCount.Render("--write"),
		)
		fmt.Printf("  %s\n", styleWarn.Render("   to apply these changes."))
	}

	if sum.failed > 0 {
		return sum, fmt.Errorf("%d write failures", sum.failed)
	}
	return sum, nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			clearScreen()
			fmt.Println(renderPageHeader("E R R O R", "󰅏", "P A N I C", "An unexpected error occurred"))
			fmt.Println()
			fmt.Printf("   %s %v\n", mErrorStyle.Render("✖"), r)
			fmt.Println()
			fmt.Println("  " + mMutedStyle.Render("Press Enter to exit..."))
			readLine()
		}
	}()

	flag.BoolVar(&cfg.write, "write", false, "Actually update MP3 tags (default: dry-run)")
	flag.BoolVar(&cfg.undo, "undo", false, "Restore files from the latest write")
	flag.IntVar(&cfg.maxGenres, "max-genres", 3, "Number of final genres to write")
	flag.IntVar(&cfg.delayMs, "delay-ms", 500, "Delay between Metal Archives requests in ms")
	flag.Parse()

	if cfg.undo {
		count, err := genrenorm.RestoreLatestUndo("metallum")
		if err != nil {
			printlnStyle(styleError, "✗ %v", err)
			os.Exit(1)
		}
		printlnStyle(styleSuccess, "Restored %d file(s)", count)
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
			printlnStyle(styleError, "✗ Error: %v", err)
			os.Exit(1)
		}
		if _, err := processTarget(cfg.target); err != nil {
			printlnStyle(styleError, "✗ %v", err)
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
				printlnStyle(styleError, "✗ Error: %v", err)
				os.Exit(1)
			}
			if _, err := processTarget(cfg.target); err != nil {
				printlnStyle(styleError, "✗ %v", err)
				os.Exit(1)
			}
			return
		}
	}

	interactiveLoop()
}
