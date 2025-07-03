package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"time"
)

const (
	defaultConfig = "HOST=https://api.netwatcher.io\nHOST_WS=wss://api.netwatcher.io/agent_ws\nID=\nPIN=\n"
)

var (
	// This will be set at build time using -ldflags
	buildDate string
	VERSION   string
)

func getExecutableHash() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	fileBytes, err := os.ReadFile(exePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(fileBytes)
	return hex.EncodeToString(hash[:8]), nil // Use first 8 bytes (16 hex chars) for brevity
}

func loadConfig(configFile string) error {
	hash, err := getExecutableHash()
	if err != nil {
		hash = "unknown"
	}

	versionStr := fmt.Sprintf("hash_%s", hash)
	if buildDate != "" {
		versionStr += "_" + buildDate
	}

	VERSION = versionStr

	fmt.Printf("NetWatcher v%s - Copyright (c) 2024-%d Shaun Agostinho\n", versionStr, time.Now().Year())

	_, err = os.Stat(configFile)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Config file '%s' does not exist, creating one now.\n", configFile)
		_, err = os.Create(configFile)
		if err != nil {
			return err
		}
		err = os.WriteFile(configFile, []byte(defaultConfig), 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		fmt.Printf("Running in PRODUCTION mode.\n")
	} else {
		fmt.Printf("Running in DEVELOPMENT mode.\n")
	}

	err = godotenv.Load(configFile)
	if err != nil {
		return err
	}

	return nil
}
