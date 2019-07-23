package cache

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/fuegas/psstat/process"
)

// Cache to retrieve and store previous stats
//
// Cache is written to provided cachePath. The file contains the time the
// statistics were gathered and and the values of a Process struct.

type Cache struct {
	Path string
}

func NewCache(cachePath string) *Cache {
	return &Cache{Path: cachePath}
}

func (c *Cache) Read(procs map[PID]*Process) (int64, map[PID]*Process) {
	var cachedProcs = make(map[PID]*Process)

	var pid int64
	var parent int64

	// Open the cache file
	file, err := os.Open(c.Path)
	if err != nil {
		return 0, cachedProcs
	}
	defer file.Close()

	// Use a scanner to read all the lines
	scanner := bufio.NewScanner(bufio.NewReader(file))

	// First line contains the time when the stats were gathered
	// So if we can't read a first line, we stop
	ok := scanner.Scan()
	if !ok {
		return 0, cachedProcs
	}

	// Get the values from the array and parse them to the correct type
	time, err := strconv.ParseInt(scanner.Text(), 10, 64)
	if err != nil {
		return 0, cachedProcs
	}

	// Loop over all lines
	for scanner.Scan() {
		// Split the line on commas
		parts := strings.Split(scanner.Text(), ",")

		// Skip corrupt lines
		if len(parts) != 5 {
			continue
		}

		// Get the pid from the array to check if it still exists
		pid, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			pid = 0
		}

		// Skip PID if it does not exist in procs
		if _, ok := procs[PID(pid)]; !ok {
			continue
		}

		// Get the values from the array and parse them to the correct type
		parent, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			parent = 0
		}

		proc := &Process{
			Pid:       PID(pid),
			Parent:    PID(parent),
			Processes: 1,
		}

		proc.UserTime, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			proc.UserTime = 0
		}

		proc.SystemTime, err = strconv.ParseFloat(parts[3], 64)
		if err != nil {
			proc.SystemTime = 0
		}

		proc.MemoryUsed, err = strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			proc.MemoryUsed = 0
		}

		// Add Process to cachedProcs
		cachedProcs[PID(pid)] = proc
	}

	return time, cachedProcs
}

func (c *Cache) Write(time int64, procs map[PID]*Process) error {
	file, err := os.Create(c.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// Start with time of gathering
	fmt.Fprintf(writer, "%d\n", time)

	// Loop over procs and add the fields to the buffer
	for _, process := range procs {
		fmt.Fprintf(
			writer,
			"%d,%d,%f,%f,%d\n",
			process.Pid,
			process.Parent,
			process.UserTime,
			process.SystemTime,
			process.MemoryUsed)
	}

	// Write the contents to the cache file
	return writer.Flush()
}
