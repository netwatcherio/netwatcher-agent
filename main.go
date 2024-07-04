package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/probes"
	"github.com/netwatcherio/netwatcher-agent/workers"
	"github.com/netwatcherio/netwatcher-agent/ws"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	fmt.Printf("Starting NetWatcher Agent...\n")

	var configPath string
	flag.StringVar(&configPath, "config", "./config.conf", "Path to the config file")
	flag.Parse()

	loadConfig(configPath)

	// Download dependency
	err := downloadTrippyDependency()
	if err != nil {
		log.Fatalf("Failed to download dependency: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			shutdown()
			return
		}
	}()

	var probeGetCh = make(chan []probes.Probe)
	var probeDataCh = make(chan probes.ProbeData)

	wsH := &ws.WebSocketHandler{
		Host:       os.Getenv("HOST"),
		HostWS:     os.Getenv("HOST_WS"),
		Pin:        os.Getenv("PIN"),
		ID:         os.Getenv("ID"),
		ProbeGetCh: probeGetCh,
	}
	wsH.InitWS()
	// init the config getter before starting the probe workers?
	workers.InitProbeDataWorker(wsH, probeDataCh)

	go func(ws *ws.WebSocketHandler) {
		for {
			time.Sleep(time.Minute * 1)
			log.Info("Getting probes again...")
			ws.GetConnection().Emit("probe_get", []byte("please"))
		}
	}(wsH)

	thisAgent, err := primitive.ObjectIDFromHex(wsH.ID)
	if err != nil {
		return
	}

	workers.InitProbeWorker(probeGetCh, probeDataCh, thisAgent)

	// todo handle if on start it isn't able to pull information from backend??
	// eg. power goes out but network fails to come up?

	// todo input channel into wsH for inbound/outbound data to be handled
	// if a list of probes is received, send it to the channel for inbound probes and such
	// once receiving probes, have it cycle through, set the unique id for it, if a different one exists as the same ID,
	//update/remove it, n use the new settings
	select {}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}

func downloadTrippyDependency() error {
	var version = "0.10.0"
	baseURL := "https://github.com/fujiapple852/trippy/releases/download/" + version + "/"

	var fileName, extractedName string

	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			fileName = "trippy-VER-x86_64-pc-windows-msvc.exe"
		} else {
			fileName = "trippy-VER-i686-pc-windows-msvc.exe"
		}
		extractedName = fileName
	case "darwin":
		fileName = "trippy-VER-x86_64-apple-darwin.tar.gz"
		extractedName = "trip"
	case "linux":
		if runtime.GOARCH == "amd64" {
			fileName = "trippy-VER-x86_64-unknown-linux-musl.tar.gz"
		} else if runtime.GOARCH == "arm64" {
			fileName = "trippy-VER-aarch64-unknown-linux-musl.tar.gz"
		} else {
			return fmt.Errorf("unsupported Linux architecture: %s", runtime.GOARCH)
		}
		extractedName = "trip"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	var format = strings.Replace(fileName, "VER", version, -1)
	url := baseURL + format
	libPath := filepath.Join(".", "lib")
	os.MkdirAll(libPath, os.ModePerm)
	filePath := filepath.Join(libPath, extractedName)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		log.Printf("Trippy binary already exists: %s\n", filePath)
		return nil
	}

	log.Printf("Downloading %s to %s\n", url, filePath)

	// Download file
	tempFilePath := filePath + ".temp"
	err := downloadFile(url, tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	var newHash string
	if runtime.GOOS != "windows" {
		// Extract the tar.gz for Linux and macOS
		newHash, err = extractTarGzAndHash(tempFilePath, libPath)
		if err != nil {
			os.Remove(tempFilePath)
			return fmt.Errorf("failed to extract archive: %v", err)
		}
		// Remove the temporary tar.gz file
		os.Remove(tempFilePath)
	} else {
		// For Windows, just rename the downloaded file and get its hash
		err = os.Rename(tempFilePath, filePath)
		if err != nil {
			os.Remove(tempFilePath)
			return fmt.Errorf("failed to rename file: %v", err)
		}
		newHash, err = getFileHash(filePath)
		if err != nil {
			return fmt.Errorf("failed to get file hash: %v", err)
		}
	}

	log.Printf("Downloaded trippy binary: %s\n", filePath)

	// Make the file executable
	err = os.Chmod(filePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to make file executable: %v", err)
	}

	// Store the hash for future comparisons
	err = os.WriteFile(filePath+".hash", []byte(newHash), 0644)
	if err != nil {
		log.Printf("Failed to write hash file: %v", err)
	}

	return nil
}

func extractTarGzAndHash(archivePath, destPath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "trip" {
			outPath := filepath.Join(destPath, "trip")
			outFile, err := os.Create(outPath)
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			hasher := sha256.New()
			writer := io.MultiWriter(outFile, hasher)

			if _, err := io.Copy(writer, tr); err != nil {
				return "", err
			}

			log.Printf("Extracted file: %s\n", header.Name)
			return hex.EncodeToString(hasher.Sum(nil)), nil
		}
	}

	return "", fmt.Errorf("'trip' binary not found in archive")
}

func getFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func isUpdateNeeded(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return true
	}

	hashFile := filePath + ".hash"
	_, err = os.Stat(hashFile)
	if os.IsNotExist(err) {
		return true
	}

	// If both files exist, assume it's up to date
	return false
}

func downloadFile(url string, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
