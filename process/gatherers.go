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

		cmd := execCommand("systemctl", "list-units", pattern, "--type=service", "--full", "--no-legend", "--no-pager", "--no-ask-password")
		unitListLines, err := cmd.Output()
		if err != nil {
			utils.PrintError("Could not list systemd units: ", pattern, err)
			continue
		}

		// Skip an empty list
		if len(unitListLines) == 0 {
			continue
		}

		// Iterate over found systemd units
	Next:
		for _, unitListLine := range bytes.Split(unitListLines, []byte{'\n'}) {
			// Skip empty line
			if len(unitListLine) == 0 {
				continue
			}

			// Get unit id from line
			unitId := string(bytes.SplitN(unitListLine, []byte{' '}, 2)[0])
			pid := 0

			cmd := execCommand("systemctl", "show", unitId, "--property=MainPID")
			unitLines, err := cmd.Output()
			if err != nil {
				utils.PrintError("Could not show systemd unit: ", unitId, err)
				continue
			}

			for _, unitLine := range bytes.Split(unitLines, []byte{'\n'}) {
				// Skip empty line
				if len(unitLine) == 0 {
					continue
				}

				// Get pid part from line
				pidStr := string(bytes.SplitN(unitLine, []byte{'='}, 2)[1])

				// Skip empty value and illegal value
				if len(pidStr) == 0 || pidStr == "0" {
					continue
				}

				pid, err = strconv.Atoi(pidStr)
				if err != nil {
					utils.PrintError("Invalid PID from systemd unit: ", unitId, pidStr)
					break Next
				}
			}

			// Use id from unit or override with name from flags
			id := strings.ReplaceAll(unitId, ".service", "")
			if name != "" {
				id = name
			}

			// Only store if we have a pid
			if pid > 0 {
				accumulator[PID(pid)] = id
			}
		}
	}
}
