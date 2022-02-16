//https://stackoverflow.com/a/11357813/1024794
package main

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/ypapax/logrus_conf"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

func slackInline(stack []byte) string {
	return strings.Join(strings.Split(string(stack), "\n"), "=====")
}

func main() {
	if err := logrus_conf.PrepareFromEnv("cpu_usage"); err != nil {
		logrus.Errorf("%+v", err)
		return
	}
	go CpuUsageInit()
	for {
		SleepByCpuUsage(time.Second, 10*time.Second, 10)
		SleepByCpuUsage(10 * time.Millisecond, 10*time.Second, 80)
		SleepByCpuUsage(time.Second, 10*time.Second, 80)
	}
}

var (
	latestCpuUsagePercent    float64
	latestCpuUsagePercentMtx sync.RWMutex
)

func SleepByCpuUsage(minSl, maxSl time.Duration, minPercentWhenReact float64) {
	s := SleepValueByCpuUsagePercent(minSl, maxSl, minPercentWhenReact)
	logrus.Infof("sleeping for %+v, stack: %+v", s, slackInline(debug.Stack()))
	time.Sleep(s)
}

func SleepValueByCpuUsagePercent(minSl, maxSl time.Duration, minPercentWhenReact float64) (sleepByCpuUsagePerc time.Duration) {
	currentCpuUsage := func() float64 {
		latestCpuUsagePercentMtx.RLock()
		defer latestCpuUsagePercentMtx.RUnlock()
		return latestCpuUsagePercent
	}()
	defer func() {
		logrus.Tracef("sleepByCpuUsagePerc: %+v for currentCpuUsage: %+v ", sleepByCpuUsagePerc, currentCpuUsage)
	}()
	if currentCpuUsage < minPercentWhenReact {
		return minSl
	}
	const (
		minPercent = 0
		maxPercent = 100
	)
	sleepByCpuUsagePerc = time.Duration(currentCpuUsage) * (maxSl - minSl) / time.Duration(maxPercent-minPercent)
	return sleepByCpuUsagePerc
}


func CpuUsageInit() {
	//logrus.Infof("starting")
	sl := time.Second
	for {
		func() {
			defer func() {
				//logrus.Infof("sleeping for %+v", sl)
				time.Sleep(sl)
			}()
			//logrus.Infof("before getting cpu usage")
			perc, err := cpuUsage()
			if err != nil {
				logrus.Errorf("couldn't get cpu usage %+v", errors.WithStack(err))
				return
			}
			func() {
				latestCpuUsagePercentMtx.Lock()
				defer latestCpuUsagePercentMtx.Unlock()
				latestCpuUsagePercent = perc
			}()
			logrus.Tracef("latestCpuUsagePercent is changed to %+v", latestCpuUsagePercent)
		}()
	}
}

func cpuUsage() (percent float64, finalErr error) {
	logrus.Tracef("starting")
	defer func() {
		if r := recover(); r != nil {
			finalErr = errors.Errorf("panic is catched: %+v", r)
		}
	}()
	//t1 := time.Now()
	//log.SetFlags(log.Llongfile | log.LstdFlags)
	cmd := exec.Command("ps", "aux")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	processes := make([]*process, 0)
	for {
		line, errR := out.ReadString('\n')
		if errR != nil {
			break
		}
		tokens := strings.Split(line, " ")
		ft := make([]string, 0)
		for _, t := range tokens {
			if t != "" && t != "\t" {
				ft = append(ft, t)
			}
		}
		//log.Println(len(ft), ft)
		pid, errR := strconv.Atoi(ft[1])
		if errR != nil {
			continue
		}
		cpu, errR := strconv.ParseFloat(ft[2], 64)
		if errR != nil {
			return 0, errors.WithStack(err)
		}
		processes = append(processes, &process{pid, cpu})
	}
	var cpuSum float64
	for _, p := range processes {
		//log.Println("Process ", p.pid, " takes ", p.cpu, " % of the CPU")
		cpuSum += p.cpu
	}
	//log.Printf("sum: %+v, time spent: %+v \n", cpuSum, time.Since(t1))
	//log.Println("The number of CPU Cores:", runtime.NumCPU())
	usage := cpuSum / float64(runtime.NumCPU())
	//log.Println("cpu usage: ", usage)
	return usage, nil
}

type process struct {
	pid int
	cpu float64
}
