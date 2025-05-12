package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"net/http"
)

type Input struct {
	Value     float64           `json:"value"`
	Timestamp int64             `json:"timestamp,omitempty"`
	Labels    map[string]string `json:"labels"`
}

func getenvOrFlag(name, fallback string) string {
	env := os.Getenv(name)
	if env != "" {
		return env
	}
	return fallback
}

func parseFlags() (Input, string, time.Duration) {
	var (
		dataFlag    = flag.String("data", "", "Metric JSON string")
		urlFlag     = flag.String("url", "", "Remote write URL")
		intervalStr = flag.String("interval", "", "Repeat interval in seconds")
	)
	flag.Parse()

	dataStr := getenvOrFlag("METRIC_DATA", *dataFlag)
	url := getenvOrFlag("REMOTE_WRITE_URL", *urlFlag)
	intervalRaw := getenvOrFlag("SEND_INTERVAL", *intervalStr)

	if dataStr == "" || url == "" {
		fmt.Fprintln(os.Stderr, "data and url are required")
		os.Exit(1)
	}

	var input Input
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		panic(fmt.Errorf("failed to parse --data: %w", err))
	}
	if input.Timestamp == 0 {
		input.Timestamp = time.Now().UnixMilli()
	}

	var interval time.Duration
	if intervalRaw != "" {
		secs, err := strconv.Atoi(intervalRaw)
		if err != nil {
			panic("invalid interval")
		}
		interval = time.Duration(secs) * time.Second
	}
	return input, url, interval
}

func send(input Input, url string) error {
	labels := make([]prompb.Label, 0, len(input.Labels))
	for k, v := range input.Labels {
		labels = append(labels, prompb.Label{Name: k, Value: v})
	}
	ts := prompb.TimeSeries{
		Labels: labels,
		Samples: []prompb.Sample{{
			Value:     input.Value,
			Timestamp: time.Now().UnixMilli(),
		}},
	}
	req := &prompb.WriteRequest{Timeseries: []prompb.TimeSeries{ts}}

	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	compressed := snappy.Encode(nil, data)

	resp, err := http.Post(url, "application/x-protobuf", bytes.NewReader(compressed))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Printf("Status: %d\n", resp.StatusCode)
	return nil
}

func main() {
	input, url, interval := parseFlags()

	if interval == 0 {
		if err := send(input, url); err != nil {
			panic(err)
		}
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := send(input, url); err != nil {
			fmt.Fprintln(os.Stderr, "send failed:", err)
		}
		<-ticker.C
	}
}
