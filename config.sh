# The externally-visible TCP port on the EC2 instance.  Clients will talk to
# this port via a TCP proxy.
incoming_port=8080

# The externally-visible TCP port on the EC2 instance that we use to get
# certificates via ACME.
incoming_acme_port=80

# The host to which the enclave will send processed data.
outgoing_destination=TODO

# The port of the host to which the enclave will send processed data.
outgoing_port=1080

# The enclave's CID, which is effectively an IP address in AF_VSOCK.
enclave_cid=4

# The CID of the parent (from the enclave's point of view) is always 3.
host_cid=3
