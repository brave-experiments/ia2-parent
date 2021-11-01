#!/bin/bash
#
# This script is run on the parent EC2 instance.  It sets up TCP forwarders
# that facilitate communication with the Nitro enclave.

source config.sh

echo "[+] Killing existing socat instances."
killall socat

echo "[+] Setting up vsock/inet forwarders for CID ${enclave_cid}."
# Note that we need at least socat in version 1.7.4 because that's where VSOCK
# support was introduced.
socat "TCP4-LISTEN:${incoming_port},nodelay,fork"      "VSOCK-CONNECT:${enclave_cid}:${incoming_port}" &
socat "TCP4-LISTEN:${incoming_acme_port},nodelay,fork" "VSOCK-CONNECT:${}:${incoming_acme_port}" &
socat "VSOCK-LISTEN:${outgoing_port},fork"             "TCP:localhost:${outgoing_port},nodelay" &
