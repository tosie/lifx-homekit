GO ?= go
SCP ?= scp

rpi:
	GOOS=linux GOARCH=arm GOARM=6 $(GO) build lifx-homekit.go
	
deploy: rpi
	ssh pi@10.0.1.213 sudo /etc/init.d/lifx-homekit stop
	$(SCP) lifx-homekit pi@10.0.1.213:~/lifx-homekit/
	ssh pi@10.0.1.213 sudo /etc/init.d/lifx-homekit start