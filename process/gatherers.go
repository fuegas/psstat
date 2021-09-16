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

func splitProperty(value []byte) (string, string) {
	arr := strings.Split(string(value), "=")
	return arr[0], arr[1]
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

		id := ""
		pid := 0

		var propertyKey string
		var propertyValue string

		cmd := execCommand("systemctl", "show", pattern, "--property=MainPID", "--property=Id")
		unitLines, err := cmd.Output()
		if err != nil {
			utils.PrintError("Could not show systemd units: ", pattern, err)
			continue
		}

	Next:
		for _, unitLine := range bytes.Split(unitLines, []byte{'\n'}) {
			// Empty line means next unit
			if len(unitLine) == 0 {
				if id == "" || pid == 0 {
					utils.PrintError("Invalid ID/PID from systemd unit: ", id, pid)
				} else {
					accumulator[PID(pid)] = id
				}

				id = ""
				pid = 0
				continue
			}

			propertyKey, propertyValue = splitProperty(unitLine)

			switch propertyKey {
			// Use id from unit or override with name from flags
			case "Id":
				id = strings.ReplaceAll(propertyValue, ".service", "")
				if name != "" {
					id = name
				}
			case "MainPID":
				pid, err = strconv.Atoi(propertyValue)
				if err != nil {
					utils.PrintError("Invalid PID from systemd unit: ", unitLine)
					break Next
				}
			default:
				utils.PrintError("Invalid property from systemd unit: ", unitLine)
				break Next
			}
		}
	}
}
