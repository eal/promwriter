package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

type input struct {
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
}

func mustAtoi(s string) int {
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal("invalid interval:", err)
	}
	return i
}

func main() {
	dataJSON := flag.String("data", os.Getenv("METRIC_DATA"), "metric JSON")
	url := flag.String("url", os.Getenv("REMOTE_WRITE_URL"), "remote-write URL")
	interval := flag.Int("interval", mustAtoi(os.Getenv("SEND_INTERVAL")), "seconds between sends")
	flag.Parse()

	if *dataJSON == "" || *url == "" {
		log.Fatal("both --data and --url are required")
	}

	var in input
	if err := json.Unmarshal([]byte(*dataJSON), &in); err != nil {
		log.Fatal("invalid JSON:", err)
	}

	// prepare WriteRequest
	labels := make([]prompb.Label, 0, len(in.Labels))
	for k, v := range in.Labels {
		labels = append(labels, prompb.Label{Name: k, Value: v})
	}
	wr := &prompb.WriteRequest{Timeseries: []prompb.TimeSeries{{
		Labels: labels,
		Samples: []prompb.Sample{{Value: in.Value}},
	}}}

	client := &http.Client{Timeout: 5 * time.Second}
	send := func() {
		wr.Timeseries[0].Samples[0].Timestamp = time.Now().UnixMilli()
		b, err := proto.Marshal(wr)
		if err != nil {
			log.Fatal(err)
		}
		compressed := snappy.Encode(nil, b)
		resp, err := client.Post(*url, "application/x-protobuf", bytes.NewReader(compressed))
		if err != nil {
			log.Println("send error:", err)
			return
		}
		resp.Body.Close()
		log.Println("status:", resp.StatusCode)
	}
    for {
        send()
        if *interval == 0 {
            break
        }
        time.Sleep(time.Duration(*interval) * time.Second)
    }
}
