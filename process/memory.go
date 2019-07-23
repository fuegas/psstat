package process

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func TotalMemory() (int64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		// Read a line from the file
		line, err := reader.ReadString('\n')
		if err != nil {
			// Probably EOF
			return 0, nil
		}

		// Split the line into fields on whitespaces
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if fields[0] == "MemTotal:" {
			value, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, err
			}

			// Value is in Kb, so multiply by 1024
			return int64(value * 1024), nil
		}
	}
}
