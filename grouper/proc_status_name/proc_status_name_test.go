package proc_status_name

import (
	"os"
	"testing"

	"github.com/k1LoW/grouped_process_exporter/grouped_proc"
	"github.com/k1LoW/grouped_process_exporter/metric"
)

const (
	cgroupTestProcPath = "../../testdata/proc"
)

func TestCollect(t *testing.T) {
	os.Setenv("GROUPED_PROCESS_PROC_MOUNT_POINT", cgroupTestProcPath)
	procStatusName := NewProcStatusName()
	gpMap := make(map[string]*grouped_proc.GroupedProc)
	enabled := make(map[metric.MetricKey]bool)

	enabled[metric.ProcIO] = true
	enabled[metric.ProcStat] = true
	err := procStatusName.Collect(gpMap, enabled)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(gpMap) != 2 {
		t.Errorf("want %d, got %d", 2, len(gpMap))
	}
	if _, ok := gpMap["nginx"]; !ok {
		t.Errorf("want %s, got none", "nginx")
	}
	if _, ok := gpMap["mysqld"]; !ok {
		t.Errorf("want %s, got none", "mysqld")
	}
}