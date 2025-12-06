[![Go Reference](https://pkg.go.dev/badge/go.iscode.ca/riemann-bridge.svg)](https://pkg.go.dev/go.iscode.ca/riemann-bridge)

# SYNOPSIS

riemann-bridge [*options*] *url*

# DESCRIPTION

Pipeline for [riemann](https://riemann.io/) events:

* read events from a [websocket](https://github.com/gorilla/websocket),
  [SSE](https://github.com/donovanhide/eventsource) or stdin
* write events to a websocket or stdout
* forward events between riemann instances

Possible uses are:

* publishing or querying riemann events

* replicate events to a test server

* partition events to another riemann server for a restricted view

* failover or load balancing riemann instances

## Stdin

Reads JSON from standard input. If a `time` field does not exist, the
field is added with the value set to the current time.

## Websocket

### Reading Events

Use `/index`. For example, if the riemann server is running on port 5556
on localhost:

```
ws://127.0.0.1:5556/index
```

### Writing Events

Use `/events`. For example, if the riemann server is running on port 5556
on localhost:

```
ws://127.0.0.1:5556/events
```

## SSE

Use `/index`. For example, if the riemann server is running on port 5558
on localhost:

```
http://127.0.0.1:5558/index
```

If the riemann service is proxied by path, adjust the URL:

```
https://example.com/event/index
```

### Reading Events

# EXAMPLES

## Sending Events

```
echo '{"service": "foo", "metric": 2}' | \
 riemann-bridge - ws://127.0.0.1:5556/events
```

## Querying Events

```
# websocket
riemann-bridge --query='service = "foo"' ws://127.0.0.1:5556/index

# SSE
riemann-bridge --query='service = "foo"' http://127.0.0.1:8080/index
```

## Forwarding Events Between Riemann Instances

```
riemann-bridge \
 --query='service = "test" and not state = "expired"' \
 ws://127.0.0.1:5556/index \
 ws://127.0.0.1:6556/events
```

# ARGS

src *string*
: Source riemann server (default: -)

Examples:

```
  ws://127.0.0.1:5556/index
  http://127.0.0.1:5558/index
```

destination
: Destination riemann server (default: -)

Examples:

```
  ws://127.0.0.1:5556/events
```

# OPTIONS

query *string*
: Riemann query (default: `not (service ~= "^riemann" or state = "expired")`)

number *int*
: Send *number* events and exit

buffer-size *uint*
: Drop any events exceeding the buffer size (defaut: `0` (unbuffered))

verbose *int*
: Debug messages

# ENVIRONMENT VARIABLES

RIEMANN_BRIDGE_SRC
: default source riemann server

RIEMANN_BRIDGE_DST
: default destination riemann server

RIEMANN_BRIDGE_QUERY
: default riemann query

# BUILD

```
go install go.iscode.ca/riemann-bridge/cmd/riemann-bridge@latest
```

To build a reproducible executable from the git repository:

```
CGO_ENABLED=0 go build -trimpath -ldflags "-w" ./cmd/riemann-bridge
```
