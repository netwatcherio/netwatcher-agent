# NetWatcher Agent

Latest version: v1.0.5?

## Intro

Netwatcher Agent is an application that ties to the control application. It is intended to be used by MSPs and various service providers to monitor their customer sites while reporting minimally invasive data. It uses the common mtr & rperf library binaries. The control application will be able to be self hosted, with various control & updatability functions in the future.

## Requirements

- See documentation for rperf, trippy, and pro-bing
- macOS, Linux, or Windows
- Running instance of guardian (https://github.com/netwatcherio/guardian) + client (https://github.com/netwatcherio/netwatcher-client)
- probably more

## Installation

Currently it is best to be built from source, and the latest code is not production ready. However, see below. *Development platform is OSX currently, need to make it work on Windows lol...*

1. `git clone https://github.com/netwatcherio/netwatcher-agent`
2. `go build`
3. Run the built application, then exit it (it will generate the configuration)
4. Modify the configuration to contain the PIN created on the control
   * If the agent hasn't been initialized on the control, it will allow the client to connect without including the agent's object ID
   * The agent ID is then saved to the configuration for later requests, as the panel will require it
5. Start the application, it should run it's checks based on the ones configured on the panel
   *Note: currently it requires sudo or set_cap to be used on linux, and Administrative permissions on Windows, with the appropriate firewall rules to allow ICMP, etc.*

Please refer to pro_ping, rperf or trippy's documentation for further information regarding permissions, or submit a pull request/issue with changes. 😄

## Features *WIP*

* [X]  MTR checks (using trippy)
* [X]  rPerf checks (simulated traffic
* [X]  Ping Tests (pro-bing)
* [ ]  Real VoIP checks?
* [ ]  nmap?
* [X]  System Information
* [X]  Network Information
* [ ]  SpeedTests (back on the todo)
* [X]  Check targets fetched from API using WebSockets and JWT
* [X]  Windows Support (kind of)
* [ ]

## Changelog

Just look at commits, eventually I'll make a change log once more stable.

## Libraries

- https://github.com/opensource-3d-p/rperf
- https://github.com/prometheus-community/pro-bing
- https://github.com/fujiapple852/trippy

# License

[`GNU Affero General Public License v3.0`](https://github.com/netwatcherio/netwatcher-agent/blob/master/LICENSE.md)
