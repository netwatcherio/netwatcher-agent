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

func TestMtrTargets(t []*agent_models.MtrTarget, triggered bool) {
	var wg sync.WaitGroup

	out = make(chan agent_models.MtrTarget, len(t))
	defer close(out)
	for i := range t {
		wg.Add(1)
		go func(tt string) {
			defer wg.Done()
			err := CheckMTR(tn1, 15)
			if err != nil {
				log.Error(err)
			}
		}(t[i])
	}

	wg.Wait()

	return
}

/*func CheckMTR(t *agent_models.MtrTarget, duration int) error {
	var cmd *exec.Cmd
	var regx string
	switch OsDetect {
	case "windows":
		log.Println("Windows")
		break
	case "darwin":
		log.Println("OSX")
		args := []string{"-c", "traceroute " + t.Address}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		// (\d+)\s+(((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}|())\s*([(]((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}[)]|())\s+(((\d+.\d+)\s+([a-z]+))|([*]))\s+(((\d+.\d+)\s+([a-z]+))|([*]))\s+(((\d+.\d+)\s+([a-z]+))|([*]))

		regx = "(\\d+)\\s+" +
			"(((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4}|())\\s*" +
			"([(]((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4}[)]|())\\s+" +
			"(((\\d+.\\d+)\\s+([a-z]+))|([*]))\\s+" +
			"(((\\d+.\\d+)\\s+" +
			"([a-z]+))|([*]))\\s+" +
			"(((\\d+.\\d+)\\s+" +
			"([a-z]+))|([*]))"
		break
	case "linux":
		log.Println("Linux")
		break
	default:
		log.Fatalf("Unknown OS")
	}

	cmd.Wait()

	out, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", out)
	if err != nil {
		log.Error(err)
		return err
	}

	regex := *regexp.MustCompile(regx) // worst
	if err != nil {
		return err
	}

	ethrLines := strings.Split(string(out), "\n")

	t.Result.Metrics = make(map[int]agent_models.MtrMetrics)

	for n, ethrLine := range ethrLines {
		if n <= 0 || ethrLine == "" {
			continue
		}

		dataMatch := regex.FindStringSubmatch(ethrLine)

		log.Infof("%s", len(dataMatch))

		avg := "*"
		if dataMatch[12] != "*" {
			avg = dataMatch[12]
		}

		best := "*"
		if dataMatch[21] != "*" {
			avg = dataMatch[20]
		}

		worst := "*"
		if dataMatch[21] != "*" {
			avg = dataMatch[20]
		}

		address := "???"
		if dataMatch[9] != "" {
			address = dataMatch[9]
		}

		fqdn := "???"
		if dataMatch[2] != "" {
			fqdn = dataMatch[2]
		}

		// hop num = result[1]
		t.Result.Metrics[convHandleStrInt(dataMatch[1])] = agent_models.MtrMetrics{
			Address:  address,
			FQDN:     fqdn,
			Sent:     0,
			Received: 0,
			Last:     "-",
			Avg:      avg,
			Best:     best,
			Worst:    worst,
		}
	}

	j, err := json.Marshal(t.Result)
	if err != nil {
		return err
	}
	log.Warnf("%s", j)

	return nil
}*/

// CheckMTR change to client controller check
func CheckMTR(host string, duration int, triggered bool, out chan agent_models.MtrTarget) error {
	startTime := time.Now()

	var cmd *exec.Cmd
	switch OsDetect {
	case "windows":
		log.Println("Windows")
		break
	case "darwin":
		log.Println("OSX")
		args := []string{"-c", "./lib/ethr_osx -no -w 1 -x " + host + " -p icmp -t mtr -d " +
			strconv.FormatInt(int64(duration), 10) + "s -4"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":
		log.Println("Linux")
		break
	default:
		log.Fatalf("Unknown OS")
	}

	output, err := cmd.CombinedOutput()
	// fmt.Printf("%s\n", output)
	if err != nil {
		log.Error(err)
		return err
	}

	ethrOutput := strings.Split(string(output), "- - - - - - - - - - - - - - - - - "+
		"- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")

	regex := *regexp.MustCompile("(\\d+).{4}((((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4})|([?]{3}))\\s+((\\d+)|([-]))\\s+((\\d+)|([-]))\\s+(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))\\s+(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))\\s+(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))\\s+(((([0-9]*\\.[0-9]+)|([0-9]+\\.))[a-z]+)|([-]))") // worst
	if err != nil {
		return err
	}

	newestSample := strings.Split(ethrOutput[len(ethrOutput)-1], "Ethr done")
	ethrLines := strings.Split(newestSample[0], "\n")

	t := agent_models.MtrTarget{}
	t.Result.StartTimestamp = startTime

	t.Result.Metrics = make(map[int]agent_models.MtrMetrics)

	for n, ethrLine := range ethrLines {
		if n == 1 || n <= 0 || ethrLine == "" {
			continue
		}

		dataMatch := regex.FindStringSubmatch(ethrLine)

		log.Infof("%s", dataMatch)

		// hop num = result[1]
		t.Result.Metrics[convHandleStrInt(dataMatch[1])] = agent_models.MtrMetrics{
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

	out <- t

	return nil
}

func mtrNumDashCheck(str string) int {
	if str == "-" {
		return 0
	}
	return convHandleStrInt(str)
}
