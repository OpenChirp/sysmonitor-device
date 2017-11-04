// This plugin simply reports the Raspberry Pi Core Temp
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	vcgencmdPath = "/usr/bin/vcgencmd"
)

func getTemp() (float64, error) {
	if _, err := os.Stat(vcgencmdPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("Cannot find vcgencmd command")
	}

	// cmd := exec.Command("echo", "temp=47.2'C")
	cmd := exec.Command(vcgencmdPath, "measure_temp")

	// cmd.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("vcgencmd command returned bad exit status")
	}
	pieces := strings.Split(out.String(), "=")
	if len(pieces) != 2 {
		return 0, fmt.Errorf("Failed to parse vcgencmd command")
	}
	pieces = strings.Split(pieces[1], "'")
	if len(pieces) != 2 {
		return 0, fmt.Errorf("Failed to parse vcgencmd command")
	}
	temp, err := strconv.ParseFloat(pieces[0], 64)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse vcgencmd command")
	}
	return temp, nil
}

func GetReport(log *logrus.Entry) map[string]string {
	temp, err := getTemp()
	if err != nil {
		return map[string]string{
			"raspberrypi_error": fmt.Sprint(err),
		}
	}

	return map[string]string{
		"coretemp": fmt.Sprint(temp),
	}
}
