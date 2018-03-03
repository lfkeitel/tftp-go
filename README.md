# TFTP Go

A simple TFTP server and client utility implemented in Go.

Build using `go build` or `go install`.

## Running a Server

`tftp [flags] [get|put] [REMOTE PATH] [LOCAL PATH]`

- `-server` - Start a TFTP server.
- `-root` - The root directory to serve. Defaults to the current working directory.
- `-nocreate` - Disable creation of non-existent files.
- `-nowrite` - Disable all writes, makes the server read-only.
- `-ow` - Allow overwriting existing files. Cannot be used with `-nowrite`. (see notes below)
- `-debug` - Output debug data.
- `-rfc1350` - Disable TFTP option extensions, works for both client and server usage.
- `-strict` - Reject clients trying to use netascii or mail transfer modes.

The server must be ran with enough privileges to listen on TFTP port 69/udp.

Remote and local path are only used if executed without the "-server" flag.

## Examples

### Client

`tftp get tftp.example.com:hello.txt hello.txt` - Get a file from tftp.example.com named hello.txt and save it to hello.txt in the current directory.

`tftp put tftp.example.com:hello2.txt hello.txt` - Send hello.txt in the current directory to tftp.example.com as filename hello2.txt.

### Server

`tftp -server` - Start a server using the current directory as the root directory.

`tftp -server -nowrite` - Start a read-only server using the current directory as the root directory.

`tftp -server -ow` - Start a server using the current directory as the root directory and allow files to be overwritten.

## Implemented RFCs

- [RFC 1350](https://tools.ietf.org/html/rfc1350) Base TFTP protocol
- [RFC 2347](https://tools.ietf.org/html/rfc2347) Options format and negotiation
- [RFC 2348](https://tools.ietf.org/html/rfc2348) Blocksize option
- [RFC 2349](https://tools.ietf.org/html/rfc2349) Timeout and transfer size options

## RFC Deviations

- ***Overwriting Files*** - Overwriting existing files is supported if the "-ow" flag is used. When a client request to write
to an existing file, the file will be truncated and the transfered file will be written in place.
Nothing is done to mitigate failed or corrupted transfers. The only mention of overwriting files
is error code 6 for "File already exists". Since this could be a useful feature, it's been
implemented but placed behind a flag. Use with caution.
- ***Transfer Modes*** - This implementation only supports the `octet` transfer mode. `mail` and `netascii` modes are ignored.
If a client tries to use either of those modes, the server will accept them but send the data as if octet mode was requested.
In practice, this shouldn't cause issues as TFTP is mainly used to transfer bootstrapping programs or firmware both of which are
transferred using octet mode. The client as no way to specify transfer mode. Use the `-strict` flag to reject clients that don't
use octet mode.

## TODOs

- Implement windowsize option from [RFC 7440](https://tools.ietf.org/html/rfc7440).
