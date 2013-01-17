package main

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func readmounts() fsmap_t {
	fsmap := make(fsmap_t)

	f, err := os.Open("/proc/mounts")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	b := bufio.NewReader(f)
	for {
		line, err := b.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		fields := strings.Split(line, " ")
		dev, path, fstype := fields[0], fields[1], fields[2]
		if *checkall || fstype == *checktype {
			fsmap[path] = fs_t{path: path, dev: dev}
		}
	}
	return fsmap
}
