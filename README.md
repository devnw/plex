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

Utilizing runtime contention this implementation is able to handle a large
number of streams, and is able to handle a large number of readers and writers.

## Installation

To import:

```bash
go get -u go.atomizer.io/plex@latest
```
