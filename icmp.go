package main

import (
	"context"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO DSCP Tags? https://github.com/rclone/rclone/issues/755

func CheckICMP(t string, duration int, out chan agent_models.IcmpTarget) error {
	var icmpTarget = agent_models.IcmpTarget{
		Address: t,
	}
	icmpTarget.Result.StartTimestamp = time.Now()

	var cmd *exec.Cmd
	switch OsDetect {
	case "windows":
		break
	case "darwin":
		args := []string{"-c", "./lib/ethr_osx -no -w 1 -x " + t + " -p icmp -t pi -d " +
			strconv.FormatInt(int64(duration), 10) + "s -4"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":
		break
	default:
		log.Fatalf("Unknown OS")
	}

	cmdOut, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", cmdOut)
	if err != nil {
		log.Error(err)
		return err
	}

	compile1, err := regexp.Compile(".=.(.)")
	if err != nil {
		return err
	}
	ethrOutput := strings.Split(string(cmdOut), "-----------------------------------------------------------------------------------------")
	metrics1 := compile1.FindAllString(ethrOutput[1], -1)
	if err != nil {
		return err
	}
	compile2, err := regexp.Compile("(([0-9]*\\.[0-9]+)|([0-9]+\\.))(?:ms)")
	if err != nil {
		return err
	}
	metrics2 := compile2.FindAllString(ethrOutput[2], -1)

	icmpTarget.Result.Metrics = agent_models.IcmpMetrics{
		Avg:         metrics2[1],
		Min:         metrics2[2],
		Max:         metrics2[9],
		Sent:        ConvHandleStrInt(metrics1[0]),
		Received:    ConvHandleStrInt(metrics1[1]),
		Loss:        ConvHandleStrInt(metrics1[2]),
		Percent50:   metrics2[3],
		Percent90:   metrics2[4],
		Percent95:   metrics2[5],
		Percent99:   metrics2[6],
		Percent999:  metrics2[7],
		Percent9999: metrics2[8],
	}
	icmpTarget.Result.StopTimestamp = time.Now()

	out <- icmpTarget

	// todo regex 🤪
	return nil
}

func TestIcmpTargets(t []string, interval int) (out chan agent_models.IcmpTarget) {
	var wg sync.WaitGroup

	out = make(chan agent_models.IcmpTarget, len(t))
	defer close(out)
	for i := range t {
		wg.Add(1)
		go func(tn1 string) {
			defer wg.Done()
			err := CheckICMP(tn1, interval, out)
			if err != nil {
				log.Errorf("%s", err)
			}
		}(t[i])
	}
	wg.Wait()
	return
}
