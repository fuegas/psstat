package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fuegas/psstat/cache"
	. "github.com/fuegas/psstat/process"
	"github.com/fuegas/psstat/utils"
)

const usage = `psstat, gather resource usage of processes

Usage:

  psstat [options]

The options are:

  --pid [<name>:]<pid-number>     PID of the process to gather stats of
  --pid-file [<name>:]<pid-file>  File containing a PID of a process to gather stats of
  --pattern [<name>:]<pattern>    Pattern to find a process with (all matches are used)
  --systemd [<name>:]<pattern>    Systemd unit to find a process with (all matches are used)

  --tag <name>=<value>    Add a tag to the output (for example: env=production)

  --cache-dir <name>      Name of the directory to store the stats cache (default: /mnt/psstat)
  --cache-name <name>     Name of the file to store the stats cache (in cache-dir)

  --multi-threaded        Run process gathering in multiple threads, this can cause strange load spikes on servers with many processes.

  --help                  Show this description
  --version               Show the current version

  When you specify a name, that name will be used as value for the process_name tag.
  If you do not provide it, it will determine the name from the process information.

Examples:

  # Get stats of a process
  psstat --pid kernel:1

  # Get stats using a pid-file
  psstat --pid-file /var/run/nginx.pid

  # Get stats using a pattern
  psstat --pattern mysqld:mysqld_safe
`

func exitWithUsage(code int) {
	fmt.Println(usage)
	os.Exit(code)
}

func exitError(msgs ...interface{}) {
	utils.PrintError(msgs...)
	os.Exit(1)
}

type ArrayFlags []string

var fPids ArrayFlags
var fPidFiles ArrayFlags
var fPatterns ArrayFlags
var fSystemds ArrayFlags

var fTags ArrayFlags

var fCacheDir = flag.String("cache-dir", "/tmp",
	"store cache data in a file in this directory")
var fCacheName = flag.String("cache-name", "psstat",
	"store cache data in a file with this name")

var fMultiThreaded = flag.Bool("multi-threaded", false,
	"Run process gathering on a multiple threads")

var fVersion = flag.Bool("version", false, "Show the current version")

var (
	version string
	commit  string
	branch  string
)

// Set defaults if no values were passed
func init() {
	if commit == "" {
		commit = "unknown"
	}
	if branch == "" {
		branch = "unknown"
	}
}

func (p *ArrayFlags) String() string {
	return "string representation"
}

func (p *ArrayFlags) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func main() {
	// Limit number of procs to prevent high load notices when threads are
	// spawned to index /proc. For more information see:
	// https://blog.avast.com/investigation-of-regular-high-load-on-unused-machines-every-7-hours
	if runtime.NumCPU() > 4 {
		runtime.GOMAXPROCS(4)
	}

	var err error
	// Parse arguments
	flag.Usage = func() { exitWithUsage(0) }
	flag.Var(&fPids, "pid", "pid")
	flag.Var(&fPidFiles, "pid-file", "file containing a pid")
	flag.Var(&fPatterns, "pattern",
		"pattern to find a process with (all matches are used)")
	flag.Var(&fSystemds, "systemd",
		"name of systemd unit to find a process with (all matches are used)")
	flag.Var(&fTags, "tag", "tag to add to the output")
	flag.Parse()

	// Check that no unknown flags were passed
	args := flag.Args()
	if len(args) > 0 {
		utils.PrintError("Unknown options passed:", args)
		exitWithUsage(1)
	}

	// Show version if requested
	if *fVersion {
		if version == "" {
			fmt.Printf("experimental @ %s on %s\n", commit, branch)
		} else {
			fmt.Printf("v%s\n", version)
		}
		return
	}

	// Determine tags
	tagsMap := make(map[string]string)
	for _, tagFlag := range fTags {
		arr := strings.SplitN(tagFlag, "=", 2)
		if len(arr) != 2 {
			utils.PrintError("Provided tag contains more than one = character: ", tagFlag)
			exitWithUsage(1)
		}

		tagsMap[arr[0]] = arr[1]
	}

	tags := new(bytes.Buffer)
	for key, value := range tagsMap {
		fmt.Fprintf(tags, ",%s=%s", key, value)
	}

	// Get number of CPUs
	numCpus := float64(runtime.NumCPU())

	// Total memory
	memTotal, err := TotalMemory()
	if err != nil {
		exitError("Failed to gather total memory", err)
	}

	// Aggregate all known processes
	currentTime := time.Now().UnixNano()
	var procs map[PID]*Process
	procs, err = GatherAllProcs(*fMultiThreaded)
	if err != nil {
		exitError("Failed to gather process information:", err)
	}

	// Prepare pid list
	pids := make(map[PID]string)
	PidsFromPidFlags(fPids, pids)
	PidsFromPidFileFlags(fPidFiles, pids)
	PidsFromPatternFlags(fPatterns, pids, procs)
	PidsFromSystemdFlags(fSystemds, pids)

	// Load cache of previous measurements
	cache := cache.NewCache(filepath.Join(*fCacheDir, *fCacheName))
	prevTime, prevProcs := cache.Read(procs)

	// Write new cache
	err = cache.Write(int64(currentTime), procs)
	if err != nil {
		utils.PrintError(err)
	}

	var prevCpuTime float64

	// Iterate over processes
	for pid, name := range pids {
		proc := procs[pid]

		// Skip processes that we can't find
		if proc == nil {
			continue
		}

		// Set name if it was not provided
		if name == "" {
			name = proc.Name
		}

		// Calculate previous cpu time with the current process as backup
		// Check if the PID is present in the prevProcs map
		if _, ok := prevProcs[pid]; ok {
			prevCpuTime, _ = prevProcs[pid].SumResources(prevProcs)
		} else {
			prevCpuTime, _ = procs[pid].SumResources(prevProcs)
		}

		// Calculate current cpu time
		cpuTime, memUsed := procs[pid].SumResources(procs)

		// Calculate Cpu percentage
		deltaTime := float64(currentTime-prevTime) / 1000000000 * numCpus
		cpuPerc := float64(cpuTime-prevCpuTime) / deltaTime * numCpus
		memPerc := float64(memUsed) / float64(memTotal)

		fmt.Printf(
			"psstat%s,process_name=%s pcpu=%.3f,pmem=%.3f,n_proc=%di\n",
			tags.String(), utils.Escape(name), cpuPerc, memPerc, proc.Processes)
	}
}
