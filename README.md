# chiotel
[![Tests](https://github.com/rmcdgl/chiotel/actions/workflows/Tests.yml/badge.svg)](https://github.com/rmcdgl/chiotel/actions/workflows/Tests.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/rmcdgl/chiotel.svg)](https://pkg.go.dev/github.com/rmcdgl/chiotel)

[OpenTelemetry](https://opentelemetry.io/) instrumentation for
[go-chi/chi](https://github.com/go-chi/chi).

## Why?

This package takes a simpler approach than
[riandyrn/otelchi](https://github.com/riandyrn/otelchi) by not matching the
route twice and by using middleware that's part of the `chi/middleware` package.
