package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const appWidth = 62

var (
	borderStyle      lipgloss.Style
	titleStyle       lipgloss.Style
	titleIconStyle   lipgloss.Style
	descAccentStyle  lipgloss.Style
	sectionStyle     lipgloss.Style
	sectionIconStyle lipgloss.Style
	keyStyle         lipgloss.Style
	rowIconStyle     lipgloss.Style
	actionLabelStyle lipgloss.Style
	menuDescStyle    lipgloss.Style
	sepStyle         lipgloss.Style
	promptStyle      lipgloss.Style
	errorStyle       lipgloss.Style
	mutedStyle       lipgloss.Style
)

func init() {
	ld := func(dark, light string) lipgloss.Color {
		if lipgloss.HasDarkBackground() {
			return lipgloss.Color(dark)
		}
		return lipgloss.Color(light)
	}

	borderStyle = lipgloss.NewStyle().Foreground(ld("#c74ded", "#7c3aed"))
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(ld("#d5ced9", "#1e1e1e"))
	titleIconStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffe66d"))
	descAccentStyle = lipgloss.NewStyle().Foreground(ld("#7cb7ff", "#3b82f6"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(ld("#c74ded", "#7c3aed"))
	sectionIconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff32aa"))
	keyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff32aa"))
	rowIconStyle = lipgloss.NewStyle().Foreground(ld("#00e8c6", "#0891b2"))
	actionLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(ld("#d5ced9", "#1e1e1e"))
	menuDescStyle = lipgloss.NewStyle().Foreground(ld("#a1a7cc", "#6b7280"))
	sepStyle = lipgloss.NewStyle().Foreground(ld("#a1a7cc", "#6b7280"))
	promptStyle = lipgloss.NewStyle().Bold(true).Foreground(ld("#f39c12", "#d97706"))
	errorStyle = lipgloss.NewStyle().Foreground(ld("#f92a72", "#e11d48"))
	mutedStyle = lipgloss.NewStyle().Foreground(ld("#a1a7cc", "#6b7280"))
}

var reader = bufio.NewReader(os.Stdin)

func readLine() string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func centerText(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func renderHeader() string {
	title := titleStyle.Render("G E N R E") + "  " + titleIconStyle.Render("") + "  " + titleStyle.Render("T O O L K I T")
	desc := descAccentStyle.Render("         Fetch, normalize, and write MP3 genre tags           ")
	return strings.Join([]string{
		borderStyle.Render("               ╭────────────────────────────────╮"),
		borderStyle.Render("╭──────────────┤") + "   " + title + "  " + borderStyle.Render("├──────────────╮"),
		borderStyle.Render("│              ╰────────────────────────────────╯              │"),
		borderStyle.Render("│") + desc + borderStyle.Render("│"),
		borderStyle.Render("├──────────────────────────────────────────────────────────────┤"),
	}, "\n") + "\n"
}

func optionText(icon, key, rest string) string {
	return rowIconStyle.Render(icon) + "  " + keyStyle.Render(key) + actionLabelStyle.Render(rest)
}

func renderMenu() {
	fmt.Print(renderHeader())
	lastfm := optionText("", "L", "ast.fm")
	metallum := optionText("", "M", "etallum")
	spotify := optionText("", "S", "potify")
	discogs := optionText("󰋙", "D", "iscogs")
	undo := optionText("󰕌", "U", "ndo")
	quit := optionText("", "Q", "uit")
	fmt.Println(borderStyle.Render("│                                                              │"))
	fmt.Println(borderStyle.Render("│") + "               " + lastfm + sepStyle.Render("     •      ") + metallum + "              " + borderStyle.Render("│"))
	fmt.Println(borderStyle.Render("│") + "               " + spotify + sepStyle.Render("     •      ") + discogs + "             " + borderStyle.Render("│"))
	fmt.Println(borderStyle.Render("│  ") + sepStyle.Render("──────────────────────────────────────────────────────────") + borderStyle.Render("  │"))
	fmt.Println(borderStyle.Render("│") + "                 " + undo + sepStyle.Render("      •      ") + quit + "                  " + borderStyle.Render("│"))
	fmt.Println(borderStyle.Render("│                                                              │"))
	fmt.Println(borderStyle.Render("╰───────────────────────────╮       ╭──────────────────────────╯"))
}

func renderPrompt() {
	fmt.Print(strings.Repeat(" ", 32))
}

func closePromptBox() {
	fmt.Println(strings.Repeat(" ", 28) + borderStyle.Render("╰───────╯"))
}

func moduleRoot() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Clean(filepath.Join(dir, "..")),
			filepath.Clean(filepath.Join(dir, "../..")),
		}
		for _, c := range candidates {
			if st, err := os.Stat(filepath.Join(c, "go.mod")); err == nil && !st.IsDir() {
				return c
			}
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go-genres")
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func lastfmBinary() string {
	root := moduleRoot()
	return filepath.Join(root, "lastfm", "bin", "lastfm-genres")
}

func metallumBinary() string {
	root := moduleRoot()
	return filepath.Join(root, "metallum", "bin", "metallum-genres")
}

func spotifyBinary() string {
	root := moduleRoot()
	return filepath.Join(root, "spotify", "bin", "spotify-genres")
}

func discogsBinary() string {
	root := moduleRoot()
	return filepath.Join(root, "discogs", "bin", "discogs-genres")
}

func runTool(name, path string, args ...string) {
	if path == "" {
		clearScreen()
		fmt.Println(renderHeader())
		fmt.Println()
		fmt.Println("  " + errorStyle.Render("✖ "+name+" binary not found."))
		fmt.Println("  " + mutedStyle.Render("Build it first:"))
		switch name {
		case "Last.fm":
			fmt.Println("  " + menuDescStyle.Render("bash lastfm/build.sh"))
		case "Metallum":
			fmt.Println("  " + menuDescStyle.Render("bash metallum/build.sh"))
		case "Spotify":
			fmt.Println("  " + menuDescStyle.Render("bash spotify/build.sh"))
		case "Discogs":
			fmt.Println("  " + menuDescStyle.Render("bash discogs/build.sh"))
		}
		fmt.Println()
		fmt.Println("  " + mutedStyle.Render("Press Enter to return..."))
		readLine()
		return
	}

	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		clearScreen()
		fmt.Println(renderHeader())
		fmt.Println()
		if _, ok := err.(*exec.ExitError); ok {
			fmt.Println("  " + errorStyle.Render("✖ "+name+" exited with an error."))
		} else {
			fmt.Println("  " + errorStyle.Render("✖ Failed to launch "+name+": "+err.Error()))
		}
		fmt.Println()
		fmt.Println("  " + mutedStyle.Render("Press Enter to return..."))
		readLine()
	}
}

func undoMenu() {
	clearScreen()
	fmt.Println(renderHeader())
	fmt.Println()
	fmt.Println("  " + optionText("", "L", "ast.fm") + "  " + menuDescStyle.Render("Restore latest Last.fm write"))
	fmt.Println("  " + optionText("", "M", "etallum") + "  " + menuDescStyle.Render("Restore latest Metallum write"))
	fmt.Println("  " + optionText("", "S", "potify") + "  " + menuDescStyle.Render("Restore latest Spotify write"))
	fmt.Println("  " + optionText("󰋙", "D", "iscogs") + "  " + menuDescStyle.Render("Restore latest Discogs write"))
	fmt.Println()
	fmt.Print(promptStyle.Render("❯ Choice:"))
	fmt.Print(" ")
	switch strings.ToLower(readLine()) {
	case "l", "1":
		runTool("Last.fm", lastfmBinary(), "--undo")
	case "m", "2":
		runTool("Metallum", metallumBinary(), "--undo")
	case "s", "3":
		runTool("Spotify", spotifyBinary(), "--undo")
	case "d", "4":
		runTool("Discogs", discogsBinary(), "--undo")
	}
}

func main() {
	for {
		clearScreen()
		renderMenu()
		renderPrompt()
		choice := strings.ToLower(readLine())
		closePromptBox()
		switch choice {
		case "l", "1":
			runTool("Last.fm", lastfmBinary())
		case "m", "2":
			runTool("Metallum", metallumBinary())
		case "s", "3":
			runTool("Spotify", spotifyBinary())
		case "d", "4":
			runTool("Discogs", discogsBinary())
		case "q":
			return
		case "u":
			undoMenu()
		}
	}
}
