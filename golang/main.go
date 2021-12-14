package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	latestBlockHeight  *prometheus.Desc
	latestBlockTimeLag *prometheus.Desc
	numberOfPeers      *prometheus.Desc
}

type Status struct {
	status string
}

type SyncInfo struct {
	LatestBlockHeight string `json:"latest_block_height"`
	LatestBlockTime   string `json:"latest_block_time"`
}

type Result struct {
	SyncInfo SyncInfo `json:"sync_info"`
	NPeers   string   `json:"n_peers"`
}

type Root struct {
	Result Result `json:"result"`
}

var targetHost = flag.String("target", "http://localhost:26657", "Target to scrape metrics from")

func getJson(url string, target *Root) error {
	var httpClient = &http.Client{Timeout: 10 * time.Second}
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func gaiaCollector() *Collector {
	return &Collector{
		latestBlockHeight:  prometheus.NewDesc("gaia_latest_block_height", "The latest block height", nil, nil),
		latestBlockTimeLag: prometheus.NewDesc("gaia_latest_block_time_lag", "Delta in seconds between localtime and latest block time", nil, nil),
		numberOfPeers:      prometheus.NewDesc("gaia_number_of_peers", "Number of peers", nil, nil),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.latestBlockHeight
	ch <- c.latestBlockTimeLag
	ch <- c.numberOfPeers
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var responseStatus Root
	var responseNPeers Root

	if err := getJson(*targetHost+"/status", &responseStatus); err != nil {
		log.Println(err)
		return
	}

	if err := getJson(*targetHost+"/net_info", &responseNPeers); err != nil {
		log.Println(err)
		return
	}

	layout := "2006-01-02T15:04:05.000000000Z"
	t, err := time.Parse(layout, responseStatus.Result.SyncInfo.LatestBlockTime)

	if err != nil {
		log.Println(err)
		return
	}

	lag := time.Now().Sub(t)
	latestBlockHeight, err := strconv.Atoi(responseStatus.Result.SyncInfo.LatestBlockHeight)
	numberOfPeers, err := strconv.Atoi(responseNPeers.Result.NPeers)

	ch <- prometheus.MustNewConstMetric(c.latestBlockHeight, prometheus.GaugeValue, float64(latestBlockHeight))
	ch <- prometheus.MustNewConstMetric(c.latestBlockTimeLag, prometheus.GaugeValue, lag.Seconds())
	ch <- prometheus.MustNewConstMetric(c.numberOfPeers, prometheus.GaugeValue, float64(numberOfPeers))
}

func main() {
	flag.Parse()
	c := gaiaCollector()
	prometheus.MustRegister(c)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9101", nil))
}
