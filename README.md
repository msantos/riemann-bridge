# SYNOPSIS

riemann-bridge [*options*] *query*

# DESCRIPTION

Forward events between riemann instances using
[websockets](https://github.com/gorilla/websocket).

Could be used for:

* testing configuration changes by streaming live events to a test
  server

* partitioning or providing a restricted view of events

* failover or load balancing riemann instances

# EXAMPLES

~~~
riemann-bridge \
 --src=ws://127.0.0.1:5556/index \
 --dst=ws://127.0.0.1:6556/events \
 'service = "test" and not state = "expired"'
~~~

# OPTIONS

--src *string*
: Source riemann server (default: ws://127.0.0.1:5557/index)

--dst *string*
: Destination riemann server (default: ws://127.0.0.1:6557/events)

--verbose *int*
: Debug messages

# ENVIRONMENT VARIABLES

RIEMANN_BRIDGE_SRC
: default source riemann server

RIEMANN_BRIDGE_DST
: default destination riemann server

# BUILD

    go get -u github.com/msantos/riemann-bridge

# TODO

* optional rate limiting

* flow control: if buffered events exceed a user specified limit, drop
  new events
