package clipboard

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aymanbagabas/go-osc52/v2"
)

// Clipboard provides an abstraction over system clipboard
type Clipboard struct {
	internal string
	primary  bool // Use primary selection on Linux
}

// New creates a new clipboard manager
func New() *Clipboard {
	return &Clipboard{
		primary: runtime.GOOS == "linux",
	}
}

// Copy writes text to clipboard
func (c *Clipboard) Copy(text string) {
	c.internal = text

	// Try OSC52 for terminal-based clipboard (SSH support)
	seq := osc52.New(text)
	fmt.Print(seq)

	// Try system clipboard as fallback
	c.copyToSystem(text)
}

// Paste reads text from clipboard
func (c *Clipboard) Paste() string {
	// First try system clipboard
	if text := c.pasteFromSystem(); text != "" {
		return text
	}

	// Fall back to internal clipboard
	return c.internal
}

// Cut copies text and returns it (caller should clear selection)
func (c *Clipboard) Cut(text string) string {
	c.Copy(text)
	return text
}

// copyToSystem attempts to copy to system clipboard
func (c *Clipboard) copyToSystem(text string) {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			log.Printf("pbcopy failed: %v", err)
		}
	case "linux":
		// Try wl-copy first (Wayland)
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return
		}

		// Fall back to xclip (X11)
		if c.primary {
			cmd = exec.Command("xclip", "-selection", "primary", "-in")
			cmd.Stdin = strings.NewReader(text)
			cmd.Run()
		}
		cmd = exec.Command("xclip", "-selection", "clipboard", "-in")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			log.Printf("xclip failed: %v", err)
		}
	case "windows":
		if err := exec.Command("powershell", "-command", "Set-Clipboard", text).Run(); err != nil {
			log.Printf("Set-Clipboard failed: %v", err)
		}
	}
}

// pasteFromSystem attempts to read from system clipboard
func (c *Clipboard) pasteFromSystem() string {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("pbpaste").Output()
		if err == nil {
			return string(out)
		}
		log.Printf("pbpaste failed: %v", err)
	case "linux":
		// Try wl-paste first (Wayland)
		out, err := exec.Command("wl-paste", "--no-newline").Output()
		if err == nil {
			return string(out)
		}

		// Fall back to xclip (X11)
		out, err = exec.Command("xclip", "-selection", "clipboard", "-out").Output()
		if err == nil {
			return string(out)
		}

		if c.primary {
			out, err = exec.Command("xclip", "-selection", "primary", "-out").Output()
			if err == nil {
				return string(out)
			}
		}

		// Last resort: try xsel
		out, err = exec.Command("xsel", "--clipboard", "--output").Output()
		if err == nil {
			return string(out)
		}

		log.Printf("All clipboard tools failed. wl-paste: %v, xclip: %v, xsel: %v", err, err, err)
	case "windows":
		out, err := exec.Command("powershell", "-command", "Get-Clipboard").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
		log.Printf("Get-Clipboard failed: %v", err)
	}
	return ""
}

// Clear removes internal clipboard content
func (c *Clipboard) Clear() {
	c.internal = ""
}
