// +build arm

// This plugin simply reports the Raspberry Pi Core Temp
package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/openchirp/sysmonitor-device/plugins"
	"github.com/sirupsen/logrus"
)

const (
	// vcgencmdPath = "/usr/bin/vcgencmd"
	vcgencmdPath = "vcgencmd"
)

type RaspberryPlugin struct {
	cmdPath string
}

func NewPlugin() (plugins.Plugin, error) {
	p := new(RaspberryPlugin)

	cmdPath, err := exec.LookPath(vcgencmdPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot find vcgencmd command")
	}
	p.cmdPath = cmdPath
	return p, nil
}

func (p *RaspberryPlugin) GetReport(log *logrus.Entry) map[string]string {
	fail := func(msg string) map[string]string {
		return map[string]string{
			"raspberrypi_error": msg,
		}
	}

	cmd := exec.Command(p.cmdPath, "measure_temp")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return fail("vcgencmd command returned bad exit status")
	}
	pieces := strings.Split(out.String(), "=")
	if len(pieces) != 2 {
		return fail("Failed to parse vcgencmd command")
	}
	pieces = strings.Split(pieces[1], "'")
	if len(pieces) != 2 {
		return fail("Failed to parse vcgencmd command")
	}
	temp, err := strconv.ParseFloat(pieces[0], 64)
	if err != nil {
		return fail("Failed to parse vcgencmd command")
	}

	return map[string]string{
		"coretemp": fmt.Sprint(temp),
	}
}
