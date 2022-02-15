//https://stackoverflow.com/a/11357813/1024794
package main

import (
	"bytes"
	"github.com/pkg/errors"
	"log"
	"os/exec"
	runtime "runtime"
	"strconv"
	"strings"
	"time"
)

type Process struct {
	pid int
	cpu float64
}

func main() {
	percent, err := cpuUsage()
	if err != nil {
		log.Printf("error: %+v\n", err)
		return
	}
	log.Println("cpu usage: ", percent)
}

func cpuUsage() (float64, error) {
	t1 := time.Now()
	log.SetFlags(log.Llongfile | log.LstdFlags)
	cmd := exec.Command("ps", "aux")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	processes := make([]*Process, 0)
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
		log.Println(len(ft), ft)
		pid, errR := strconv.Atoi(ft[1])
		if errR != nil {
			continue
		}
		cpu, errR := strconv.ParseFloat(ft[2], 64)
		if errR != nil {
			return 0, errors.WithStack(err)
		}
		processes = append(processes, &Process{pid, cpu})
	}
	var cpuSum float64
	for _, p := range processes {
		//log.Println("Process ", p.pid, " takes ", p.cpu, " % of the CPU")
		cpuSum += p.cpu
	}
	log.Printf("sum: %+v, time spent: %+v \n", cpuSum, time.Since(t1))
	log.Println("The number of CPU Cores:", runtime.NumCPU())
	usage := cpuSum/float64(runtime.NumCPU())
	log.Println("cpu usage: ", usage)
	return usage, nil
}
