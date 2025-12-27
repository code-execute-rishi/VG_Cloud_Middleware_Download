.PHONY: all api zerotier telemetry livekit camera clean

all: api zerotier telemetry livekit camera

api:
	go build -o bin/vyom-api cmd/api/main.go

zerotier:
	go build -o bin/vyom-zerotier cmd/zerotier/main.go

telemetry:
	go build -o bin/vyom-telemetry cmd/telemetry/main.go

livekit:
	go build -o bin/vyom-livekit cmd/livekit/main.go

camera:
	go build -o bin/vyom-camera cmd/camera/main.go

clean:
	rm -rf bin/
