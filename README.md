# TFTP Go

A simple TFTP server implemented in Go.

Build using `go build` or `go install`.

Run with `tftp-go ./server_root`. The only argument is the root directory the server will serve.
The server must be ran with enough privileges to listen on TFTP port 69/udp.

Implements the following RFCs:

- [RFC 1350](https://tools.ietf.org/html/rfc1350)

TODOs:

- Implement options as described in RFCs 2347 (base), 2348 (block size), 2349 (timeout, transfer size),
and 7440 (windowsize).