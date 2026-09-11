package helper

import "github.com/fatih/color"

func FormatBool(value bool) string {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if value {
		return green("✓")
	}

	return red("✗")
}
