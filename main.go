// Craig Hesling
// November 3, 2017
//
// This is an System Monitor OpenChirp device. It will report the system status
// to an openchirp device at a scheduled interval.
package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/openchirp/framework"
	"github.com/openchirp/sysmonitor-device/plugins"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wercker/journalhook"
)

const (
	version string = "2.0"
)

const (
	defaultIntervalDuration = "60s"
	defaultDiskMountPath    = "/"
	triggerTopic            = "trigger"
	intervalTopic           = "interval"
)

func run(ctx *cli.Context) error {
	/* Setup Logging */
	log := logrus.New()
	log.SetLevel(logrus.Level(uint32(ctx.Int("log-level"))))
	if ctx.Bool("systemd") {
		log.AddHook(&journalhook.JournalHook{})
		log.Out = ioutil.Discard
	}

	/* Setup Parameters */
	diskPath := ctx.String("disk-path")
	intervalDuration, err := time.ParseDuration(ctx.String("interval"))
	if err != nil {
		log.Fatalf("Failed to parse interval duration %s: %v", ctx.String("interval"), err)
		return cli.NewExitError(nil, 1)
	}
	pluginPathStr := strings.TrimSpace(ctx.String("plugin-paths"))
	pluginPaths := make([]string, 0)
	if pluginPathStr != "" {
		pluginPaths = strings.Split(pluginPathStr, ";")
	}
	pluginManager, err := plugins.NewPluginManger(pluginPaths)
	if err != nil {
		log.Fatalf("Failed to setup plugin manager: %v", err)
	}

	/* Setup Runtime Variables */
	intervalChange := make(chan time.Duration)

	log.Info("Starting System Monitor Device with interval of ", intervalDuration)

	/* Start framework service client */
	c, err := framework.StartDeviceClient(
		ctx.String("framework-server"),
		ctx.String("mqtt-server"),
		ctx.String("device-id"),
		ctx.String("device-token"))
	if err != nil {
		log.Error("Failed to StartDeviceClient: ", err)
		return cli.NewExitError(nil, 1)
	}
	defer c.StopClient()
	log.Info("Started device")

	/* Setup signal channel */
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	/* Helper Methods */
	reportStat := func(subtopic string, value interface{}) {
		log.Debugf("Publishing %s: %s", subtopic, fmt.Sprint(value))
		err = c.Publish(subtopic, fmt.Sprint(value))
		if err != nil {
			log.Errorf("Error publishing %s: %v", subtopic, err)
		}
	}
	reportError := func(value interface{}) {
		log.Errorf(fmt.Sprint(value))
		reportStat("error", value)
	}
	doReport := func() {
		log.Info("Doing Report")
		gb := math.Pow(1024, 3)

		v, err := mem.VirtualMemory()
		if err != nil {
			reportError(fmt.Sprintf("Failed to retrieve memory usage: %v", err))
		} else {
			reportStat("mem_total", float64(v.Total)/gb)
			reportStat("mem_available", float64(v.Available)/gb)
			reportStat("mem_used", float64(v.Used)/gb)
			reportStat("mem_usedpercent", v.UsedPercent)
		}

		d, err := disk.Usage(diskPath)
		if err != nil {
			reportError(fmt.Sprintf("Failed to retrieve disk usage for %s: %v", diskPath, err))
		} else {
			reportStat("disk_used", float64(d.Used)/gb)
			reportStat("disk_free", float64(d.Free)/gb)
			reportStat("disk_total", float64(d.Total)/gb)
			reportStat("disk_usedpercent", d.UsedPercent)
		}

		l, err := load.Avg()
		if err != nil {
			reportError(fmt.Sprintf("Failed to retrieve cpu load: %v", err))
		} else {
			reportStat("load_1min", l.Load1)
			reportStat("load_5min", l.Load5)
			reportStat("load_15min", l.Load15)
		}

		reports := pluginManager.GetReports(log)
		for topic, report := range reports {
			reportStat(topic, report)
		}
	}

	/* Publish current interval */
	err = c.Publish(intervalTopic, fmt.Sprint(intervalDuration))
	if err != nil {
		log.Fatalf("Error publishing to interval topic: %v", err)
		return cli.NewExitError(nil, 1)
	}

	/* Subscribe to trigger topic */
	err = c.Subscribe(triggerTopic, func(topic string, payload []byte) {
		log.Info("Received trigger to push report")
		doReport()
	})
	if err != nil {
		log.Fatalf("Error subscribing to trigger topic: %v", err)
		return cli.NewExitError(nil, 1)
	}

	/* Subscribe to interval topic */
	err = c.Subscribe(intervalTopic, func(topic string, payload []byte) {
		strInterval := string(payload)
		log.Debug("Received interval change of ", strInterval)
		interval, err := time.ParseDuration(strInterval)
		if err != nil {
			reportError(fmt.Sprintf("Failed to parse interval \"%s\": %v", strInterval, err))
			return
		}

		intervalChange <- interval

	})
	if err != nil {
		log.Fatalf("Error subscribing to open topic: %v", err)
		return cli.NewExitError(nil, 1)
	}

	doReport()

	for {
		select {
		case <-time.After(intervalDuration):
			doReport()
		case interval := <-intervalChange:
			intervalDuration = interval
			log.Info("Changing interval to ", intervalDuration)
			doReport()
		case sig := <-signals:
			log.WithField("signal", sig).Info("Received signal")
			goto cleanup
		}
	}

cleanup:
	log.Warning("Shutting down")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "example-device"
	app.Usage = ""
	app.Copyright = "See https://github.com/openchirp/sysmonitor-device for copyright information"
	app.Version = version
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "framework-server",
			Usage:  "OpenChirp framework server's URI",
			Value:  "http://localhost:7000",
			EnvVar: "FRAMEWORK_SERVER",
		},
		cli.StringFlag{
			Name:   "mqtt-server",
			Usage:  "MQTT server's URI (e.g. scheme://host:port where scheme is tcp or tls)",
			Value:  "tls://localhost:1883",
			EnvVar: "MQTT_SERVER",
		},
		cli.StringFlag{
			Name:   "device-id",
			Usage:  "OpenChirp device id",
			EnvVar: "DEVICE_ID",
		},
		cli.StringFlag{
			Name:   "device-token",
			Usage:  "OpenChirp device token",
			EnvVar: "DEVICE_TOKEN",
		},
		cli.IntFlag{
			Name:   "log-level",
			Value:  4,
			Usage:  "debug=5, info=4, warning=3, error=2, fatal=1, panic=0",
			EnvVar: "LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "interval",
			Value:  defaultIntervalDuration,
			Usage:  "Reporting interval in as Golang parseable duration. (60s or 1h45m)",
			EnvVar: "INTERVAL",
		},
		cli.StringFlag{
			Name:   "disk-path",
			Value:  defaultDiskMountPath,
			Usage:  "The mount point of the disk to monitor",
			EnvVar: "DISK_PATH",
		},
		cli.StringFlag{
			Name:   "plugin-paths",
			Value:  "",
			Usage:  "List of plugin file paths separated by a semicolon",
			EnvVar: "PLUGIN_PATHS",
		},
		cli.BoolFlag{
			Name:   "systemd",
			Usage:  "Indicates that this service can use systemd specific interfaces.",
			EnvVar: "SYSTEMD",
		},
	}
	app.Run(os.Args)
}
