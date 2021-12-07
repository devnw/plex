# Plex the I/O Multiplexer

[![Build & Test](https://github.com/devnw/plex/actions/workflows/build.yml/badge.svg)](https://github.com/devnw/plex/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/go.atomizer.io/plex)](https://goreportcard.com/report/go.atomizer.io/plex)
[![codecov](https://codecov.io/gh/devnw/plex/branch/main/graph/badge.svg)](https://codecov.io/gh/devnw/plex)
[![Go Reference](https://pkg.go.dev/badge/go.atomizer.io/plex.svg)](https://pkg.go.dev/go.atomizer.io/plex)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

Plex manages multiple I/O streams and provides a unified interface for accessing
readers and writers. Consumers can request a stream from the multiplexer, and
the multiplexer will return any available stream (if there is one) or
block/timeout until one becomes available.

Utilizing runtime contention Plex is able to handle a large number of streams
for both reading and writing to underlying io.Readers/io.Writers and
io.ReadWriters.

## Installation

```bash
go get -u go.atomizer.io/plex@latest
```

## Usage

Plex uses the options pattern to allow for users to configure the multiplexer.
For a full list of options see the [Options](https://pkg.go.dev/go.atomizer.io/plex#Option)
documentation.

```go
import "go.atomizer.io/plex"

...

m, err := plex.New(
    ctx, 
    plex.WithReader(myreaders...), 
    plex.WithWriter(mywriters...),
)

// Access a ReadStream / io.ReadCloser
r, err := m.Reader(ctx, timeout)

// Access a WriteStream / io.WriteCloser
w, err := m.Writer(ctx, timeout)

```

Users are responsible for closing any streams they acquire from the multiplexer.
If a stream is not closed the stream will not be returned to the multiplexer
pool for reuse.

For further options and documentation visit our documentation page using the
docs badge at the top of the README.
