package utils

import (
	"fmt"
	"os"
)

func PrintError(msgs ...interface{}) {
	fmt.Fprint(os.Stderr, "E!")
	fmt.Fprintln(os.Stderr, msgs...)
}
