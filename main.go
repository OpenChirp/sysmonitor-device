// Craig Hesling
// November 3, 2017
//
// This is an example OpenChirp device. It sets up arguments and runs a door
// controller
package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openchirp/framework"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	version string = "1.0"
)

func run(ctx *cli.Context) error {
	/* Set logging level */
	log.SetLevel(log.Level(uint32(ctx.Int("log-level"))))

	log.Info("Starting Example Device")

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
	// log.Info("Processing device updates")
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	/* Subscribe to door open topic */
	err = c.Subscribe(framework.TransducerPrefix+"/open", func(topic string, payload []byte) {
		value := string(payload)
		log.Infof("Received open command %s", value)
	})
	if err != nil {
		log.Fatalf("Error subscribing to open topic: %v", err)
		return cli.NewExitError(nil, 1)
	}

	for {
		select {
		case <-time.After(time.Second * time.Duration(2)):
			status := rand.Uint32() % 2
			log.Debugf("Publishing status %v", status)
			err = c.Publish(framework.TransducerPrefix+"/status", fmt.Sprint(status))
			if err != nil {
				log.Errorf("Error publishing status %v: %v", status, err)
			}
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
	}
	app.Run(os.Args)
}
