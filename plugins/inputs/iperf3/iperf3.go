package iperf3

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

var (
	execCommand = exec.Command // execCommand is used to mock commands in tests.
)

type Iperf3 struct {
	// name of iperf3 binary file
	Binary string

	// iperf3 servers (ips or names) to connect to
	Hosts []string

	// protocol to test (tcp or udp)
	Protocol string

	// time in seconds to transmit for
	TransmitTime int `toml:"transmit_time"`
}

var sampleConfig = `
  ## iperf3 servers to test connections to  
  hosts = ["127.0.0.1"]

  ## name or path of iperf3 binary
  # binary = "iperf3"

  ## protocol to test (tcp or udp)
  # protocol = "tcp"

  ## time in seconds to transmit for
  # transmit_time = 10
`

func (f *Iperf3) Description() string {
	return "Measure network bandwidth via iperf3."
}

func (f *Iperf3) SampleConfig() string {
	return sampleConfig
}

type tcpResult struct {
	Total struct {
		Sent struct {
			Bytes         float64 `json:"bytes"`
			Seconds       float64 `json:"seconds"`
			BitsPerSecond float64 `json:"bits_per_second"`
			Retransmits   float64 `json:"retransmits"`
		} `json:"sum_sent"`
		Received struct {
			Bytes         float64 `json:"bytes"`
			Seconds       float64 `json:"seconds"`
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
	} `json:"end"`
}

type udpResult struct {
	Total struct {
		Sum struct {
			Bytes         float64 `json:"bytes"`
			Seconds       float64 `json:"seconds"`
			BitsPerSecond float64 `json:"bits_per_second"`
			JitterMS      float64 `json:"jitter_ms"`
			Packets       float64 `json:"packets"`
			LostPackets   float64 `json:"lost_packets"`
			LostPercent   float64 `json:"lost_percent"`
		}
	} `json:"end"`
}

func (ip *Iperf3) Gather(acc telegraf.Accumulator) error {
	binary := ip.Binary
	hosts := ip.Hosts
	protocol := ip.Protocol
	transmitTime := ip.TransmitTime

	for _, host := range hosts {
		var args []string
		args = append(args, "-c", host, "--json")
		args = append(args, "-t", strconv.Itoa(transmitTime))
		if protocol == "udp" {
			args = append(args, "-u")
		}

		cmd := execCommand(binary, args...)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to run command %s: %s - %s", strings.Join(cmd.Args, " "), err, string(out))
		}

		metrics := make(map[string]interface{})
		if protocol == "tcp" {
			var res tcpResult
			json.Unmarshal(out, &res)
			fmt.Println(res)
			metrics["sent_bps"] = res.Total.Sent.BitsPerSecond
			metrics["sent_retransmits"] = res.Total.Sent.Retransmits
			metrics["received_bps"] = res.Total.Received.BitsPerSecond
		} else {
			var res udpResult
			json.Unmarshal(out, &res)
			fmt.Printf("%+v", res)
			metrics["bps"] = res.Total.Sum.BitsPerSecond
			metrics["jitter_ms"] = res.Total.Sum.JitterMS
			metrics["packets"] = res.Total.Sum.Packets
			metrics["lost_packets"] = res.Total.Sum.LostPackets
			metrics["lost_percent"] = res.Total.Sum.LostPercent
		}

		tags := map[string]string{
			"host":     host,
			"protocol": protocol,
		}
		acc.AddFields("iperf3", metrics, tags)
	}
	return nil
}

// Init ensures the plugin is configured correctly.
func (ip *Iperf3) Init() error {
	path, err := exec.LookPath(ip.Binary)
	if err != nil {
		return errors.New("could not find iperf3 binary")

	}
	//fmt.Printf("found binary: %s", path)
	ip.Binary = path

	//fmt.Printf("\nin Init() %+v\n", p)
	if len(ip.Hosts) < 1 {
		return errors.New("must specify at least one target host")
	}

	if (ip.Protocol != "tcp") && (ip.Protocol != "udp") {
		return errors.New("protocol must be either udp or tcp")
	}

	return nil
}

func init() {
	// default values
	f := Iperf3{
		Binary:       "iperf3",
		Protocol:     "tcp",
		TransmitTime: 10,
	}

	//fmt.Printf("\nin init(): %+v\n", f)
	inputs.Add("iperf3", func() telegraf.Input {
		// copy config structure incase it's used by multiple measurement definitions
		f := f
		return &f
	})
}
