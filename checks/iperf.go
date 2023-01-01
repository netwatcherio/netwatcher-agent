package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type IperfResults struct {
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	Start          struct {
		Connected []struct {
			Socket     int    `json:"socket"`
			LocalHost  string `json:"local_host"`
			LocalPort  int    `json:"local_port"`
			RemoteHost string `json:"remote_host"`
			RemotePort int    `json:"remote_port"`
		} `json:"connected"`
		Version    string `json:"version"`
		SystemInfo string `json:"system_info"`
		Timestamp  struct {
			Time     string `json:"time"`
			Timesecs int    `json:"timesecs"`
		} `json:"timestamp"`
		ConnectingTo struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"connecting_to"`
		Cookie       string `json:"cookie"`
		SockBufsize  int    `json:"sock_bufsize"`
		SndbufActual int    `json:"sndbuf_actual"`
		RcvbufActual int    `json:"rcvbuf_actual"`
		TestStart    struct {
			Protocol   string `json:"protocol"`
			NumStreams int    `json:"num_streams"`
			Blksize    int    `json:"blksize"`
			Omit       int    `json:"omit"`
			Duration   int    `json:"duration"`
			Bytes      int    `json:"bytes"`
			Blocks     int    `json:"blocks"`
			Reverse    int    `json:"reverse"`
			Tos        int    `json:"tos"`
		} `json:"test_start"`
	} `json:"start"`
	/*Intervals []struct {
		Streams []struct {
			Socket        int     `json:"socket"`
			Start         float64 `json:"start"`
			End           float64 `json:"end"`
			Seconds       float64 `json:"seconds"`
			Bytes         int     `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			Packets       int     `json:"packets"`
			Omitted       bool    `json:"omitted"`
			Sender        bool    `json:"sender"`
		} `json:"streams"`
		Sum struct {
			Start         float64 `json:"start"`
			End           float64 `json:"end"`
			Seconds       float64 `json:"seconds"`
			Bytes         int     `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			Packets       int     `json:"packets"`
			Omitted       bool    `json:"omitted"`
			Sender        bool    `json:"sender"`
		} `json:"sum"`
	} `json:"intervals"`*/
	End struct {
		Streams []struct {
			Udp struct {
				Socket        int     `json:"socket"`
				Start         int     `json:"start"`
				End           float64 `json:"end"`
				Seconds       float64 `json:"seconds"`
				Bytes         int     `json:"bytes"`
				BitsPerSecond float64 `json:"bits_per_second"`
				JitterMs      float64 `json:"jitter_ms"`
				LostPackets   int     `json:"lost_packets"`
				Packets       int     `json:"packets"`
				LostPercent   int     `json:"lost_percent"`
				OutOfOrder    int     `json:"out_of_order"`
				Sender        bool    `json:"sender"`
			} `json:"udp"`
		} `json:"streams"`
		Sum struct {
			Start         int     `json:"start"`
			End           float64 `json:"end"`
			Seconds       float64 `json:"seconds"`
			Bytes         int     `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			JitterMs      float64 `json:"jitter_ms"`
			LostPackets   int     `json:"lost_packets"`
			Packets       int     `json:"packets"`
			LostPercent   int     `json:"lost_percent"`
			Sender        bool    `json:"sender"`
		} `json:"sum"`
		CpuUtilizationPercent struct {
			HostTotal    float64 `json:"host_total"`
			HostUser     float64 `json:"host_user"`
			HostSystem   float64 `json:"host_system"`
			RemoteTotal  float64 `json:"remote_total"`
			RemoteUser   float64 `json:"remote_user"`
			RemoteSystem float64 `json:"remote_system"`
		} `json:"cpu_utilization_percent"`
	} `json:"end"`
	Error string `json:"error"`
}

func RunServer(cd *CheckData) error {
	osDetect := runtime.GOOS

	var cmd *exec.Cmd
	switch osDetect {
	case "windows":
		break
	case "darwin":
		targetHost := strings.Split(cd.Target, ":")
		args := []string{"-c", "./lib/iperf3_osx -s -B " + targetHost[0] + " -p " + targetHost[1]}
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

	return nil
}

func (r *IperfResults) Check(cd *CheckData) error {
	osDetect := runtime.GOOS
	r.StartTimestamp = time.Now()

	var cmd *exec.Cmd
	switch osDetect {
	case "windows":
		break
	case "darwin":
		targetHost := strings.Split(cd.Target, ":")
		args := []string{"-c", "./lib/iperf3_darwin -c " + targetHost[0] + " -p " + targetHost[1] + " -u -t " + strconv.Itoa(cd.Duration) + " -b 8K --json"}
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

	r.StopTimestamp = time.Now()
	err = json.Unmarshal(out, &r)
	if err != nil {
		return err
	}

	cd.Result = r

	return nil
}
