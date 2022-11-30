package main

import (
	"context"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO DSCP Tags? https://github.com/rclone/rclone/issues/755

func CheckICMP(t string, duration int) (*agent_models.IcmpTarget, error) {
	var icmpTarget = &agent_models.IcmpTarget{
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
		return nil, err
	}

	compile1, err := regexp.Compile(".=.(.)")
	if err != nil {
		return nil, err
	}
	ethrOutput := strings.Split(string(cmdOut), "-----------------------------------------------------------------------------------------")
	metrics1 := compile1.FindAllString(ethrOutput[1], -1)
	if err != nil {
		return nil, err
	}
	compile2, err := regexp.Compile("(([0-9]*\\.[0-9]+)|([0-9]+\\.))(?:ms)")
	if err != nil {
		return nil, err
	}
	metrics2 := compile2.FindAllString(ethrOutput[2], -1)

	icmpTarget.Result.Metrics = agent_models.IcmpMetrics{
		Avg:         strings.ReplaceAll(strings.ReplaceAll(metrics2[1], " ", ""), "\n", ""),
		Min:         strings.ReplaceAll(metrics2[2], " ", ""),
		Max:         strings.ReplaceAll(metrics2[9], " ", ""),
		Sent:        ConvHandleStrInt(metrics1[0]),
		Received:    ConvHandleStrInt(metrics1[1]),
		Loss:        ConvHandleStrInt(metrics1[2]),
		Percent50:   strings.ReplaceAll(metrics2[3], " ", ""),
		Percent90:   strings.ReplaceAll(metrics2[4], " ", ""),
		Percent95:   strings.ReplaceAll(metrics2[5], " ", ""),
		Percent99:   strings.ReplaceAll(metrics2[6], " ", ""),
		Percent999:  strings.ReplaceAll(metrics2[7], " ", ""),
		Percent9999: strings.ReplaceAll(metrics2[8], " ", ""),
	}
	icmpTarget.Result.StopTimestamp = time.Now()

	// todo regex 🤪
	return icmpTarget, nil
}

func TestIcmpTargets(t []string, length int) ([]*agent_models.IcmpTarget, error) {
	var targets []*agent_models.IcmpTarget

	ch2 := make(chan *agent_models.IcmpTarget, len(t))

	var wg sync.WaitGroup
	for i := range t {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			target, err := CheckICMP(s, length)
			if err != nil {
				log.Error(err)
			}

			ch2 <- target
		}(t[i])
	}
	wg.Wait()
	close(ch2)

	for i := range ch2 {
		targets = append(targets, i)
		log.Warnf("ICMP: %s", i.Address)
	}

	return targets, nil
}

func CheckICMP(t string, duration int) (*agent_models.IcmpTarget, error) {
	var icmpTarget = &agent_models.IcmpTarget{
		Address: t,
	}
	icmpTarget.Result.StartTimestamp = time.Now()

	var cmd *exec.Cmd
	switch OsDetect {
	case "windows":
		break
	case "darwin":
		args := []string{"-c", "./lib/ethr_osx -no -w 1 -x " + t + " -p icmp -t pi -d " +
			strconv.FormatInt(int64(duration), 10) + "m -4"}
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
		return nil, err
	}

	fmt.Fprintf(os.Stderr, string(cmdOut))

	compile1, err := regexp.Compile(".=.(.)")
	if err != nil {
		return nil, err
	}

	ethrOutput := strings.Split(string(cmdOut), "-----------------------------------------------------------------------------------------")

	metrics1 := compile1.FindAllString(ethrOutput[1], -1)
	if err != nil {
		return nil, err
	}

	compile2, err := regexp.Compile("(\\s+(([0-9]+.[0-9]+..)|(.+)))")
	if err != nil {
		return nil, err
	}
	metrics2 := compile2.FindAllString(ethrOutput[2], -1)

	icmpTarget.Result.Metrics = agent_models.IcmpMetrics{
		Avg:         strings.ReplaceAll(strings.ReplaceAll(metrics2[1], " ", ""), "\n", ""),
		Min:         strings.ReplaceAll(metrics2[2], " ", ""),
		Max:         strings.ReplaceAll(metrics2[9], " ", ""),
		Sent:        ConvHandleStrInt(metrics1[0]),
		Received:    ConvHandleStrInt(metrics1[1]),
		Loss:        ConvHandleStrInt(metrics1[2]),
		Percent50:   strings.ReplaceAll(metrics2[3], " ", ""),
		Percent90:   strings.ReplaceAll(metrics2[4], " ", ""),
		Percent95:   strings.ReplaceAll(metrics2[5], " ", ""),
		Percent99:   strings.ReplaceAll(metrics2[6], " ", ""),
		Percent999:  strings.ReplaceAll(metrics2[7], " ", ""),
		Percent9999: strings.ReplaceAll(metrics2[8], " ", ""),
	}
	icmpTarget.Result.StopTimestamp = time.Now()

	// todo regex 🤪
	return icmpTarget, nil
}
