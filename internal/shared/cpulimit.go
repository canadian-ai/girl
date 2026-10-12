package shared

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ApplyCPULimit lowers GOMAXPROCS to the container CPU quota. Go 1.25 does
// this itself on many hosts but not on every cgroup v1 layout (for example
// cpu and cpuacct mounted separately), where a 1-CPU container would run one
// analysis worker per host core and thrash. An explicit GOMAXPROCS wins.
func ApplyCPULimit() {
	if os.Getenv("GOMAXPROCS") != "" {
		return
	}
	if limit, ok := cgroupCPULimit("/"); ok && limit < runtime.GOMAXPROCS(0) {
		runtime.GOMAXPROCS(limit)
	}
}

// cgroupCPULimit returns the CPU quota of the current cgroup rounded up to
// whole CPUs, reading files under root (normally "/").
func cgroupCPULimit(root string) (int, bool) {
	paths := selfCgroupPaths(filepath.Join(root, "proc", "self", "cgroup"))
	sys := filepath.Join(root, "sys", "fs", "cgroup")

	// cgroup v2: "<quota> <period>" or "max <period>".
	if p, ok := paths[""]; ok {
		for _, dir := range []string{filepath.Join(sys, p), sys} {
			if data, err := os.ReadFile(filepath.Join(dir, "cpu.max")); err == nil {
				fields := strings.Fields(string(data))
				if len(fields) == 2 && fields[0] != "max" {
					return quotaCPUs(fields[0], fields[1])
				}
				return 0, false
			}
		}
	}
	// cgroup v1: cpu.cfs_quota_us / cpu.cfs_period_us, quota -1 = unlimited.
	if p, ok := paths["cpu"]; ok {
		for _, mount := range []string{"cpu", "cpu,cpuacct", "cpuacct,cpu"} {
			for _, dir := range []string{filepath.Join(sys, mount, p), filepath.Join(sys, mount)} {
				quota, err1 := os.ReadFile(filepath.Join(dir, "cpu.cfs_quota_us"))
				period, err2 := os.ReadFile(filepath.Join(dir, "cpu.cfs_period_us"))
				if err1 == nil && err2 == nil {
					return quotaCPUs(strings.TrimSpace(string(quota)), strings.TrimSpace(string(period)))
				}
			}
		}
	}
	return 0, false
}

// selfCgroupPaths maps controller name to cgroup path; the v2 unified
// hierarchy is keyed by "".
func selfCgroupPaths(file string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(file)
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[1] == "" {
			out[""] = parts[2]
			continue
		}
		for _, controller := range strings.Split(parts[1], ",") {
			out[controller] = parts[2]
		}
	}
	return out
}

func quotaCPUs(quota, period string) (int, bool) {
	q, err1 := strconv.ParseFloat(quota, 64)
	p, err2 := strconv.ParseFloat(period, 64)
	if err1 != nil || err2 != nil || q <= 0 || p <= 0 {
		return 0, false
	}
	return int(math.Max(1, math.Ceil(q/p))), true
}
