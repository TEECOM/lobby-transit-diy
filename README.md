# Lobby Transit DIY Server

## Background
This project contains an implementation of a [Lobby Transit](https://teecom.com/appsforbuildings/lobbytransit/)
data source server, written in Go. The server is intended to be used with future
versions of Lobby Transit, allowing full extensibility for transit systems that
are not included by default.

The server will need to be run from your own machine. You will provide a configuration
file based on `example-config.json` that contains the relevant information for the
new system in question.

Details about connecting to the Lobby Transit app are forthcoming.

## Compiling
In the project directory, simply run `go build ltdiy.go`. Once that has compiled,
simply execute the binary `ltdiy` with the config file: `./ltdiy -config=example-config.json`.
The server defaults to port 8080.

For faster development, simply run from the project directory: `go run ltdiy.go -config=example-config.json`

## Licensing
This software is released under the MIT license and is available "as is." Please
see `LICENSE.md` for the full license and disclosure.
