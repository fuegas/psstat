package process

import (
	"bytes"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fuegas/psstat/utils"
)

func splitFlag(value string) (string, string) {
	arr := strings.Split(value, ":")
	if len(arr) == 1 {
		return arr[0], ""
	} else {
		return arr[1], arr[0]
	}
}

// Loop through PID's and and split them to get the PID and name
func PidsFromPidFlags(flags []string, accumulator map[PID]string) {
	for _, pidFlag := range flags {
		pidStr, name := splitFlag(pidFlag)

		// Convert PID to an integer
		pidInt, err := strconv.Atoi(pidStr)
		if err != nil {
			utils.PrintError("Invalid PID number provided: ", pidStr, err)
			continue
		}

		accumulator[PID(pidInt)] = name
	}
}

// Loop through PID files to get the PID
func PidsFromPidFileFlags(flags []string, accumulator map[PID]string) {
	for _, pidFileFlag := range flags {
		pidFile, name := splitFlag(pidFileFlag)

		// Try to read the PID file
		pidStr, err := ioutil.ReadFile(pidFile)
		if err != nil {
			utils.PrintError("Could not read pidfile: ", pidFile)
			continue
		}

		// Convert PID string to an int
		pidInt, err := strconv.Atoi(strings.TrimSpace(string(pidStr)))
		if err != nil {
			utils.PrintError("Invalid PID provided: ", pidStr, err)
			continue
		}

		accumulator[PID(pidInt)] = name
	}
}

// Loop through process names and add all matches to the accumulator
func PidsFromPatternFlags(flags []string, accumulator map[PID]string, procs map[PID]*Process) {
	for _, patternFlag := range flags {
		pattern, name := splitFlag(patternFlag)

		// Loop through the processes to check if the process name matches the pattern
		for pid, proc := range procs {
			matched, err := regexp.MatchString(pattern, proc.Name)
			if matched && err == nil {
				accumulator[pid] = name
			}
		}
	}
}

// Allow mocking of exec.Command for possible tests
var execCommand = exec.Command

// Retrieve PID from Systemd and add it to the accumulator
func PidsFromSystemdFlags(flags []string, accumulator map[PID]string) {
	for _, systemdFlag := range flags {
		pattern, name := splitFlag(systemdFlag)

		cmd := execCommand("systemctl", "show", pattern)
		output, err := cmd.Output()
		if err != nil {
			utils.PrintError("Could not retrieve systemd unit: ", pattern, err)
			continue
		}

		// Iterate over lines to find MainPID
	Next: // Label to skip to next pattern
		for _, line := range bytes.Split(output, []byte{'\n'}) {
			// Split lines on '=' to split key from value
			parts := bytes.SplitN(line, []byte{'='}, 2)

			// If we have more or less than 2 parts, something is wrong
			// For example, a line without an '='
			if len(parts) != 2 {
				continue
			}

			// Skip if the key is not MainPID
			if !bytes.Equal(parts[0], []byte("MainPID")) {
				continue
			}

			// Skip to next pattern if value is 0
			if len(parts[1]) == 0 || bytes.Equal(parts[1], []byte("0")) {
				continue Next
			}

			pid, err := strconv.Atoi(string(parts[1]))
			if err != nil {
				utils.PrintError("Invalid PID from systemd unit: ", pattern, parts[1])
				continue Next
			}

			accumulator[PID(pid)] = name
		}
	}
}
