package main

import (
	"errors"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

const (
	configFile    = "./config.conf"
	defaultConfig = "API_URL=*PUT URL HERE*\nPIN=*PUT PIN HERE*\nHASH=\n"
)

const VERSION = "0.1.0rc1"

func setup() error {
	log.Printf("NetWatcher v%s - Copyright (c) 2021-%d Shaun Agostinho\n", VERSION, time.Now().Year())
	// Check if the config file exists in the local directory
	_, err := os.Stat(configFile)
	// If the check returns an error indicating the file doesn't exist, create it
	if errors.Is(err, os.ErrNotExist) {
		// Log to terminal that a new file will be created
		log.Warnf("Config file does '%s' does not exist, creating one now.", configFile)
		// Attempt to create the config file
		_, err = os.Create(configFile)
		if err != nil {
			return err
		}
		// Attempt to write the default config pattern to the config file
		err = os.WriteFile("./config.conf", []byte(defaultConfig), 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// Log the current production mode to console
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		log.Infof("Running in PRODUCTION mode.")
	} else {
		log.Infof("Running in %sDEVELOPMENT%s mode.", "\u001B[1;33m", "\033[m")
	}
	// Attempt to load the config file
	err = godotenv.Load("config.conf")
	if err != nil {
		return err
	}
	// Return normally
	return nil
}
