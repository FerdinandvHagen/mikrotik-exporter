package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

// /interface/monitor-traffic

type bandwidthCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newBandwidthCollector() routerOSCollector {
	c := &bandwidthCollector{}
	c.init()
	return c
}

func (c *bandwidthCollector) init() {
	c.props = []string{"name", "rx-packets-per-second", "rx-bits-per-second", "tx-bits-per-second", "tx-packets-per-second"}

	labelNames := []string{"name", "address", "interface", "type", "disabled", "comment", "running", "slave"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("interface", p, labelNames)
	}
}

func (c *bandwidthCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *bandwidthCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *bandwidthCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/interface/monitor-traffic", "=interface=uplink", "=once=")

	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *bandwidthCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range c.props[1:] {
		fmt.Println("collectMetricForProperty", p)
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *bandwidthCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]

	if value := re.Map[property]; value != "" {
		var (
			v   int64
			err error
		)
		switch property {
		case "running":
			if value == "true" {
				v = 1
			} else {
				v = 0
			}
		default:
			v, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				log.WithFields(log.Fields{
					"device":    ctx.device.Name,
					"interface": re.Map["name"],
					"property":  property,
					"value":     value,
					"error":     err,
				}).Error("error parsing interface metric value")
				return
			}
		}

		fmt.Println("Sending metric...")

		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(v), ctx.device.Name, ctx.device.Address,
			re.Map["name"], re.Map["type"], re.Map["disabled"], re.Map["comment"], re.Map["running"], re.Map["slave"])
	}
}
