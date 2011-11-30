# noeqd - A fault-tolerant network service for meaningful GUID generation

Based on [snowflake][].

## Motivation

GUIDs (Globally Unique IDs) are useful for a number a obvious reasons:
database keys, logging, etc.

Generating GUIDs with pure randomness is not always ideal because it doesn't
cluster well, produces terrible locality, and no insight as to when it was
generated.

This network service should also have these properties (Differences from [snowflake][]):

* easy distribution with *no dependencies* and little to *no setup*
* dirt simple wire-protocol (trivial to implement clients without added dependencies and complexity)
* low memory footprint (starts and stays around ~1MB)
* zero configuration
* reduced network IO when multiple keys are needed at once

## Glossary of terms to follow

* `GUID`: Globally Unique Identifier
* `datacenter`: A facility used to house computer systems.
* `worker`: A single `noeq` process with a worker and datacenter ID combination unique to their cohort.
* `datacenter-id`: An integer representing a particular datacenter.
* `worker-id`: An integer representing a particular worker.
* `machine-id`: The comination of `datacenter-id` and `worker-id`
* `twepoch`: custom epoch (same as [snowflake][])

## Important note:

Reliability, and guarantees depend on:

**System clock depedency and skew protection:** - (From [snowflake][] README and slightly modified)

You should use NTP to keep your system clock accurate. Noeq protects from
non-monotonic clocks, i.e. clocks that run backwards. If your clock is running
fast and NTP tells it to repeat a few milliseconds, Noeq will refuse to
generate ids until a time that is after the last time we generated an id. Even
better, run in a mode where ntp won't move the clock backwards. See
<http://wiki.dovecot.org/TimeMovedBackwards#Time_synchronization> for tips on how
to do this.

**Avoiding the reuse of a worker-id + datacenter-id too quickly**

It's important to know that a newly born process has no way of tracking its
previous life and where it left of. This means time could have moved
backwards while it was dead.

It's important to **not** use the same worker-id + datacenter-id without
telling the new process when to start generating new IDs to avoid duplicates.

It is only safe to reuse the same worker-id + datacenter-id when you can
guarantee the current time is greater than the time of death. You can use the
`-t` option to specifiy this.

You may have up to 1024 machine ids. It's generally safe to not reuse them
until you've reached this limit.

## Install

You can install noeqd by downloading the binary
[here](http://github.com/bmizerany/noeq/downloads) and putting it in your
`PATH`.

*or*

Clone the repo and build with [Go](http://golang.org/doc/install.html) (Requires Go `b4a91b693374 weekly/weekly.2011-11-18` or later)

		$ git clone http://github.com/bmizerany/noeq
		$ cd noeq
		$ make install

## Run

		$ noeq -h
		Usage of noeq:
		  -d=0: datacenter id
		  -l="0.0.0.0:4444": the address to listen on
		  -w=0: worker id

**Coordinating machine-ids**

Noeq does not assume you're using any automated coordination because it isn't
always correct to assume this. Its easy to do without baking it in. Here is an
example script in the repo for doing so if you need it (using [Doozer][]):

		#!/bin/sh
		# usage: ./coord-exec.sh <datacenter-id>

		did=$1
		wid=0

		[ -z "$did" ] && did=0

		_set() {
		  printf 1 | doozer set /goflake/$did/$wid 0
		}

		while ! _set
		do wid=`expr $wid + 1`
		done

		exec noeq -w $wid -d $did

## The Why

**Uniqness**

We must know that a GUID, once generated, has never and will never be generated
again (i.e. Globally Unique) in our system.

**Performance**

Heroku serves many 10's of thousands of requests a second. Each request can
require multiple actions that need their own id. To be on the safe side, we
will require a minimum of 100k ids/sec (without network latency); possibly more
in the very near future. See benchmarks near the end of this README.

**Uncoordinated**

We need all `noeq`s to be able to generate GUIDs without coordinating with
other `noeq` processes. Coordination requires more time complexity than if
we didn't require it and reduces the amount of GUIDs we can generate during
that time. It also affects the yield (the probability the service will complete
a request).

**Direcly sortable by time (roughly)**

Noeq (like [snowflake][]) will guarantee the GUIDs will be k-sorted within
a reasonable bound (10's of ms to no more than 1s). More on this in "How it works."

References:

<http://portal.acm.org/citation.cfm?id=70413.70419>

<http://portal.acm.org/citation.cfm?id=110778.110783>

# The "Why not snowflake?"

At Heroku, we value services that are simple, as self-contained as possible,
and use nothing more than they can reasonably get away with. The setup and
distribution of an application should be as quick and painless as possible.
This means ruthlessly eliminating as much baggage, waste, and other overhead as
possible.

# How it works

## GUID generation and guarantees

GUIDs are represented as 64bit integers and are composed of (as described by the [snowflake][] README):

* time - 41 bits (millisecond precision with a custom epoch gives us 69 years)
* configured machine id - 10 bits - gives us up to 1024 machines
* sequence number - 12 bits - rolls over every 4096 per machine (with protection to avoid rollover in the same ms)

## Sorting - Time Ordered

*Strictly sorted*:

* GUIDs generated in a single request by a worker will strictly sort.
* GUIDs generated one second or longer apart, by more than one worker, will strictly sort.
* GUIDs generated over multiple requests by the same worker, will strictly sort.

*Roughly sorted*:

* GUIDs generated by multiple workers within a second could roughly sort.

An example of roughly sorted:

If client A requests three GUIDs from worker A in one request, and client B
requests three GUIDs from worker B in another request, and both requests are
processed within the same second, together they may sort like:

		GUID-A1
		GUID-A2
		GUID-B1
		GUID-B2
		GUID-A3
		GUID-B3

NOTE: The A GUIDs will strictly sort, as will B's.

## Clients

Clients implement a simple wire-protocol that is specified below. Implementing
a client in your favorite language is trivial and should require no
dependencies.

**Failure Recovery**

Each client should keep a list of addresses of all known worker process (or
use DNS) so that if one fails, it can move to another. To recover
from a lost connection, a client should randomly select another address from
its list, or in the case of DNS: reconnect using the same address allowing DNS
to choose the next IP.

See [noeq.go](http://github.com/bmizerany/noeq.go) for a working
example.

## Protocol

*Request*:

		-------
		|uint8|
		-------

A request must contain only one byte. The value of the byte tells the
server how many ids to respond with. A client can request up to 255 (or max uint8)
ids per request.

*Response*:

		-------------------------------------------------- ...
		|uint8|uint8|uint8|uint8|uint8|uint8|uint8|uint8|  ...
		-------------------------------------------------- ...

Each id comes as a 64bit integer in network byte order. The
number of 64bit integers returned is the `request-byte * 8`

*Errors*:

Errors are logged by the server to stdout. Clients will have their
connection closed to signal the need to try elsewhere until the server can
recover. This generally happens if the servers clock is running backwards.

## Benchmarks (fwiw)

**MacAir 3, OS X 10.7.2, 2.13 GHz Intel Core 2 Duo, 4 GB 1067 MHz DDR3)**


**Id Generation *without encoding* or network latency**

This is the benchmark done by [snowflake] and reported in their README.

		BenchmarkIdGeneration	675 ns/op	# 1.481 million ids/s
		
**Id Generation *with* encoding and *without* network latency**

I find these benchmarks more realistic. The ids must be encoded so we want to
know how fast an id can be generated and encoded in order to hit the wire.
Benchmarks including a network are left as an exercise for the reader because
all networks vary.

These show that when a client can safely ask for more one than one id at a time,
they can reduce time to wire and the expensive read/write operations.

		BenchmarkServe01	 1677 ns/op	# 596303 ids/sec
		BenchmarkServe02	 2352 ns/op	# 850340 ids/sec
		BenchmarkServe03	 3067 ns/op	# 978155 ids/sec
		BenchmarkServe05	 4436 ns/op	# 1.127 million ids/sec
		BenchmarkServe08	 6436 ns/op	# 1.243 million ids/sec
		BenchmarkServe13	10169 ns/op	# 1.278 million ids/sec
		BenchmarkServe21	16257 ns/op	# 1.292 million ids/sec
		BenchmarkServe34	25603 ns/op	# 1.328 million ids/sec
		BenchmarkServe55	39693 ns/op	# 1.386 million ids/sec

## Contributing

This is Github. You know the drill. Please make sure you keep your changes in a
branch other than `master` and in nice, clean, atomic commits. If you modify a
`.go` file, please use `gofmt` with no parameters to format it; then hit the
pull-request button.

## Issues

These are tracked in this repos Github [issues tracker](http://github.com/bmizerany/noeq).

## See Also

Noeq command line util:
<http://github.com/bmizerany/noeq>

Noeq.go for Go:
<http://github.com/bmizerany/noeq.go>

## Thank you

I want to make sure I give the Snowflake team at Twitter as much credit as
possible. The heart of this program is their doing.

## LICENSE

Copyright (C) 2011 by Blake Mizerany ([@bmizerany](http://twitter.com/bmizerany))

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE. 

[Doozer]: http://github.com/ha/doozerd
[snowflake]: http://github.com/twitter/snowflake
