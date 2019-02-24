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
	quitSignals = []os.Signal{
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
	configFile := flag.String("config", "", "path to a woodwatch JSON config file")
	verbose := flag.Bool("verbose", false, "verbose output and webhook dispatch")
	flag.Parse()

	logger := log.New(os.Stdout, "woodwatch ", log.LstdFlags)
	if *configFile == "" {
		logger.Fatal("you must specify a -config file")
	}

	// Load a Config instance from disk
	c, err := woodwatch.LoadConfigFile(*configFile)
	if err != nil {
		logger.Fatalf("error loading config %q: %v\n", *configFile, err)
	}

	// Create the woodwatch server
	server, err := woodwatch.NewServer(
		logger,
		*verbose,
		*listenAddress,
		c)
	if err != nil {
		logger.Fatalf("error creating server: %v\n", err)
	}

	// Listen for quitSignals. When one is received close the server.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, quitSignals...)
	go func() {
		<-sigChan
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
