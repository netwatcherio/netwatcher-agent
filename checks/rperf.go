package checks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type RPerfResults struct {
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	Config         struct {
		Additional struct {
			IpVersion   int  `json:"ip_version"`
			OmitSeconds int  `json:"omit_seconds"`
			Reverse     bool `json:"reverse"`
		} `json:"additional"`
		Common struct {
			Family  string `json:"family"`
			Length  int    `json:"length"`
			Streams int    `json:"streams"`
		} `json:"common"`
		Download struct {
		} `json:"download"`
		Upload struct {
			Bandwidth    int     `json:"bandwidth"`
			Duration     float64 `json:"duration"`
			SendInterval float64 `json:"send_interval"`
		} `json:"upload"`
	} `json:"config"`
	Streams []struct {
		Abandoned bool `json:"abandoned"`
		Failed    bool `json:"failed"`
		Intervals struct {
			Receive []struct {
				BytesReceived     int     `json:"bytes_received"`
				Duration          float64 `json:"duration"`
				JitterSeconds     float64 `json:"jitter_seconds"`
				PacketsDuplicated int     `json:"packets_duplicated"`
				PacketsLost       int     `json:"packets_lost"`
				PacketsOutOfOrder int     `json:"packets_out_of_order"`
				PacketsReceived   int     `json:"packets_received"`
				Timestamp         float64 `json:"timestamp"`
				UnbrokenSequence  int     `json:"unbroken_sequence"`
			} `json:"receive"`
			Send []struct {
				BytesSent    int     `json:"bytes_sent"`
				Duration     float64 `json:"duration"`
				PacketsSent  int     `json:"packets_sent"`
				SendsBlocked int     `json:"sends_blocked"`
				Timestamp    float64 `json:"timestamp"`
			} `json:"send"`
			Summary struct {
				BytesReceived            int     `json:"bytes_received"`
				BytesSent                int     `json:"bytes_sent"`
				DurationReceive          float64 `json:"duration_receive"`
				DurationSend             float64 `json:"duration_send"`
				FramedPacketSize         int     `json:"framed_packet_size"`
				JitterAverage            float64 `json:"jitter_average"`
				JitterPacketsConsecutive int     `json:"jitter_packets_consecutive"`
				PacketsDuplicated        int     `json:"packets_duplicated"`
				PacketsLost              int     `json:"packets_lost"`
				PacketsOutOfOrder        int     `json:"packets_out_of_order"`
				PacketsReceived          int     `json:"packets_received"`
				PacketsSent              int     `json:"packets_sent"`
			} `json:"summary"`
		} `json:"intervals"`
	} `json:"streams"`
	Success bool `json:"success"`
	Summary struct {
		BytesReceived            int     `json:"bytes_received"`
		BytesSent                int     `json:"bytes_sent"`
		DurationReceive          float64 `json:"duration_receive"`
		DurationSend             float64 `json:"duration_send"`
		FramedPacketSize         int     `json:"framed_packet_size"`
		JitterAverage            float64 `json:"jitter_average"`
		JitterPacketsConsecutive int     `json:"jitter_packets_consecutive"`
		PacketsDuplicated        int     `json:"packets_duplicated"`
		PacketsLost              int     `json:"packets_lost"`
		PacketsOutOfOrder        int     `json:"packets_out_of_order"`
		PacketsReceived          int     `json:"packets_received"`
		PacketsSent              int     `json:"packets_sent"`
	} `json:"summary"`
}

//./rperf -c 0.0.0.0 -p 5199 -b 8K -t 10 --udp -f json

func (r *RPerfResults) Check(cd *CheckData) error {
	osDetect := runtime.GOOS
	r.StartTimestamp = time.Now()

	var cmd *exec.Cmd
	switch osDetect {
	case "windows":
		break
	case "darwin":
		targetHost := strings.Split(cd.Target, ":")
		args := []string{"-c", "./lib/rperf_darwin -c " + targetHost[0] + " -p " + targetHost[1] + " -b 8K -t " + strconv.Itoa(cd.Duration) + " --udp -f json"}
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
		break
	case "linux":

		break
	default:
		log.Fatalf("Unknown OS")
		panic("TODO")
	}

	out, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", out)
	if err != nil {
		return err
	}

	lineN := -1

	beforeJson := ""

	lines := strings.Split(string(out), "\n")
	for n, l := range lines {
		if l != "{" {
			beforeJson += l + "\n"
		} else {
			lineN = n
			break
		}
	}

	if lineN == -1 {
		return errors.New("something went wrong")
	}

	justJson := strings.ReplaceAll(string(out), beforeJson, "")

	r.StopTimestamp = time.Now()
	err = json.Unmarshal([]byte(justJson), &r)
	if err != nil {
		return err
	}

	cd.Result = r

	return nil
}
