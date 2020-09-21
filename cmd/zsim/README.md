# zsim - zq simulator

This directory implements zsim, a command-line tool for running multiple
instances of the zq data engine in a simulated environment.
Time runs in simulated time and communication is simply modeled
as latency and throughout delays.  This is easily implemented by
wrapping timers and the http package with some simulation event hooks.

> TBD: right now, this is hardwired to some simple test runs modeling
> a developer use case to motivate HLAP.  The simulation framework
> is coming soon.
