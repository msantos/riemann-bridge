# SYNOPSIS

riemann-bridge [*options*] *url*

# DESCRIPTION

Pipeline for [riemann](https://riemann.io/) events:

* read events from a [websocket](https://github.com/gorilla/websocket),
  [SSE](https://github.com/donovanhide/eventsource) or stdin
* write events to a websocket or stdout
* forward events between riemann instances

Some possible uses are:

* sending or querying riemann events

* stream live events to a test server

* partition events to another riemann server for a restricted view

* failover or load balancing riemann instances

## Stdin

Reads JSON from standard input. If a `time` field does not exist, the
field is added with the value set to the current time.

# EXAMPLES

## Sending Events

~~~
echo '{"service": "foo", "metric": 2}' | \
 riemann-bridge --src=- ws://127.0.0.1:5556/events
~~~

## Querying Events

~~~
# websocket
riemann-bridge --src=ws://127.0.0.1:5556/index --query='service = "foo"' -

# SSE
riemann-bridge --src=http://127.0.0.1:8080/index --query='service = "foo"' -
~~~

## Forwarding Events Between Riemann Instances

~~~
riemann-bridge \
 --src=ws://127.0.0.1:5556/index \
 --query='service = "test" and not state = "expired"' \
 ws://127.0.0.1:6556/events
~~~

# OPTIONS

--src *string*
: Source riemann server (default: ws://127.0.0.1:5557/index)

--query *string*
: Riemann query (default: not (service ~= "^riemann" or state = "expired"))

--number *int*
: Send *number* events and exit

--buffer-size *uint*
: Drop any events exceeding the buffer size (defaut: 0 (unbuffered))

--verbose *int*
: Debug messages

# ENVIRONMENT VARIABLES

RIEMANN_BRIDGE_SRC
: default source riemann server

RIEMANN_BRIDGE_DST
: default destination riemann server

RIEMANN_BRIDGE_QUERY
: default riemann query

# BUILD

    go get -u github.com/msantos/riemann-bridge
