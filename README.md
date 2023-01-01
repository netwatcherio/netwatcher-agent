# NetWatcher Agent

## Intro

Netwatcher Agent is an application that ties to the control application. It is intended to be used by MSPs and various service providers to monitor their customer sites while reporting minimally invasive data. It uses the common mtr & iperf3 library binaries. The control application will be able to be self hosted, with various control & updatability functions in the future.

## Installation

Currently it is best to be built from source, and the latest code is not production ready. However, see below. *Development platform is OSX currently, need to make it work on Windows lol...*

1. `git clone https://github.com/netwatcherio/netwatcher-agent`
2. `go build`
3. Run the built application, then exit it (it will generate the configuration)
4. Modify the configuration to contain the PIN created on the control
   * If the agent hasn't been initialized on the control, it will allow the client to connect without including the agent's object ID
   * The agent ID is then saved to the configuration for later requests, as the panel will require it
5. Start the application, it should run it's checks based on the ones configured on the panel

## Features *WIP*

* [X]  MTR checks
* [X]  iPerf checks
* [ ]  Real VoIP checks?
* [X]  Network Information
* [X]  SpeedTests
* [ ]  All check targets grabbed on control (I guess this is more of a control WIP feature)

## Changelog

Just look at commits, eventually I'll make a change log once more stable.

# License

[`GNU Affero General Public License v3.0`](https://github.com/netwatcherio/netwatcher-agent/blob/master/LICENSE.md)
