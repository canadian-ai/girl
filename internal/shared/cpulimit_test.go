package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCgroupCPULimit(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  int
		ok    bool
	}{
		{"v2 quota", map[string]string{
			"proc/self/cgroup":                    "0::/kubepods/pod1\n",
			"sys/fs/cgroup/kubepods/pod1/cpu.max": "150000 100000\n",
		}, 2, true},
		{"v2 unlimited", map[string]string{
			"proc/self/cgroup":      "0::/\n",
			"sys/fs/cgroup/cpu.max": "max 100000\n",
		}, 0, false},
		{"v1 docker, separate cpu mount", map[string]string{
			"proc/self/cgroup":                    "9:name=systemd:/docker/abc\n4:cpu:/docker/abc\n3:cpuacct:/docker/abc\n",
			"sys/fs/cgroup/cpu/cpu.cfs_quota_us":  "100000\n",
			"sys/fs/cgroup/cpu/cpu.cfs_period_us": "100000\n",
		}, 1, true},
		{"v1 combined cpu,cpuacct on host path", map[string]string{
			"proc/self/cgroup": "4:cpu,cpuacct:/ci/job\n",
			"sys/fs/cgroup/cpu,cpuacct/ci/job/cpu.cfs_quota_us":  "250000\n",
			"sys/fs/cgroup/cpu,cpuacct/ci/job/cpu.cfs_period_us": "100000\n",
		}, 3, true},
		{"v1 unlimited", map[string]string{
			"proc/self/cgroup":                    "4:cpu:/\n",
			"sys/fs/cgroup/cpu/cpu.cfs_quota_us":  "-1\n",
			"sys/fs/cgroup/cpu/cpu.cfs_period_us": "100000\n",
		}, 0, false},
		{"fractional quota rounds up to one", map[string]string{
			"proc/self/cgroup":      "0::/\n",
			"sys/fs/cgroup/cpu.max": "20000 100000\n",
		}, 1, true},
		{"no cgroup info", map[string]string{}, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, c.files)
			got, ok := cgroupCPULimit(root)
			if got != c.want || ok != c.ok {
				t.Fatalf("cgroupCPULimit = %d, %v; want %d, %v", got, ok, c.want, c.ok)
			}
		})
	}
}
