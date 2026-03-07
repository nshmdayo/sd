package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	errColor  = color.New(color.FgRed, color.Bold)
	hintColor = color.New(color.FgYellow)
	okColor   = color.New(color.FgGreen)
	dimColor  = color.New(color.Faint)
)

// SetColor enables or disables color output globally.
func SetColor(enabled bool) {
	color.NoColor = !enabled
}

// Errorf prints an error message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, errColor.Sprint("Error: ")+format+"\n", args...)
}

// Hintf prints a hint message to stderr.
func Hintf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, hintColor.Sprint("Hint:  ")+format+"\n", args...)
}

// Infof prints an informational message to stderr.
func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Successf prints a success message to stderr.
func Successf(format string, args ...any) {
	fmt.Fprint(os.Stderr, okColor.Sprintf(format+"\n", args...))
}

// Dimf prints a dim/muted message to stderr.
func Dimf(format string, args ...any) {
	fmt.Fprint(os.Stderr, dimColor.Sprintf(format+"\n", args...))
}

// Path writes the resolved path to stdout so the shell wrapper can capture it.
func Path(p string) {
	fmt.Println(p)
}
