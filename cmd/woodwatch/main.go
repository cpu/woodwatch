// Package main provides the `woodwatch` binary.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cpu/woodwatch"
)

var (
	// quitSignals are the signals that can be used to tell the woodwatch binary to
	// shut down cleanly.
	quitSignals []os.Signal = []os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}
	// listenAddress is the command line flag for the server listen address.
	listenAddress = flag.String(
		"listen",
		"0.0.0.0",
		"Interface address to listen to for IPv4 ICMP messages")
)

// main runs the woodwatch program.
func main() {
	logger := log.New(os.Stdout, "woodwatch ", log.LstdFlags)

	// Configure sources
	// TODO(@cpu): Read sources from config
	var sources []*woodwatch.Source

	mustSource := func(name string, network string) *woodwatch.Source {
		source, err := woodwatch.NewSource(name, network)
		if err != nil {
			logger.Fatalf("error creating source %q with network %q: %v\n",
				name, network, err)
		}
		return source
	}

	sources = append(sources, mustSource("Groupe Acces", "24.226.129.0/24"))
	sources = append(sources, mustSource("Explornet", "208.114.129.0/24"))
	sources = append(sources, mustSource("WoodWeb", "192.168.2.0/24"))

	// Create a server
	server, err := woodwatch.NewServer(logger, *listenAddress, sources)
	if err != nil {
		logger.Fatalf("error creating server: %v\n", err)
	}
	logger.Println("starting")

	// Listen for the quitSignals. When one is received close the server.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, quitSignals...)
	go func() {
		_ = <-sigChan
		logger.Println("ending")
		if err := server.Close(); err != nil {
			logger.Fatalf("err closing: %v\n", err)
		}
	}()

	// Start listening for packets to the server. This will block until
	// server.Close() is called by the signal handler above.
	if err := server.Listen(); err != nil {
		logger.Fatalf("error: %v\n", err)
	}
}
