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
	path  string
	Hosts []string `toml:"hosts"`
}

var sampleConfig = `
  hosts = ['127.0.0.1']
`

func (f *Iperf3) Description() string {
	return "Measure network bandwidth via iperf3."
}

func (f *Iperf3) SampleConfig() string {
	return sampleConfig
}

type iperf3TCPResult struct {
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
	if len(f.path) == 0 {
		return errors.New("iperf3 not found: verify that iperf3 is installed and in your PATH")
	}

	name := f.path
	hosts := f.Hosts

	for _, host := range hosts {
		var arg []string
		args := append(arg, "-c", host, "-t", "1", "--json")
		tags := make(map[string]string)
		cmd := execCommand(name, args...)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to run command %s: %s - %s", strings.Join(cmd.Args, " "), err, string(out))
		}
		var tcpRes iperf3TCPResult
		json.Unmarshal(out, &tcpRes)
		//fmt.Println(tcpRes)
		metrics := make(map[string]interface{})
		metrics["bps"] = tcpRes.Total.Received.Bytes / tcpRes.Total.Received.Seconds
		tags["host"] = host
		acc.AddFields("iperf3", metrics, tags)
	}
	return nil
}

func init() {
	f := Iperf3{}
	path, _ := exec.LookPath("iperf3")
	if len(path) > 0 {
		f.path = path
	}
	inputs.Add("iperf3", func() telegraf.Input {
		// copy config structure incase it's used by multiple measurement definitions
		f := f
		return &f
	})
}
