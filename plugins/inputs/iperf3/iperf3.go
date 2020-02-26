package iperf3

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

var (
	execCommand = exec.Command // execCommand is used to mock commands in tests.
)

type Iperf3 struct {
	Binary string
	Hosts  []string `toml:"hosts"`
}

var sampleConfig = `
  binary = "iperf3"
  hosts = ["127.0.0.1"]
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
			Bytes   float64 `json:"bytes"`
			Seconds float64 `json:"seconds"`
		} `json:"sum_sent"`
		Received struct {
			Bytes   float64 `json:"bytes"`
			Seconds float64 `json:"seconds"`
		} `json:"sum_received"`
	} `json:"end"`
}

func (f *Iperf3) Gather(acc telegraf.Accumulator) error {
	binary := f.Binary
	hosts := f.Hosts

	for _, host := range hosts {
		var arg []string
		args := append(arg, "-c", host, "-t", "1", "--json")
		tags := make(map[string]string)
		cmd := execCommand(binary, args...)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to run command %s: %s - %s", strings.Join(cmd.Args, " "), err, string(out))
		}
		var tcpRes tcpResult
		json.Unmarshal(out, &tcpRes)
		fmt.Println(tcpRes)
		metrics := make(map[string]interface{})
		metrics["bps"] = tcpRes.Total.Received.Bytes / tcpRes.Total.Received.Seconds
		tags["host"] = host
		acc.AddFields("iperf3", metrics, tags)
	}
	return nil
}

// Init ensures the plugin is configured correctly.
func (p *Iperf3) Init() error {
	path, err := exec.LookPath(p.Binary)
	if err != nil {
		return errors.New("could not find iperf3 binary")

	}
	//fmt.Printf("found binary: %s", path)
	p.Binary = path

	//fmt.Printf("\nin Init() %+v\n", p)
	if len(p.Hosts) < 1 {
		return errors.New("must specify at least one target host")
	}

	return nil
}

func init() {
	// default values
	f := Iperf3{
		Binary: "iperf3",
	}

	//fmt.Printf("\nin init(): %+v\n", f)
	inputs.Add("iperf3", func() telegraf.Input {
		// copy config structure incase it's used by multiple measurement definitions
		f := f
		return &f
	})
}
