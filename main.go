// Craig Hesling
// November 3, 2017
//
// This is an System Monitor OpenChirp device. It will report the system status
// to an openchirp device at a scheduled interval.
package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openchirp/framework"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	version string = "1.0"
)

const (
	defaultIntervalDuration = "60s"
	defaultDiskMountPath    = "/"
	triggerTopic            = framework.TransducerPrefix + "/trigger"
	intervalTopic           = framework.TransducerPrefix + "/interval"
)

func run(ctx *cli.Context) error {
	/* Setup Parameters */
	diskPath := ctx.String("disk-path")
	intervalDuration, err := time.ParseDuration(ctx.String("interval"))
	if err != nil {
		log.Fatalf("Failed to parse interval duration %s: %v", ctx.String("interval"), err)
		return cli.NewExitError(nil, 1)
	}

	/* Setup Runtime Variables */
	intervalChange := make(chan time.Duration)

	/* Set logging level */
	log.SetLevel(log.Level(uint32(ctx.Int("log-level"))))

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
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	/* Helper Methonds */
	reportStat := func(subtopic string, value interface{}) {
		log.Debugf("Publishing %s: %s", subtopic, fmt.Sprint(value))
		err = c.Publish(framework.TransducerPrefix+"/"+subtopic, fmt.Sprint(value))
		if err != nil {
			log.Errorf("Error publishing %s: %v", subtopic, err)
		}
	}
	reportError := func(value interface{}) {
		log.Errorf(fmt.Sprint(value))
		reportStat("error", value)
	}
	doreport := func() {
		log.Info("Doing Report")
		gb := math.Pow(1024, 3)

		v, err := mem.VirtualMemory()
		if err != nil {
			reportError(fmt.Sprintf("Failed to retrieve memory usage: %v", err))
		} else {
			reportStat("mem_total", float64(v.Total)/gb)
			reportStat("mem_avaliable", float64(v.Available)/gb)
			reportStat("mem_used", float64(v.Used)/gb)
			reportStat("mem_usedpercent", v.UsedPercent)
		}

		d, err := disk.Usage(diskPath)
		if err != nil {
			reportError(fmt.Sprintf("Failed to retrieve disk usage for %s: %v", ctx.String("disk-path"), err))
		} else {
			reportStat("disk_used", float64(d.Used)/gb)
			reportStat("disk_free", float64(d.Free)/gb)
			reportStat("disk_total", float64(d.Total)/gb)
			reportStat("disk_usedpercent", v.UsedPercent)
		}

		l, err := load.Avg()
		if err != nil {
			reportError(fmt.Sprintf("Failed to retrieve cpu load: %v", err))
		} else {
			reportStat("load_1min", l.Load1)
			reportStat("load_5min", l.Load5)
			reportStat("load_15min", l.Load15)
		}
	}

	/* Subscribe to trigger topic */
	err = c.Subscribe(triggerTopic, func(topic string, payload []byte) {
		log.Info("Received trigger to push report")
		doreport()
	})
	if err != nil {
		log.Fatalf("Error subscribing to open topic: %v", err)
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

	doreport()

	for {
		select {
		case <-time.After(intervalDuration):
			doreport()
		case interval := <-intervalChange:
			intervalDuration = interval
			log.Info("Changing interval to ", intervalDuration)
			doreport()
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
	app.Copyright = "See https://github.com/openchirp/example-device for copyright information"
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
	}
	app.Run(os.Args)
}
