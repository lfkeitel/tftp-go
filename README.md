# TFTP Go

A simple TFTP server and client utility implemented in Go.

Build using `go build` or `go install`.

## Running a Server

`tftp [flags] [remote path] [local path]`

- `-server` - Start a TFTP server.
- `-rootdir` - The root directory to serve. Defaults to the current working directory.
- `-nocreate` - Disable creation of non-existent files.
- `-nowrite` - Disable all writes, makes the server read-only.
- `-ow` - Allow overwriting existing files. Cannot be used with `-nowrite`. (see notes below)

The server must be ran with enough privileges to listen on TFTP port 69/udp.

Remote and local path are only used if executed without the "-server" flag.

## Implemented RFCs

- [RFC 1350](https://tools.ietf.org/html/rfc1350)
- [RFC 2347](https://tools.ietf.org/html/rfc2347)
- [RFC 2348](https://tools.ietf.org/html/rfc2348)
- [RFC 2349](https://tools.ietf.org/html/rfc2349)

## Notes

- Overwriting existing files is supported if the "-ow" flag is used. When a client request to write
to an existing file, the file will be truncated and the transfered file will be written in place.
Nothing is done to mitigate failed or corrupted transfers. The only mention of overwriting files
is error code 6 for "File already exists". Since this could be a useful feature, it's been
implemented but placed behind a flag. Use with caution.

## TODOs

- Implement windowsize option from [RFC 7440](https://tools.ietf.org/html/rfc7440).
