# sensu-http-perf-go

## Table of Contents

- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)

## Overview

The sensu-http-perf-go is a [Sensu Check][6] that measures the performance of HTTP requests. And was inspired by the ruby based http-perf check. However that check did not support chanign the TLS timeout, which was a requirement for my use case. So I decided to write my own check in go. As in the ruby version, this check will measure the following metrics: dns_duration, tls_handshake_duration, connect_duration, first_byte_duration, total_request_duration. And it outputs metrics in nagios_perfdata format.

## Files

- `bin/sensu-http-perf-go`

## Usage examples

```bash
sensu-http-perf-go -u https://example.com
sensu-http-perf-go OK: 0.790421s | dns_duration=0.047340, tls_handshake_duration=0.089218, connect_duration=0.049823, first_byte_duration=0.601708, total_request_duration=0.790421

```

help:

```bash
sensu-http-perf-go -h
Alternate version of http-perf

Usage:
  sensu-http-perf-go [flags]
  sensu-http-perf-go [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -c, --critical float32       Critical threshold, in seconds (default 2)
  -h, --help                   help for sensu-http-perf-go
  -i, --insecure-skip-verify   Skip TLS certificate verification (not recommended!)
  -m, --output-in-ms           Provide output in milliseconds (default false, display in seconds)
  -T, --timeout int            Request timeout in seconds (default 15)
  -z, --tls-timeout int        TLS handshake timeout in milliseconds (default 1000)
  -u, --url string             URL to test (default http://localhost:80/) (default "http://localhost:80/")
  -w, --warning float32        Warning threshold, in seconds (default 1)
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```bash
sensuctl asset add DoctorOgg/sensu-http-perf-go
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/DoctorOgg/sensu-http-perf-go].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-http-perf-go
  namespace: default
spec:
  command: sensu-http-perf-go --url https://example.com
  subscriptions:
  - system
  runtime_assets:
  - DoctorOgg/sensu-http-perf-go
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-http-perf-go repository:

```bash
go build
```

[6]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
