// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

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

type Inputs struct {
	configFilePath string
	showVersion    bool
}

type PlainFormatter struct {
	TimestampFormat string
}

const (
	DEFAULT_CONFIG_FILE_PATH = "config.yaml"
	VERSION                  = "v1.0.0-beta"
)

var (
	DEFAULT_LOG_FILE = filepath.Join("var", "log", "SmartOverloadProtection.log")
	energyGuard      *ems.EnergyGuard
)

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s %s\n", timestamp, strings.ToUpper(entry.Level.String()), entry.Message)), nil
}

func parseCommandLine() (inputs Inputs) {
	flag.StringVar(&inputs.configFilePath, "configFilePath", DEFAULT_CONFIG_FILE_PATH, "Set the configuration file path")
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
	log.SetLevel(logConfig.Level)

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

	if err != nil {
		log.Fatalf("Cannot create log: %v", err)
	}

	outputs := io.MultiWriter(logWriter, os.Stdout)
	log.SetOutput(outputs)

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
