#!/bin/bash
#
# This script is run on the parent EC2 instance.  It sets up tools that we need
# to facilitate communication with the Nitro enclave.

source config.sh

echo "[+] Killing existing VIProxy and SOCKS proxy instances."
killall viproxy socksproxy

echo "[+] Setting up VIProxy as AF_INET <-> AF_VSOCK forwarder."
export IN_ADDRS="0.0.0.0:${incoming_port},0.0.0.0:${incoming_acme_port},${host_cid}:${incoming_acme_port}"
export OUT_ADDRS="${enclave_cid}:${incoming_port},${enclave_cid}:${incoming_acme_port},localhost:${outgoing_port}"
viproxy &

echo "[+] Setting up SOCKSv5 proxy."
socksproxy -addr ":$outgoing_port" &
