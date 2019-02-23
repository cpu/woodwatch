# woodwatch

[![Build Status](https://travis-ci.com/cpu/woodwatch.svg?branch=master)](https://travis-ci.com/cpu/woodwatch)
[![Coverage Status](https://coveralls.io/repos/github/cpu/woodwatch/badge.svg)](https://coveralls.io/github/cpu/woodwatch)
[![Go Report Card](https://goreportcard.com/badge/github.com/cpu/woodwatch)](https://goreportcard.com/report/github.com/cpu/woodwatch)
[![GolangCI](https://golangci.com/badges/github.com/cpu/woodwatch.svg)](https://golangci.com/r/github.com/cpu/woodwatch)

`woodwatch` is a small Go program that can be used to POST webhooks when
peers stop sending ICMP echo requests (pings) for long enough to be considered
down.

Lots of systems have the ability to send pings and so they all work with
`woodwatch` out of the box! No client software needs to be installed.

Despite using ICMP `woodwatch` can be run without root privileges with Linux
capabilities. A small `bless.sh` script is included to give `cap_net_raw`
capabilities to the `woodwatch` binary.

The Internet is flaky. `woodwatch` tries to eliminate basic jitter and sporadic
packet loss by supporting configurable up/down thresholds. The thresholds can be
configured globally and also per-monitored-peer.

Notify all the things! `woodwatch`'s webbhooks work out of the box with Slack
for channel notifications when a peer goes up/down. Like the thresholds webhooks
can be configured globally and also per-monitored-peer.

# Installation

1. Pick a [woodwatch release](https://github.com/cpu/woodwatch/releases) and
   download the `.tar.gz` for your architecture (most probably
   `Linux_x86_64.tar.gz`).

       wget https://github.com/cpu/woodwatch/releases/download/v0.0.1/woowdwatch_v0.0.1_Linux_x86_64.tar.gz

1. Extract the release archive and `cd` into it.

       mkdir /tmp/woodwatch && tar xf woodwatch_*.tar.gz -C /tmp/woodwatch --strip-components=1 && cd /tmp/woodwatch

1. Put the `woodwatch` binary in `/usr/local/bin`.

       sudo cp woodwatch /usr/local/bin

1. Make the `bless.sh` script executable and use it on the installed `woodwatch`
   binary to give it `cap_net_raw`.

       chmod +x ./bless.sh && sudo ./bless.sh /usr/local/bin/woodwatch

1. Create the `woodwatch` config directory.

       sudo mkdir -p /etc/woodwatch

1. Copy the example config in place.

       sudo cp example.config.json /etc/woodwatch/config.json

1. Customize the example config.

       sudo $EDITOR /etc/woodwatch/config.json

1. Add a `woodwatch` user with no shell, a disabled password, and no home
   directory.

       sudo adduser --disabled-password --no-create-home --shell=/bin/false --gecos "" woodwatch

1. Give the `woodwatch` user ownership of the `woodwatch` config directory.

       sudo chown -R woodwatch:woodwatch /etc/woodwatch

1. Install the example systemd service.

       sudo cp example.woodwatch.service /etc/systemd/system/woodwatch.service

1. Reload the systemd manager configuration.

       sudo systemctl daemon-reload

1. Enable the `woodwatch` service to start at boot.

       sudo systemctl enable woodwatch

1. Start the `woodwatch` service.

       sudo systemctl start woodwatch

1. Check the `woodwatch` service logs to ensure there are no errors.

       journalctl -u woodwatch --no-pager -e

# Configuration

## Global Configuration

* `UpThreshold` - an unsigned integer expressing how many checks **without**
    a peer timeout must occur before the peer is considered up.
* `DownThreshold` - an unsigned integer expressing how many checks **with**
    a peer timeout must occur before the peer is considered down.
* `MonitorCycle` - a required duration string expressing how often peers are checked for
    timeouts. This should be shorter than the `PeerTimeout`.
* `PeerTimeout` - a required duration string expressing how long must elapse between
    seeing ICMP echo requests from a peer before it is considered timed out.
    This should be longer than the `MonitorCycle`.
* `Webhook` - an optional string specifying a URL to be POSTed for notable
    events (or all state change events if `-verbose` is used).
* `Peers` - one or more objects describing a peer configuration.

## Peer Configuration

* `Name` - a required string representing the name of the peer. Use Slack emoji like
    `:satellite:` to make your webhook events more memorable.
* `Network` - a required CIDR notation network that the peer will be sending ICMP echo
    requests from. E.g. `192.168.1.0/24` to expect pings from `192.168.1.1`
    through `192.168.1.254`. You may find [a CIDR
    calculator](http://www.subnet-calculator.com/cidr.php) helpful.
* `UpThreshold` - an optional unsigned integer to override the global
    `UpThreshold` for this peer.
* `DownThreshold` - an optional unsigned integer to override the global
    `DownThreshold` for this peer.
* `Webhook` - an optional string specifying a URL to override the global
    `Webhook` for this peer.

## Example Configuration

```
{
  "UpThreshold": 3,
  "DownThreshold": 3,
  "MonitorCycle": "2s",
  "PeerTimeout": "4s",
  "Webhook": "http://localhost:9090/woodwatch-hook"
  "Peers": [
    {
      "Name": "LAN",
      "Network": "192.168.1.0/24",
      "UpThreshold": 2,
      "DownThreshold": 5,
      "Webhook": "http://localhost:9090/custom-lan-hook"
    }
  ]
}
```

The above configuration will have `woodwatch` monitor a LAN for connectivity by
expecting periodic ICMP echo requests from any host in the `192.168.1.0/24`
network, at least every 4s.

After 2 checks (4s) having seen the expected pings the LAN network will be
considered Up and a webhook POST will be sent to `http://localhost:9090/custom-lan-hook`.

If the pings stop being sent, after 5 checks (10s) the LAN network will be
considered Down and a webhook POST will be sent to
`http://localhost:9090/custom-lan-hook`.

## Example Webhook POSTs

For the example configuration shared above the configured webhook for the LAN
peer will receive an Up event as a HTTP POST request like:

```
POST /custom-lan-hook HTTP/1.1
Host: localhost:9090
User-Agent: cpu.woodwatch 0.0.1 (linux; amd64)
Content-Length: 299
Content-Type: application/json
Accept-Encoding: gzip

{
  "title": "Peer LAN is Up",
  "text": "LAN (last seen 2019-02-24 11:22:44 AM -0500) was previously Maybe Up (2 of 2) and is now Up",
  "timestamp": "2019-02-24T11:22:45.045655028-05:00",
  "lastSeen": "2019-02-24T11:22:44.660459371-05:00",
  "newState": "Up",
  "prevState": "Maybe Up (2 of 2)"
}
```

Similarly for a Down event the configured webhook for the LAN
peer will receive an HTTP POST request like:

```
POST /custom-lan-hook HTTP/1.1
Host: localhost:9090
User-Agent: cpu.woodwatch 0.0.1 (linux; amd64)
Content-Length: 309
Content-Type: application/json
Accept-Encoding: gzip

{
  "title": "Peer LAN is Down",
  "text": "LAN (last seen 2019-02-24 11:22:54 AM -0500) was previously Maybe Down (5 of 5) and is now Down",
  "timestamp": "2019-02-24T11:23:09.045820003-05:00",
  "lastSeen": "2019-02-24T11:22:54.695991071-05:00",
  "newState": "Down",
  "prevState": "Maybe Down (5 of 5)"
}
```

# Development

`woodwatch` is built to support Go 1.11.x and uses
[Go 1.11 modules](https://github.com/golang/go/wiki/Modules) and [vendored
dependencies](https://github.com/golang/go/wiki/Modules#how-do-i-use-vendoring-with-modules-is-vendoring-going-away).
Presently the only dependency outside of the Go stdlib is
[`x/net/`](https://golang.org/x/net/). Releases are built and published with
[GoReleaser](https://goreleaser.com/).

Presently `woodwatch` supports Linux and the `x86_64`, `arm64`, `armv7` and
`arm6` architectures. Woodwatch does not support other OSes at this time
because:

1. The `x/net/icmp.ListenPacket` function can only bind a raw ICMP endpoint [on
Darwin and Linux](https://godoc.org/golang.org/x/net/icmp#ListenPacket).
1. Running `woodwatch` without root requires [Linux
capabilities](https://linux.die.net/man/7/capabilities), specifically
`CAP_NET_RAW`. It's **a very bad idea** to run `woodwatch` as root. Don't do it!

## Linting

To run linters locally install `golangci-lint`:

       go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

and then start the linters by running `golangci-lint` in the root of the project
directory:

       golangci-lint run

## Unit tests

In the root of the project directory run:

       go test -race ./...

## Echo Webhook POSTs

You might find it useful to test `woodwatch` webhooks by running a small
webserver that echoes all received POST requests. One option for this is the
[Node.js `http-echo-server`](https://www.npmjs.com/package/http-echo-server).
You can run the server as follows:

       PORT=9090 http-echo-server

Configure a webhook URL using the `http-echo-server` by setting the URL in your
config file:

       ...
       "UpHook": "http://localhost:9090/woodwatch-up-hook",
       ...

## Making a snapshot release

First make sure you have [installed
GoReleaser](https://goreleaser.com/install/#compiling-from-source). Then run:

       goreleaser --snapshot --skip-publish --rm-dist

This will result in a `dist/` directory structure similar to the following:

       dist
       ├── checksums.txt
       ├── config.yaml
       ├── linux_amd64
       │   └── woodwatch
       ├── linux_arm_6
       │   └── woodwatch
       ├── linux_arm64
       │   └── woodwatch
       ├── linux_arm_7
       │   └── woodwatch
       ├── woodwatch_v0.0.0-next_Linux_arm64.tar.gz
       ├── woodwatch_v0.0.0-next_Linux_armv6.tar.gz
       ├── woodwatch_v0.0.0-next_Linux_armv7.tar.gz
       └── woodwatch_v0.0.0-next_Linux_x86_64.tar.gz
