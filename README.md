# TFTP Go

A simple TFTP server implemented in Go.

Build using `go build` or `go install`.

## Running a Server

`tftp-go [flags]`

- `-rootdir` - The root directory to serve. Defaults to the current working directory.
- `-create` - Allow creation of non-existant files.
- `-nowrite` - Disable all writes, makes the server read-only.

The server must be ran with enough privileges to listen on TFTP port 69/udp.

## RFCs

- [RFC 1350](https://tools.ietf.org/html/rfc1350)
- [RFC 2348](https://tools.ietf.org/html/rfc2348)
- [RFC 2349](https://tools.ietf.org/html/rfc2349)

## TODOs

- Implement options as described in RFCs 2347 (base) and 7440 (windowsize).
