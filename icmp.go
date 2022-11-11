package main

import (
	"context"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TODO DSCP Tags? https://github.com/rclone/rclone/issues/755

func CheckICMP(t *agent_models.IcmpTarget, duration int) error {
	var cmd *exec.Cmd
	switch OsDetect {
	case "windows":
		log.Println("Windows")
		break
	case "darwin":
		log.Println("OSX")
		args := []string{"-c", "./lib/ethr_osx -x " + t.Address + " -p icmp -t pi -d " + string(duration) + "s -4"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":
		log.Println("Linux")
		break
	default:
		log.Fatalf("Unknown OS")
	}

	out, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", out)
	if err != nil {
		log.Error(err)
		return err
	}

	compile1, err := regexp.Compile(".=.(.)")
	if err != nil {
		return err
	}
	ethrOutput := strings.Split(string(out), "-----------------------------------------------------------------------------------------")
	metrics1 := compile1.FindAllString(ethrOutput[1], -1)
	if err != nil {
		return err
	}
	compile2, err := regexp.Compile("(([0-9]*\\.[0-9]+)|([0-9]+\\.))(ms)\n")
	if err != nil {
		return err
	}
	metrics2 := compile2.FindAllString(ethrOutput[2], -1)
	if err != nil {
		return err
	}

	t.Result.Metrics = agent_models.IcmpMetrics{
		Avg:         metrics2[0],
		Min:         metrics2[1],
		Max:         metrics2[8],
		Sent:        convHandleStrInt(metrics1[0]),
		Received:    convHandleStrInt(metrics1[1]),
		Loss:        convHandleStrInt(metrics1[2]),
		Percent50:   metrics2[2],
		Percent90:   metrics2[3],
		Percent95:   metrics2[4],
		Percent99:   metrics2[5],
		Percent999:  metrics2[6],
		Percent9999: metrics2[7],
	}

	// todo regex 🤪

	return nil
}

func TestIcmpTargets(t []*agent_models.IcmpTarget, interval int) {
	var wg sync.WaitGroup

	log.Infof("len %v", len(t))
	for _, tn := range t {
		log.Infof("starting icmp for %s", tn.Address)
		wg.Add(1)
		go func(tn1 *agent_models.IcmpTarget) {
			defer wg.Done()
			err := CheckICMP(tn1, interval)
			if err != nil {
				//  read ip 0.0.0.0: raw-read ip4 0.0.0.0: i/o timeout
				log.Errorf("%s", err)
			}
			tn1.Result.StopTimestamp = time.Now()
			// "sleep" is handled by ethr because it would be running the test
			// otherwise it would throw an error.
			log.Infof("ending icmp for %s", tn1.Address)
		}(tn)
	}
	wg.Wait()
}
