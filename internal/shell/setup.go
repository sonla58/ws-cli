package shell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// rcLines maps a shell name to the one-liner users add to their rc file.
// Kept in one place so AlreadyInstalled's sentinel, Install's write, and
// the `ws init <shell>` wrapper form all agree.
var rcLines = map[string]string{
	"bash": `eval "$(ws init bash)"`,
	"zsh":  `eval "$(ws init zsh)"`,
	"fish": `ws init fish | source`,
}

// Detect picks the most likely rc file to edit based on $SHELL. Returns
// (shellName, rcPath) where shellName is one of bash/zsh/fish.
func Detect() (shell, rcPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	sh := filepath.Base(os.Getenv("SHELL"))
	switch sh {
	case "zsh":
		return "zsh", filepath.Join(home, ".zshrc"), nil
	case "bash":
		// macOS login shells source .bash_profile; Linux typically uses .bashrc.
		rc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(rc); err == nil {
			return "bash", rc, nil
		}
		return "bash", filepath.Join(home, ".bash_profile"), nil
	case "fish":
		return "fish", filepath.Join(home, ".config", "fish", "config.fish"), nil
	}
	return "", "", fmt.Errorf("unsupported shell %q — run `ws init <shell>` and add the output to your rc file manually", sh)
}

// AlreadyInstalled reports whether the rc file already references `ws init`.
func AlreadyInstalled(rcPath string) (bool, error) {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(data), "ws init"), nil
}

// Install appends the integration line for sh to rcPath (creating parent
// directories if needed). Caller is responsible for checking AlreadyInstalled
// first when idempotency matters.
func Install(sh, rcPath string) error {
	line, ok := rcLines[sh]
	if !ok {
		return fmt.Errorf("unknown shell %q", sh)
	}
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, "\n# ws shell integration\n"+line+"\n")
	return err
}
