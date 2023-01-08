package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"log"
	"os/exec"
	"runtime"
	"time"
)

type MtrResult struct {
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	Triggered      bool      `bson:"triggered"json:"triggered"`
	Report         struct {
		Mtr struct {
			Src        string `json:"src"bson:"src"`
			Dst        string `json:"dst"bson:"dst"`
			Tos        int    `json:"tos"bson:"tos"`
			Tests      int    `json:"tests"bson:"tests"`
			Psize      string `json:"psize"bson:"psize"`
			Bitpattern string `json:"bitpattern"bson:"bitpattern"`
		} `json:"mtr"bson:"mtr"`
		Hubs []struct {
			Count int     `json:"count"bson:"count"`
			Host  string  `json:"host"bson:"host"`
			ASN   string  `json:"ASN"bson:"ASN"`
			Loss  float64 `json:"Loss%"bson:"Loss%"`
			Drop  int     `json:"Drop"bson:"Drop"`
			Rcv   int     `json:"Rcv"bson:"Rcv"`
			Snt   int     `json:"Snt"bson:"Snt"`
			Best  float64 `json:"Best"bson:"Best"`
			Avg   float64 `json:"Avg"bson:"Avg"`
			Wrst  float64 `json:"Wrst"bson:"Wrst"`
			StDev float64 `json:"StDev"bson:"StDev"`
			Gmean float64 `json:"Gmean"bson:"Gmean"`
			Jttr  float64 `json:"Jttr"bson:"Jttr"`
			Javg  float64 `json:"Javg"bson:"Javg"`
			Jmax  float64 `json:"Jmax"bson:"Jmax"`
			Jint  float64 `json:"Jint"bson:"Jint"`
		} `json:"hubs"bson:"hubs"`
	} `json:"report"bson:"report"`
}

// CheckMtr run the check for mtr, take input from checkdata for the test, and update the mtrresult object
func CheckMtr(cd *api.AgentCheck, triggered bool) (MtrResult, error) {
	osDetect := runtime.GOOS
	var mtrResult MtrResult
	mtrResult.StartTimestamp = time.Now()

	var cmd *exec.Cmd
	switch osDetect {
	case "windows":
		// mtr needs to be installed manually currently
		args := []string{"/C", "./lib/mtr_windows_x86 " + cd.Target + " -z --show-ips -o LDRSBAWVGJMXI --json"}
		cmd = exec.CommandContext(context.TODO(), "cmd", args...)
		break
	case "darwin":
		// mtr needs to be installed manually currently
		args := []string{"-c", "./lib/mtr_darwin " + cd.Target + " -z --show-ips -o LDRSBAWVGJMXI --json"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":
		// mtr needs to be installed manually currently
		args := []string{"-c", "mtr " + cd.Target + " -z --show-ips -o LDRSBAWVGJMXI --json"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	default:
		log.Fatalf("Unknown OS")
		panic("TODO")
	}

	output, err := cmd.Output()
	fmt.Printf("%s\n", output)
	if err != nil {
		return mtrResult, err
	}

	err = json.Unmarshal(output, &mtrResult)
	if err != nil {
		return mtrResult, err
	}
	/*r.StopTimestamp = time.Now()*/
	mtrResult.StopTimestamp = time.Now()
	mtrResult.Triggered = triggered

	return mtrResult, nil
}

func mtrNumDashCheck(str string) int {
	if str == "-" {
		return 0
	}
	return ConvHandleStrInt(str)
}
