# go-dsmr [![build](https://github.com/alexbakker/go-dsmr/actions/workflows/build.yml/badge.svg)](https://github.com/alexbakker/go-dsmr/actions/workflows/build.yml)

__go-dsmr__ is a Go package for reading
[DSMR](https://www.netbeheernederland.nl/_upload/Files/Slimme_meter_15_a727fce1f1.pdf)
telegrams of Dutch smart meters. There's also a [Prometheus
exporter](cmd/dsmr-exporter) that reads from the serial P1 port and exports the
data as metrics.

It currently supports a limited set of metrics exposed by the Kaifa MA304.
