package main

import (
	"fmt"
	flag "github.com/ogier/pflag"
	"os"
	"strings"
	"syscall"
	"time"
)

var verbose = flag.BoolP("verbose", "v", false, "verbose")
var timeout = flag.DurationP("timeout", "T", 5*time.Second, "timeout (ex: 100ms or 3s)")
var printpath = flag.BoolP("path", "p", true, "show path")
var printdev = flag.BoolP("dev", "d", false, "show device")
var quiet = flag.BoolP("quiet", "q", false, "quiet mode, only set exit status")
var checktype = flag.StringP("type", "t", "nfs", "filesystem type to check")
var checkall = flag.BoolP("all", "a", false, "check all filesystems types")

type fs_t struct {
	path     string
	dev      string
	resptime time.Duration
	done     bool
	err      error
}

type fsmap_t map[string]fs_t

func (fs fs_t) String() (s string) {
	switch {
	case fs.err != nil:
		if strings.Index(fs.err.Error(), "stale") >= 0 {
			s += fmt.Sprintf("%-9s", "STALE")
		} else {
			s += fmt.Sprintf("%-9s", "ERROR")
		}
	case !fs.done:
		s += fmt.Sprintf("%-9s", "TIMEOUT")
	default:
		s += fmt.Sprintf("%-9s", fs.resptime)
	}
	if *printpath {
		s += fmt.Sprintf(" %-20s", fs.path)
	}
	if *printdev {
		s += fmt.Sprintf(" %-30s", fs.dev)
	}
	if fs.err != nil {
		s += fmt.Sprintf(" %s", fs.err)
	}
	return s
}

func statit(fs fs_t, ch chan fs_t) {
	var buf syscall.Statfs_t
	start := time.Now()
	fs.err = syscall.Statfs(fs.path, &buf)
	fs.resptime = time.Since(start)
	fs.done = true
	ch <- fs
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "fsping: check responsiveness of mounted filesystems\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	fsnum := 0
	fsch := make(chan fs_t)

	fsmap := readmounts()

	args := flag.Args()

	// if extra args were specified keep only filesystems listed
	if len(args) > 0 {
	L:
		for fs, _ := range fsmap {
			for _, a := range args {
				if fs == a {
					continue L
				}
			}
			delete(fsmap, fs)
		}
	}

	// fire them up
	for _, fs := range fsmap {
		fsnum += 1
		// fire off a goroutine for each mounted nfs
		// filesystem. fsch is used for completion
		// notifications
		go statit(fs, fsch)
	}

	// timeout ticker
	tick := time.Tick(*timeout)

	// collect results
	for fsnum > 0 {
		select {
		case fs := <-fsch:
			if *verbose || fs.err != nil {
				fmt.Println(fs)
			}
			fsmap[fs.path] = fs
			fsnum -= 1
		case <-tick:
			// ticker has kicked in, print timeout
			// messages for filesystems that haven't
			// finished
			for _, fs := range fsmap {
				if !*quiet && !fs.done {
					fmt.Println(fs)
				}
			}
			os.Exit(1)
		}
	}
}
