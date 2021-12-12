# Plex the net.Conn Multiplexer

[![Build & Test](https://github.com/devnw/plex/actions/workflows/build.yml/badge.svg)](https://github.com/devnw/plex/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/go.atomizer.io/plex)](https://goreportcard.com/report/go.atomizer.io/plex)
[![codecov](https://codecov.io/gh/devnw/plex/branch/main/graph/badge.svg)](https://codecov.io/gh/devnw/plex)
[![Go Reference](https://pkg.go.dev/badge/go.atomizer.io/plex.svg)](https://pkg.go.dev/go.atomizer.io/plex)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

Plex is a library for managing net.Conn connection types with minimal effort
this library utilizes the [options](#options) pattern to allow for easy
configuration and customization of the multiplexer. This library provides
multiple types which are used as wrappers to the configured net.Conn types
which allow for general streaming of data through channels of bytes as an
alternative to the more go traditional io.Reader/io.Writer interfaces.

One important caveat is that the Plex library makes no assumptions of which
net.Conn object is returned from either Reader or Writer. This means that
messages must be routed by the consumer based on the data in the message
rather than assuming that a response to a received message will be sent to
the same connection which initiated the message round trip transmission.

The goal of this library is to abstract the complexity of managing multiple
connections and their associated channels of data into a single API which
consumers can request Reader and Writer types from. These types implement
the io.Closer method which will re-queue the Reader or Writer to the correct
internal buffer within the multiplexer for another consumer down the line.

## Installation

```bash
go get -u go.atomizer.io/plex@latest
```

## Usage

Plex uses the options pattern to allow for users to configure the multiplexer.
For a full list of options see the [options](#options) documentation.

```go
import "go.atomizer.io/plex"

...

m, err := plex.New(
    ctx, 
    plex.WithConnections(
        mynetconn1,
        mynetconn2,
        mynetconnN,
    ),
)

// Access a plex.Reader
r, err := m.Reader(ctx, timeout)

// Access a plex.Writer
w, err := m.Writer(ctx, timeout)

```

Users are responsible for closing any plex.Readers/plex.Writers they acquire
from the multiplexer. If they are not closed they will not be returned to the
multiplexer pool for reuse.

For further options and documentation visit our documentation page using the
docs badge at the top of the README.

## Options

Availble configuration options include the following:

NOTE: For the full documentation see [Options](https://pkg.go.dev/go.atomizer.io/plex#Option)

### `WithConnections(connections ...net.Conn)`

Adds a pre-configured list of net.Conn objects to the multiplexer. This option
also sets the internal buffer size to the number of the Multiplexer when NOT
used in conjunction with the `WithMaxCapacity` option.

NOTE: Buffer size of the multiplexer does NOT change for the lifetime of the
multiplexer.

### `WithMaxCapacity(max int)`

Sets a pre-configured maximum capacity for the multiplexer. This max capacity
will not change for the lifetime of the multiplexer.

### `WithConnector(c Connect)`

Allows for the configuration of a custom Connect function. This function has
the following signature: `func(ctx context.Context, conn net.Conn) error`. The
configuration of the Connector allows the Multiplexer to re-create failed
connections that already exist or to create new connections when using the
`WithAutoScaling` option.

### `WithAutoScaling(timeout time.Duration)`

Configures an internal timeout within the multiplexer to automatically add new
connections to the multiplexer in the event that there are not any available
connections in the multiplexer. This option is only available when using the
`WithConnector` option.
