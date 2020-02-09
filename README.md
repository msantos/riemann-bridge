# SYNOPSIS

riemann-bridge [*options*] *url*

# DESCRIPTION

A command line, pipeline for [riemann](https://riemann.io/) events:

* read events from a [websocket](https://github.com/gorilla/websocket)
  or stdin
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
riemann-bridge --src=ws://127.0.0.1:5556/index --query='service = "foo"' -
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

# TODO

* optional rate limiting

* flow control: if buffered events exceed a user specified limit, drop
  new events
