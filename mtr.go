package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

func TestMtrTargets(t []*agent_models.MtrTarget, triggered bool) {
	var wg sync.WaitGroup

	// ([0-9]*).{4}(((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4})\s*([0-9]*)\s*([0-9]*)\s*(([0-9]*\.[0-9]+)|([0-9]+\.))(ms)\s*(([0-9]*\.[0-9]+)|([0-9]+\.))(ms)\s*(([0-9]*\.[0-9]+)|([0-9]+\.))(ms)\s*(([0-9]*\.[0-9]+)|([0-9]+\.))(ms)

	for _, tn := range t {
		wg.Add(1)
		go func(tn1 *agent_models.MtrTarget) {
			defer wg.Done()
			err := CheckMTR(tn1, 5)
			if err != nil {
				log.Error(err)
			}

			tn1.Result.StopTimestamp = time.Now()
			tn1.Result.Triggered = triggered
		}(tn)
	}

	wg.Wait()
}

func CheckMTR(t *agent_models.MtrTarget, duration int) error {
	var cmd *exec.Cmd
	switch OsDetect {
	case "windows":
		log.Println("Windows")
		break
	case "darwin":
		log.Println("OSX")
		args := []string{"-c", "./lib/ethr_osx -x " + t.Address + " -p tcp -t mtr -d ", string(duration), "s -4"}
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

	ethrOutput := strings.Split(string(out), "- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")

	regex := regexp.MustCompile("([0-9]*).{4}(((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4})\\s*([0-9]*)\\s*([0-9]*)\\s*(([0-9]*\\.[0-9]+)|([0-9]+\\.))(ms)\\s*(([0-9]*\\.[0-9]+)|([0-9]+\\.))(ms)\\s*(([0-9]*\\.[0-9]+)|([0-9]+\\.))(ms)\\s*(([0-9]*\\.[0-9]+)|([0-9]+\\.))(ms)\n")
	if err != nil {
		return err
	}
	hops := regex.Split(ethrOutput[1], -1)

	for _, hop := range hops {
		result := regex.FindAll([]byte(hop), -1)
		if err != nil {
			return err
		}

		// hop num = result[1]
		t.Result.Mtr[convHandleStrInt(string(result[0][1]))] = agent_models.MtrHop{
			Address:  string(result[1]),
			Sent:     convHandleStrInt(string(result[0][5])),
			Received: convHandleStrInt(string(result[0][6])),
			Last:     string(result[0][8]),
			Avg:      string(result[0][9]),
			Best:     string(result[0][10]),
			Worst:    string(result[0][11]),
		}
	}

	j, err := json.Marshal(t.Result)
	if err != nil {
		return err
	}
	log.Warnf("%s", j)

	return nil
}
