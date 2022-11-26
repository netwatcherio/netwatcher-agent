package main

import (
	"context"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func TestMtrTargets(t []string, triggered bool) ([]*agent_models.MtrTarget, error) {
	var targets []*agent_models.MtrTarget

	ch := make(chan *agent_models.MtrTarget)

	var wg sync.WaitGroup
	for i := range t {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target, err := CheckMTR(t[i], 15, triggered)
			if err != nil {
				log.Error(err)
			}

			ch <- target
		}()
	}
	targets = append(targets, <-ch)
	wg.Wait()
	return targets, nil
}

// CheckMTR change to client controller check
func CheckMTR(host string, duration int, triggered bool) (*agent_models.MtrTarget, error) {
	startTime := time.Now()

	var cmd *exec.Cmd
	switch OsDetect {
	case "windows":
		break
	case "darwin":
		args := []string{"-c", "./lib/ethr_osx -no -w 1 -x " + host + " -p icmp -t mtr -d " +
			strconv.FormatInt(int64(duration), 10) + "s -4"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":
		break
	default:
		log.Fatalf("Unknown OS")
	}

	output, err := cmd.CombinedOutput()
	// fmt.Printf("%s\n", output)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	ethrOutput := strings.Split(string(output), "- - - - - - - - - - - - - - - - - "+
		"- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")

	regex := *regexp.MustCompile("(\\d+).{4}" +
		"((((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4})|([?]{3}))\\s+" +
		"((\\d+)|([-]))\\s+((\\d+)|([-]))\\s+(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))\\s+" +
		"(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))\\s+" +
		"(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))\\s+" +
		"(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))") // worst
	if err != nil {
		return nil, err
	}

	newestSample := strings.Split(ethrOutput[len(ethrOutput)-1], "Ethr done")
	ethrLines := strings.Split(newestSample[0], "\n")

	t := &agent_models.MtrTarget{}
	t.Result.StartTimestamp = startTime

	t.Result.Metrics = make(map[int]agent_models.MtrMetrics)

	for n, ethrLine := range ethrLines {
		if n == 1 || n <= 0 || ethrLine == "" {
			continue
		}

		dataMatch := regex.FindStringSubmatch(ethrLine)

		log.Infof("%s", dataMatch)

		// hop num = result[1]
		t.Result.Metrics[ConvHandleStrInt(dataMatch[1])] = agent_models.MtrMetrics{
			Address:  dataMatch[2],
			Sent:     mtrNumDashCheck(dataMatch[8]),
			Received: mtrNumDashCheck(dataMatch[11]),
			Last:     dataMatch[14],
			Avg:      dataMatch[20],
			Best:     dataMatch[26],
			Worst:    dataMatch[32],
		}
	}
	t.Address = host
	t.Result.StopTimestamp = time.Now()
	t.Result.Triggered = triggered
	// fmt.Printf("%s", j)
	// j, err := json.Marshal(t.Result)
	// if err != nil {
	// 	return err
	// }

	return t, nil
}

func mtrNumDashCheck(str string) int {
	if str == "-" {
		return 0
	}
	return ConvHandleStrInt(str)
}
