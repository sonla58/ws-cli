package shell

import "fmt"

// InitScript returns the shell function wrapper for the given shell.
// Users install it with: eval "$(ws init <shell>)" in their rc file.
func InitScript(sh string) (string, error) {
	switch sh {
	case "bash", "zsh":
		return bashZsh, nil
	case "fish":
		return fish, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (bash, zsh, fish)", sh)
	}
}

const bashZsh = `# ws shell integration — eval "$(ws init zsh)" in your rc file.
ws() {
  local out
  out=$(command ws --emit-path "$@")
  local rc=$?
  if [ $rc -ne 0 ]; then
    return $rc
  fi
  if [ -n "$out" ] && [ -d "$out" ]; then
    cd "$out" || return $?
  fi
}
`

const fish = `# ws shell integration — ws init fish | source
function ws
  set -l out (command ws --emit-path $argv)
  set -l rc $status
  if test $rc -ne 0
    return $rc
  end
  if test -n "$out" -a -d "$out"
    cd $out
  end
end
`
