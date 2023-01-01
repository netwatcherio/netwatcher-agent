package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"
)

type MtrData struct {
	Report struct {
		Mtr struct {
			Src        string `json:"src"`
			Dst        string `json:"dst"`
			Tos        int    `json:"tos"`
			Tests      int    `json:"tests"`
			Psize      string `json:"psize"`
			Bitpattern string `json:"bitpattern"`
		} `json:"mtr"`
		Hubs []struct {
			Count int     `json:"count"`
			Host  string  `json:"host"`
			Loss  float64 `json:"Loss%"`
			Snt   int     `json:"Snt"`
			Last  float64 `json:"Last"`
			Avg   float64 `json:"Avg"`
			Best  float64 `json:"Best"`
			Wrst  float64 `json:"Wrst"`
			StDev float64 `json:"StDev"`
		} `json:"hubs"`
	} `json:"report"`
}

type MtrResult struct {
	Metrics        MtrData   `json:"metrics"bson:"metrics"`
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
}

/*type MtrMetrics struct {
	Address  string `json:"address"bson:"address"`
	FQDN     string `bson:"fqdn"json:"fqdn"`
	Sent     int    `json:"sent"bson:"sent"`
	Received int    `json:"received"bson:"received"`
	Last     string `bson:"last"json:"last"`
	Avg      string `bson:"avg"json:"avg"`
	Best     string `bson:"best"json:"best"`
	Worst    string `bson:"worst"json:"worst"`
}*/

func (r *MtrResult) Check(cd *CheckData) error {
	osDetect := runtime.GOOS
	r.StartTimestamp = time.Now()

	var cmd *exec.Cmd
	switch osDetect {
	case "windows":
		break
	case "darwin":
		// mtr needs to be installed manually currently
		args := []string{"-c", "./lib/mtr_darwin " + cd.Target + " --json"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":

		break
	default:
		log.Fatalf("Unknown OS")
		panic("TODO")
	}

	output, err := cmd.Output()
	fmt.Printf("%s\n", output)
	if err != nil {
		return err
	}

	mtr := MtrData{}

	err = json.Unmarshal(output, &mtr)
	if err != nil {
		return err
	}

	r.Metrics = mtr

	r.StopTimestamp = time.Now()
	cd.Result = r

	return nil
}

func mtrNumDashCheck(str string) int {
	if str == "-" {
		return 0
	}
	return ConvHandleStrInt(str)
}
