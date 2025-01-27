package cgroup

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/k1LoW/grouped_process_exporter/grouped_proc"
	"github.com/k1LoW/grouped_process_exporter/metric"
	"golang.org/x/sync/semaphore"
)

// Subsystems cgroups subsystems list
var Subsystems = []string{
	"cpuset",
	"cpu",
	"cpuacct",
	"blkio",
	"memory",
	"devices",
	"freezer",
	"net_cls",
	"net_prio",
	"perf_event",
	"hugetlb",
	"pids",
	"rdma",
}

type Cgroup struct {
	fsPath string
	nRe    *regexp.Regexp
	eRe    *regexp.Regexp
}

func (c *Cgroup) Name() string {
	return "cgroup"
}

func (c *Cgroup) Collect(gprocs *grouped_proc.GroupedProcs, enabled map[metric.MetricKey]bool, sem *semaphore.Weighted) error {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, s := range Subsystems {
		searchDir := filepath.Clean(filepath.Join(c.fsPath, s))

		err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if f == nil {
				return nil
			}
			err = sem.Acquire(ctx, 2)
			if err != nil {
				return err
			}
			defer sem.Release(2)
			if f.IsDir() {
				cPath := strings.Replace(path, searchDir, "", 1)
				if c.eRe != nil {
					if c.eRe.MatchString(cPath) {
						return nil
					}
				}
				if c.nRe != nil {
					matches := c.nRe.FindStringSubmatch(cPath)
					if len(matches) > 1 {
						cPath = matches[1]
					}
				}
				if cPath != "" {
					f, err := os.Open(filepath.Clean(filepath.Join(path, "cgroup.procs")))
					if err != nil {
						_ = f.Close()
						return nil
					}
					var (
						gproc *grouped_proc.GroupedProc
						ok    bool
					)
					gproc, ok = gprocs.Load(cPath)
					if !ok {
						gproc = grouped_proc.NewGroupedProc(enabled)
						gprocs.Store(cPath, gproc)
					}
					gproc.Exists = true
					reader := bufio.NewReaderSize(f, 1028)
					for {
						line, _, err := reader.ReadLine()
						if err == io.EOF {
							break
						} else if err != nil {
							_ = f.Close()
							return err
						}
						pid, err := strconv.Atoi(string(line))
						if err != nil {
							_ = f.Close()
							return err
						}
						err = sem.Acquire(ctx, gproc.RequiredWeight)
						if err != nil {
							_ = f.Close()
							return err
						}
						wg.Add(1)
						go func(wg *sync.WaitGroup, pid int, gproc *grouped_proc.GroupedProc) {
							_ = gproc.AppendProcAndCollect(pid)
							sem.Release(gproc.RequiredWeight)
							wg.Done()
						}(wg, pid, gproc)
					}
				}
				return nil
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	wg.Wait()
	return nil
}

func (c *Cgroup) SetNormalizeRegexp(nReStr string) error {
	if nReStr == "" {
		return nil
	}
	nRe, err := regexp.Compile(nReStr)
	if err != nil {
		return err
	}
	if nRe.NumSubexp() != 1 {
		return errors.New("number of parenthesized subexpressions in this regexp should be 1")
	}
	c.nRe = nRe
	return nil
}

func (c *Cgroup) SetExcludeRegexp(eReStr string) error {
	if eReStr == "" {
		return nil
	}
	eRe, err := regexp.Compile(eReStr)
	if err != nil {
		return err
	}
	c.eRe = eRe
	return nil
}

// NewCgroup
func NewCgroup(fsPath string) *Cgroup {
	return &Cgroup{
		fsPath: fsPath,
	}
}
