// This plugin contributes some simple network stats
package main

import (
	"fmt"

	"github.com/openchirp/sysmonitor-device/plugins"
	"github.com/shirou/gopsutil/net"
	"github.com/sirupsen/logrus"
)

const topicPrefix = "net_"

// NetworkPlugin holds the context for the Network stats plugin
type NetworkPlugin struct {
}

// NewPlugin initializes the Network stats plugin
func NewPlugin() (plugins.Plugin, error) {
	p := new(NetworkPlugin)
	return p, nil
}

// GetReport returns the report with Network stats
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
		values[fmt.Sprintf("%s%s_bytesrecv", topicPrefix, name)] = fmt.Sprint(s.BytesRecv)
		values[fmt.Sprintf("%s%s_bytessend", topicPrefix, name)] = fmt.Sprint(s.BytesSent)
		values[fmt.Sprintf("%s%s_errin", topicPrefix, name)] = fmt.Sprint(s.Errin)
		values[fmt.Sprintf("%s%s_errout", topicPrefix, name)] = fmt.Sprint(s.Errout)
		values[fmt.Sprintf("%s%s_dropin", topicPrefix, name)] = fmt.Sprint(s.Dropin)
		values[fmt.Sprintf("%s%s_dropout", topicPrefix, name)] = fmt.Sprint(s.Dropout)
		values[fmt.Sprintf("%s%s_packetsrecv", topicPrefix, name)] = fmt.Sprint(s.PacketsRecv)
		values[fmt.Sprintf("%s%s_packetssent", topicPrefix, name)] = fmt.Sprint(s.PacketsSent)
	}
	return values
}
