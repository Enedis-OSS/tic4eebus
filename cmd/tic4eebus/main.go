// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

/*
tic4eebus handles electrical vehicle charge according to available energy provided by the Linky meter.
It uses a configuration file to change application behaviour.

Usage:

	tic4eebus [flags]

The flags are:

	-configFilePath string
		Set the configuration file path (default "examples/config.yaml")
	-help
		Show help
	-version
		Show software version

When tic4eebus starts, it connects to the Eebus wallbox and receive the Linky meter
data using TIC2WebSocket API periodically (from 1 to 3 seconds) using configurations parameters specified.
According to the available energy, it decides to increase or decrease the charge of the electric vehicle.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Enedis-OSS/tic4eebus/config"
	"github.com/Enedis-OSS/tic4eebus/ems"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

// Command line inputs.
type Inputs struct {
	configFilePath string
	showVersion    bool
}

// Log formatter
type PlainFormatter struct {
	TimestampFormat string
}

const (
	VERSION = "v1.1.0"
)

var (
	energyGuard *ems.EnergyGuard
)

// Format renders a single log entry with timestamp Level and message.
func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s %s\n", timestamp, strings.ToUpper(entry.Level.String()), entry.Message)), nil
}

func parseCommandLine() (inputs Inputs) {
	flag.StringVar(&inputs.configFilePath, "configFilePath", filepath.Join("examples", "config.yaml"), "Set the configuration file path")
	flag.BoolVar(&inputs.showVersion, "version", false, "Show software version")
	flag.Parse()

	if inputs.showVersion {
		applicationName := filepath.Base(os.Args[0])
		fmt.Println(applicationName + " " + VERSION)
		os.Exit(0)
	}

	return inputs
}

func initLogger(logConfig config.LogConfig) {
	// Set log level
	log.SetLevel(logConfig.Level)
	// Compute log rotation parameters
	logFileExtension := filepath.Ext(logConfig.FilePath)
	logFilePathWithoutExtension := strings.TrimSuffix(logConfig.FilePath, logFileExtension)
	logFilePathWithPattern := logFilePathWithoutExtension + logConfig.Rotation.PeriodPattern + logFileExtension
	logRotationTime := time.Duration(float64(logConfig.Rotation.PeriodInHours) * float64(time.Hour))
	logMaxAge := time.Duration(float64(logConfig.Rotation.PeriodCount) * float64(logConfig.Rotation.PeriodInHours) * float64(time.Hour))
	// Configure logs with rotation matching with config
	logWriter, err := rotatelogs.New(
		logFilePathWithPattern,
		rotatelogs.WithMaxAge(logMaxAge),
		rotatelogs.WithRotationTime(logRotationTime),
	)
	// Check log creation
	if err != nil {
		log.Fatalf("Cannot create log: %v", err)
	}
	// Set log output to file and stdout
	outputs := io.MultiWriter(logWriter, os.Stdout)
	log.SetOutput(outputs)
	// Set log format
	log.SetFormatter(&PlainFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func init() {
	inputs := parseCommandLine()
	configData, err := config.LoadConfig(inputs.configFilePath)
	if err != nil {
		log.Fatal(err)
	}
	initLogger(configData.Log)

	energyGuard = ems.NewEnergyGuard(configData)
}

func start() {
	energyGuard.Start()
}

func stop() {
	energyGuard.Stop()
}

func waitExit() {
	// Clean exit to make sure mdns shutdown is invoked
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	// User exit
}

func main() {
	start()
	waitExit()
	stop()
}
