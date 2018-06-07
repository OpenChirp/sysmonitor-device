// This plugin contributes some simple network stats
package main

import (
	"fmt"

	"github.com/openchirp/sysmonitor-device/plugins"
	"github.com/shirou/gopsutil/net"
	"github.com/sirupsen/logrus"
)

type NetworkPlugin struct {
}

func NewPlugin() (plugins.Plugin, error) {
	p := new(NetworkPlugin)
	return p, nil
}

func (p *NetworkPlugin) GetReport(log *logrus.Entry) map[string]string {
	fail := func(msg string) map[string]string {
		return map[string]string{
			"networkplugin_error": msg,
		}
	}

	values := make(map[string]string)

	stats, err := net.IOCounters(true)
	if err != nil {
		fail(fmt.Sprint(err))
	}

	for _, s := range stats {
		name := s.Name
		values[fmt.Sprintf("%s_bytesrecv", name)] = fmt.Sprint(s.BytesRecv)
		values[fmt.Sprintf("%s_bytessend", name)] = fmt.Sprint(s.BytesSent)
		values[fmt.Sprintf("%s_errin", name)] = fmt.Sprint(s.Errin)
		values[fmt.Sprintf("%s_errout", name)] = fmt.Sprint(s.Errout)
		values[fmt.Sprintf("%s_dropin", name)] = fmt.Sprint(s.Dropin)
		values[fmt.Sprintf("%s_dropout", name)] = fmt.Sprint(s.Dropout)
		values[fmt.Sprintf("%s_packetsrecv", name)] = fmt.Sprint(s.PacketsRecv)
		values[fmt.Sprintf("%s_packetssent", name)] = fmt.Sprint(s.PacketsSent)
	}
	return values
}
