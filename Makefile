GO ?= go

rpi:
	GOOS=linux GOARCH=arm GOARM=6 $(GO) build lifx-homekit.go