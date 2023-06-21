package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/alexbakker/go-dsmr/dsmr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.bug.st/serial"
)

var (
	flagHttpAddr       = flag.String("http-addr", ":9111", "the addr to have the http server listen on")
	flagDevice         = flag.String("device", "", "the name of the serial device")
	flagDeviceBaudRate = flag.Int("device-baud-rate", 115200, "the baud rate to use for the serial device")

	ctx, cancel = context.WithCancel(context.Background())
)

func main() {
	flag.Parse()

	if flagDevice == nil {
		exitWithError("Specify a -device to read from")
	}

	port, err := serial.Open(*flagDevice, &serial.Mode{
		BaudRate: *flagDeviceBaudRate,
	})
	if err != nil {
		exitWithError(fmt.Sprintf("Failed to open serial port: %s", err))
	}
	defer func() {
		if err := port.Close(); err != nil {
			exitWithError(fmt.Sprintf("Failed to close serial port: %s", err))
		}
	}()

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(collectors.NewGoCollector())

	r := dsmr.NewReader(port)
	collector := NewCollector(r, *flagDevice)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := collector.Run(ctx, func() error {
			return reg.Register(collector)
		}); err != nil {
			exitWithError(fmt.Sprintf("Failed to run collector: %s", err))
		}
	}()

	httpSrv := http.Server{
		Addr:    *flagHttpAddr,
		Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpSrv.ListenAndServe(); err != nil {
			exitWithError(fmt.Sprintf("Failed to serve http: %s", err))
		}
	}()

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, os.Interrupt)
	<-exitChan

	fmt.Fprintf(os.Stderr, "Shutting down")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := httpSrv.Shutdown(ctx2); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to shut down http server: %s", err)
	}

	cancel()
	wg.Wait()
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
