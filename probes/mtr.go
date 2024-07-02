package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

type MtrResult struct {
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	Report         struct {
		Info struct {
			Target struct {
				IP       string `json:"ip"`
				Hostname string `json:"hostname"`
			} `json:"target"`
		} `json:"info"`
		Hops []struct {
			TTL   int `json:"ttl"`
			Hosts []struct {
				IP       string `json:"ip"`
				Hostname string `json:"hostname"`
			} `json:"hosts"`
			Extensions []string `json:"extensions"`
			LossPct    string   `json:"loss_pct"`
			Sent       int      `json:"sent"`
			Last       string   `json:"last"`
			Recv       int      `json:"recv"`
			Avg        string   `json:"avg"`
			Best       string   `json:"best"`
			Worst      string   `json:"worst"`
			StdDev     string   `json:"stddev"`
		} `json:"hops"`
	} `json:"report"bson:"report"`
}

/*type MtrResult struct {
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	Triggered      bool      `bson:"triggered"json:"triggered"`
	Report         struct {
		Mtr struct {
			Src        string      `json:"src"bson:"src"`
			Dst        string      `json:"dst"bson:"dst"`
			Tos        interface{} `json:"tos"bson:"tos"`
			Tests      interface{} `json:"tests"bson:"tests"`
			Psize      string      `json:"psize"bson:"psize"`
			Bitpattern string      `json:"bitpattern"bson:"bitpattern"`
		} `json:"mtr"bson:"mtr"`
		Hubs []struct {
			Count interface{} `json:"count"bson:"count"`
			Host  string      `json:"host"bson:"host"`
			ASN   string      `json:"ASN"bson:"ASN"`
			Loss  float64     `json:"Loss%"bson:"Loss%"`
			Drop  int         `json:"Drop"bson:"Drop"`
			Rcv   int         `json:"Rcv"bson:"Rcv"`
			Snt   int         `json:"Snt"bson:"Snt"`
			Best  float64     `json:"Best"bson:"Best"`
			Avg   float64     `json:"Avg"bson:"Avg"`
			Wrst  float64     `json:"Wrst"bson:"Wrst"`
			StDev float64     `json:"StDev"bson:"StDev"`
			Gmean float64     `json:"Gmean"bson:"Gmean"`
			Jttr  float64     `json:"Jttr"bson:"Jttr"`
			Javg  float64     `json:"Javg"bson:"Javg"`
			Jmax  float64     `json:"Jmax"bson:"Jmax"`
			Jint  float64     `json:"Jint"bson:"Jint"`
		} `json:"hubs"bson:"hubs"`
	} `json:"report"bson:"report"`
}*/

// Mtr run the check for mtr, take input from checkdata for the test, and update the mtrresult object
func Mtr(cd *Probe, triggered bool) (MtrResult, error) {
	osDetect := runtime.GOOS
	var mtrResult MtrResult
	mtrResult.StartTimestamp = time.Now()

	//todo convert this to use trippycli??

	/*var cmdStr string*/
	if runtime.GOARCH == "amd64" {
		// Load your x86-specific external library here
	} else if runtime.GOARCH == "arm64" {
		// Load your ARM-specific external library here
	} else {
		fmt.Println("Unsupported architecture for MTR test")
	}

	/*cmdStr += " " + cd.Config.Target[0].Target + " -z --show-ips -o LDRSBAWVGJMXI --json"*/

	ctx := context.Background()

	triggeredCount := 5

	if triggered {
		triggeredCount = 15
	}
	var cmd *exec.Cmd
	switch osDetect {
	case "windows":
		args := []string{
			"--icmp",
			"--mode", "json",
			"--multipath-strategy", "classic",
			"--dns-resolve-method", "cloudflare",
			"--report-cycles", strconv.Itoa(triggeredCount),
			"--dns-lookup-as-info", cd.Config.Target[0].Target,
		}

		// Construct the command with context and arguments
		cmd = exec.CommandContext(ctx, "./lib/trip_windows-x86_64.exe", args...)
		break
	case "darwin":
		// mtr needs to be installed manually currently
		/*args := []string{"-c", "./lib/mtr_darwin " + cd.Config.Target[0].Target + " -z --show-ips -o LDRSBAWVGJMXI --json"}
		cmd = exec.CommandContext(ctx, "/bin/bash", args...)*/
		args := []string{"-c", "./lib/trip_darwin --icmp --mode json --multipath-strategy paris --dns-resolve-method cloudflare --report-cycles " + strconv.Itoa(triggeredCount) + " --dns-lookup-as-info " + cd.Config.Target[0].Target}
		cmd = exec.CommandContext(ctx, "/bin/bash", args...)
		break
	case "linux":
		// mtr needs to be installed manually currently
		/*args := []string{"-c", "mtr " + cd.Config.Target[0].Target + " -z --show-ips -o LDRSBAWVGJMXI --json"}
		cmd = exec.CommandContext(ctx, "/bin/bash", args...)*/
		args := []string{"-c", "./lib/trip_linux-x86_64 --icmp --mode json --multipath-strategy paris --dns-resolve-method cloudflare --report-cycles " + strconv.Itoa(triggeredCount) + " --dns-lookup-as-info " + cd.Config.Target[0].Target}
		cmd = exec.CommandContext(ctx, "/bin/bash", args...)
		break
	default:
		log.Fatalf("Unknown OS")
	}

	var cancel context.CancelFunc
	// timeout after 60 seconds
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()

	output, err := cmd.CombinedOutput()
	// fmt.Printf("%s\n", output)
	if err != nil {
		return mtrResult, err
	}

	err = json.Unmarshal(output, &mtrResult.Report)
	if err != nil {
		return mtrResult, err
	}

	/*r.StopTimestamp = time.Now()*/
	mtrResult.StopTimestamp = time.Now()

	return mtrResult, nil
}

func mtrNumDashCheck(str string) int {
	if str == "-" {
		return 0
	}
	return ConvHandleStrInt(str)
}
