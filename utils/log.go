package utils

import (
	"fmt"
	"os"
	"regexp"
)

func PrintError(msgs ...interface{}) {
	fmt.Fprint(os.Stderr, "E! ")
	fmt.Fprintln(os.Stderr, msgs...)
}

func Escape(val string) string {
	re := regexp.MustCompile(`([\s=,])`)
	return re.ReplaceAllString(val, "\\$1")
}
