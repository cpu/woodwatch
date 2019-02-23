#!/usr/bin/env bash
#
# Usage:
#  - `./bless.sh`          - assumes binary is $GOPATH/bin/woodwatch
#  - './bless.sh <binary>` - uses provided arg as binary path.
#
# `blessh.sh` gives the woodwatch binary the cap_net_raw+ep capability
# required to listen for ICMP messages on an interface without being root.
#
set -e

DEFAULT_BIN_PATH="${GOPATH:-"$HOME/go"}/bin/woodwatch"

BIN_PATH="${1:-$DEFAULT_BIN_PATH}"

if ! [ -f "$BIN_PATH" ]; then
  echo "$BIN_PATH does not exist."
  exit 1
fi

if [ $EUID != 0 ]; then
  echo "Please run $0 as root."
  exit 1
fi

if ! [ -x "$(command -v setcap)" ]; then
  echo "setcap is not in your \$PATH or is not executable."
  exit 1
fi

if ! [ -x "$(command -v getcap)" ]; then
  echo "getcap is not in your \$PATH or is not executable."
  exit 1
fi

setcap cap_net_raw+ep "$BIN_PATH"
getcap "$BIN_PATH"
