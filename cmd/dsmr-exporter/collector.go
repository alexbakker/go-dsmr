package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/alexbakker/go-dsmr/dsmr"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	r     *dsmr.Reader
	frame *dsmr.Frame
	m     sync.RWMutex

	descs              map[string]*Desc
	descConnectionInfo *prometheus.Desc
	descDeviceInfo     *prometheus.Desc
}

type Desc struct {
	ValueType   prometheus.ValueType
	Subsystem   string
	Name        string
	Help        string
	Labels      []string
	labelValues []string
	desc        *prometheus.Desc
}

func NewCollector(r *dsmr.Reader, deviceName string) *Collector {
	return &Collector{
		r: r,
		descConnectionInfo: prometheus.NewDesc(
			prometheus.BuildFQName("dsmr", "connection", "info"),
			"Information about the connection to the smart meter",
			[]string{"protocol_header", "protocol_version"},
			prometheus.Labels{
				"serial_device": deviceName,
			},
		),
		descDeviceInfo: prometheus.NewDesc(
			prometheus.BuildFQName("dsmr", "device", "info"),
			"Information about devices connected to the smart meter",
			[]string{"channel", "device_id", "device_type"},
			prometheus.Labels{},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	c.m.RLock()
	defer c.m.RUnlock()

	if c.frame == nil {
		ch <- prometheus.NewInvalidDesc(errors.New("no metrics collected yet from serial port"))
		return
	}

	if c.descs == nil {
		c.descs = make(map[string]*Desc)
		c.createDescMulti(map[string][]string{
			"1-0:1.8.1": {"low"},
			"1-0:1.8.2": {"normal"},
		}, &Desc{
			ValueType: prometheus.CounterValue,
			Subsystem: "electricity",
			Name:      "power_delivered_kwh_total",
			Help:      "Meter Reading electricity delivered to client in 0.001 kWh",
			Labels:    []string{"tarrif"},
		})
		c.createDesc("1-0:1.7.0", &Desc{
			ValueType: prometheus.GaugeValue,
			Subsystem: "electricity",
			Name:      "power_delivered_kw",
			Help:      "Instantaneous power delivered (+P) in 1 Watt resolution",
		})
		c.createDesc("1-0:2.7.0", &Desc{
			ValueType: prometheus.GaugeValue,
			Subsystem: "electricity",
			Name:      "power_received_kw",
			Help:      "Instantaneous power received (-P) in 1 Watt resolution",
		})
		c.createDescMulti(map[string][]string{
			"1-0:32.7.0": {"l1"},
			"1-0:52.7.0": {"l2"},
			"1-0:72.7.0": {"l3"},
		}, &Desc{
			ValueType: prometheus.GaugeValue,
			Subsystem: "electricity",
			Name:      "phase_voltage_v",
			Help:      "Instantaneous phase voltage in V resolution",
			Labels:    []string{"phase"},
		})
		c.createDescMulti(map[string][]string{
			"1-0:31.7.0": {"l1"},
			"1-0:51.7.0": {"l2"},
			"1-0:71.7.0": {"l3"},
		}, &Desc{
			ValueType: prometheus.GaugeValue,
			Subsystem: "electricity",
			Name:      "phase_current_a",
			Help:      "Instantaneous phase current in A resolution",
			Labels:    []string{"phase"},
		})
		c.createDescMulti(map[string][]string{
			"1-0:21.7.0": {"l1"},
			"1-0:41.7.0": {"l2"},
			"1-0:61.7.0": {"l3"},
		}, &Desc{
			ValueType: prometheus.GaugeValue,
			Subsystem: "electricity",
			Name:      "phase_power_delivered_kw",
			Help:      "Instantaneous phase power delivered (+P) in 1 Watt resolution",
			Labels:    []string{"phase"},
		})
		c.createDescMulti(map[string][]string{
			"1-0:22.7.0": {"l1"},
			"1-0:42.7.0": {"l2"},
			"1-0:62.7.0": {"l3"},
		}, &Desc{
			ValueType: prometheus.GaugeValue,
			Subsystem: "electricity",
			Name:      "phase_power_received_kw",
			Help:      "Instantaneous phase power received (-P) in 1 Watt resolution",
			Labels:    []string{"phase"},
		})
		c.createDesc("0-1:24.2.1", &Desc{
			ValueType: prometheus.CounterValue,
			Subsystem: "gas",
			Name:      "delivered_m3",
			Help:      "Last 5-minute value gas delivered to client in m3",
		})
	}

	for _, desc := range c.descs {
		ch <- desc.desc
	}

	ch <- c.descConnectionInfo
	ch <- c.descDeviceInfo
}

// Describe implements the prometheus.Collector interface.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.m.RLock()
	defer c.m.RUnlock()

	for id, desc := range c.descs {
		obj, ok := c.frame.Objects[id]
		if !ok {
			ch <- prometheus.NewInvalidMetric(desc.desc, errors.New("metric not found in latest dsmr frame"))
			continue
		}

		value, err := strconv.ParseFloat(obj.Value.Data, 64)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(desc.desc, fmt.Errorf("parse dsmr metric: %w", err))
			continue
		}

		metric := prometheus.MustNewConstMetric(desc.desc, prometheus.GaugeValue, value, desc.labelValues...)
		if !obj.Time.IsZero() {
			metric = prometheus.NewMetricWithTimestamp(obj.Time, metric)
		} else if !c.frame.Time.IsZero() {
			metric = prometheus.NewMetricWithTimestamp(c.frame.Time, metric)
		}

		ch <- metric
	}

	ch <- prometheus.MustNewConstMetric(c.descConnectionInfo, prometheus.GaugeValue, 1, c.frame.Header, c.frame.Version)
}

func (c *Collector) Run(ctx context.Context, onReady func() error) error {
	for {
		frame, err := c.r.Next()
		if err != nil {
			return err
		}

		c.m.Lock()
		ready := c.frame == nil
		c.frame = frame
		c.m.Unlock()

		if ready {
			if err := onReady(); err != nil {
				return err
			}
		}
	}
}

func (c *Collector) createDesc(objectID string, desc *Desc) {
	if _, ok := c.frame.Objects[objectID]; ok {
		desc.desc = prometheus.NewDesc(
			prometheus.BuildFQName("dsmr", desc.Subsystem, desc.Name),
			desc.Help,
			desc.Labels,
			nil,
		)
		c.descs[objectID] = desc
	}
}

func (c *Collector) createDescMulti(objectIDs map[string][]string, desc *Desc) {
	desc.desc = prometheus.NewDesc(
		prometheus.BuildFQName("dsmr", desc.Subsystem, desc.Name),
		desc.Help,
		desc.Labels,
		nil,
	)

	for objectID, values := range objectIDs {
		desc := *desc
		desc.labelValues = values
		c.descs[objectID] = &desc
	}
}
