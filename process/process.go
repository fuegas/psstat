package process

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ClockTicks = 100
)

var (
	PageSize = int64(os.Getpagesize())
)

type PID int

type Process struct {
	Pid        PID
	Name       string
	Parent     PID
	UserTime   float64
	SystemTime float64
	MemoryUsed int64
	Processes  int
}

// Gathers all processes in /proc and creates a map of PID to Process
//
// When it builds the map, the relevant information of a Process is read from
// stat and statm
func GatherAllProcs(multiThreaded bool) (map[PID]*Process, error) {
	if multiThreaded {
		return GatherAllProcsMultiThreaded()
	} else {
		return GatherAllProcsSingleThreaded()
	}
}

// Does what GatherAllProcs claims, but single threaded
func GatherAllProcsSingleThreaded() (map[PID]*Process, error) {
	procs := make(map[PID]*Process)

	// Find all pids
	files, err := ioutil.ReadDir(procPath())
	if err != nil {
		return nil, err
	}

	// Iterate over processes
	for _, file := range files {
		pid, err := strconv.Atoi(file.Name())
		if file.IsDir() && err == nil {
			proc, err := NewProcess(PID(pid))
			if err == nil {
				procs[proc.Pid] = proc
			}
		}
	}

	return procs, nil
}

// Does what GatherAllProcs claims, but multi threaded
func GatherAllProcsMultiThreaded() (map[PID]*Process, error) {
	procs := make(map[PID]*Process)
	done := make(chan *Process)

	// Find all pids
	files, err := ioutil.ReadDir(procPath())
	if err != nil {
		return nil, err
	}

	// Iterate over processes
	for _, file := range files {
		go func(file os.FileInfo) {
			pid, err := strconv.Atoi(file.Name())
			if file.IsDir() && err == nil {
				proc, err := NewProcess(PID(pid))
				if err == nil {
					done <- proc
				} else {
					done <- nil
				}
			} else {
				done <- nil
			}
		}(file)
	}

	// Wait for all processes to finish and build map from them
	for _ = range files {
		proc := <-done
		if proc != nil {
			procs[proc.Pid] = proc
		}
	}

	return procs, nil
}

// Create a new Process struct and fill it with the correct values
func NewProcess(pid PID) (*Process, error) {
	p := &Process{Pid: pid, Processes: 1}
	err := p.ParseStat()
	if err == nil {
		err = p.ParseStatm()
	}
	return p, err
}

// Build the path for a file in /proc
func procPath(parts ...string) string {
	parts = append([]string{"/proc"}, parts...)
	return filepath.Join(parts...)
}

// Slimmed down version of the function fillFromStat of
// github.com/shirou/gopsutil/process/process_linux.go
func (p *Process) ParseStat() error {
	// Get contents of stat file
	statPath := procPath(strconv.Itoa(int(p.Pid)), "stat")
	contents, err := ioutil.ReadFile(statPath)
	if err != nil {
		return err
	}
	stats := strings.Fields(string(contents))

	// Determine start of proc mapping
	// First item is the pid, so we can start looking from 1
	start := 1
	for !strings.HasSuffix(stats[start], ")") {
		start++
	}

	// Name (string surrounded by ())
	name := strings.Join(stats[1:start+1], " ")
	p.Name = name[1 : len(name)-1]

	// Parent PID
	ppid, err := strconv.ParseInt(stats[start+2], 10, 32)
	if err != nil {
		return err
	}
	p.Parent = PID(ppid)

	// User time
	usertime, err := strconv.ParseFloat(stats[start+12], 64)
	if err != nil {
		return err
	}
	p.UserTime = usertime / ClockTicks

	// System time
	systemtime, err := strconv.ParseFloat(stats[start+13], 64)
	if err != nil {
		return err
	}
	p.SystemTime = systemtime / ClockTicks

	return nil
}

func (p *Process) ParseStatm() error {
	// Get contents of statm file
	statPath := procPath(strconv.Itoa(int(p.Pid)), "statm")
	contents, err := ioutil.ReadFile(statPath)
	if err != nil {
		return err
	}
	stats := strings.Fields(string(contents))

	// Only interested in RSS (memory usage)
	rss, err := strconv.ParseInt(stats[1], 10, 32)
	if err != nil {
		return err
	}
	p.MemoryUsed = rss * PageSize

	return nil
}

func (p *Process) SumResources(procs map[PID]*Process) (float64, int64) {
	times := p.UserTime + p.SystemTime
	memory := p.MemoryUsed

	// Loop through all processes and add up user/system time and memory
	for _, proc := range procs {
		// Check if process is a child
		if proc.Parent == p.Pid {
			times += proc.UserTime + proc.SystemTime
			memory += proc.MemoryUsed
			p.Processes++
		}
	}

	return times, memory
}
