# lifx-homekit

This projects combines http://github.com/brutella/hc and http://github.com/pdf/golifx to make LIFX light bulbs available to Apple HomeKit.

## Usage

At first you need to install Go (https://golang.org) and set up your $GOPATH (https://github.com/golang/go/wiki/GOPATH).

When your `go` command works you can run

`go get -u github.com/tosie/lifx-homekit`

to download the source code and all of its dependencies to your computer. They will be put into `$GOPATH/src/github.com/tosie/lifx-homekit`.

After downloading the projects and dependencies, go to the project's root directory:

`cd $GOPATH/src/github.com/tosie/lifx-homekit`

Here you can run

`go build lifx-homekit.go`

to compile the project. The resulting file will be named `lifx-homekit` and can be run immediately via

`./lifx-homekit --pin 12341234`

At startup you must provide a pin code that you can choose by yourself, but must be made up of exactly eight digits (e.g. 12341234). Other than this pin code there is no other configuration to be done. The LIFX lights will be discovered automatically and, upon discovery, will be announced as HomeKit devices.

Now you need to use a HomeKit app in your iOS device (e.g. eve, Insteon, Home, ...; just search for HomeKit on the App Store) to setup the LIFX bulbs as HomeKit devices.



## Running as a server on a Raspberry Pi

When you run

`GOOS=linux GOARCH=arm GOARM=6 go build lifx-homekit.go`

the project will be compiled in a way that makes it run on a Raspberry Pi. In the future there might be a manual here to describe the steps neccessary to set up everything, but for now this must suffice as a starting point for you.

