.PHONY: all api zerotier telemetry livekit camera ui clean package

VERSION := 1.0.0
ARCH ?= amd64
PKG_NAME := vyom-middleware
PKG_DIR := deb_package
DEB_NAME := $(PKG_NAME)_$(VERSION)_$(ARCH).deb

EXTRA_DEPENDS :=
ifeq ($(ARCH),arm64)
	EXTRA_DEPENDS := gstreamer1.0-libcamera,
endif

all: api zerotier telemetry livekit camera

api:
	GOOS=linux GOARCH=$(ARCH) go build -o bin/vyom-api cmd/api/main.go

zerotier:
	GOOS=linux GOARCH=$(ARCH) go build -o bin/vyom-zerotier cmd/zerotier/main.go

telemetry:
	GOOS=linux GOARCH=$(ARCH) go build -o bin/vyom-telemetry cmd/telemetry/main.go

livekit:
	GOOS=linux GOARCH=$(ARCH) go build -o bin/vyom-livekit cmd/livekit/main.go

camera:
	GOOS=linux GOARCH=$(ARCH) go build -o bin/vyom-camera cmd/camera/main.go

ui:
	cd ui && npm install && npm run build

clean:
	rm -rf bin/ $(PKG_DIR) *.deb

package: all
	@echo "📦 Packaging $(PKG_NAME)..."
	mkdir -p $(PKG_DIR)/opt/vyom/bin
	mkdir -p $(PKG_DIR)/opt/vyom/ui
	mkdir -p $(PKG_DIR)/etc/systemd/system
	mkdir -p $(PKG_DIR)/DEBIAN
	
	# Copy Binaries
	cp bin/* $(PKG_DIR)/opt/vyom/bin/
	cp start_microservices.sh $(PKG_DIR)/opt/vyom/bin/
	chmod +x $(PKG_DIR)/opt/vyom/bin/start_microservices.sh
	
	# Copy UI (Check if dist exists, if not warn)
	if [ -d "ui/dist" ]; then \
		cp -r ui/dist $(PKG_DIR)/opt/vyom/ui/; \
	else \
		echo "⚠️  UI dist not found. Run 'make ui' first if needed."; \
	fi
	
	# Copy Systemd Services (Monolithic)
	cp vyom-middleware.service $(PKG_DIR)/etc/systemd/system/
	
	# Create Control File
	echo "Package: $(PKG_NAME)" > $(PKG_DIR)/DEBIAN/control
	echo "Version: $(VERSION)" >> $(PKG_DIR)/DEBIAN/control
	echo "Section: base" >> $(PKG_DIR)/DEBIAN/control
	echo "Priority: optional" >> $(PKG_DIR)/DEBIAN/control
	echo "Architecture: $(ARCH)" >> $(PKG_DIR)/DEBIAN/control
	echo "Depends: zerotier-one, libgstreamer1.0-0, gstreamer1.0-tools, gstreamer1.0-plugins-base, gstreamer1.0-plugins-good, gstreamer1.0-libav, $(EXTRA_DEPENDS) psmisc, procps" >> $(PKG_DIR)/DEBIAN/control
	echo "Maintainer: Vyom <support@vyom.com>" >> $(PKG_DIR)/DEBIAN/control
	echo "Description: Vyom Device Middleware (Microservices)" >> $(PKG_DIR)/DEBIAN/control
	echo "  Handles Telemetry, Video, and ZeroTier for Vyom Drones." >> $(PKG_DIR)/DEBIAN/control
	
	# Create Post-Install Script
	echo "#!/bin/bash" > $(PKG_DIR)/DEBIAN/postinst
	echo "set -e" >> $(PKG_DIR)/DEBIAN/postinst
	echo "chmod +x /opt/vyom/bin/*" >> $(PKG_DIR)/DEBIAN/postinst
	echo "systemctl daemon-reload" >> $(PKG_DIR)/DEBIAN/postinst
	echo "mkdir -p /var/log/vyom" >> $(PKG_DIR)/DEBIAN/postinst
	echo "chmod 755 /var/log/vyom" >> $(PKG_DIR)/DEBIAN/postinst
	echo "systemctl enable vyom-middleware" >> $(PKG_DIR)/DEBIAN/postinst
	echo "echo '✅ Vyom Middleware Installed.'" >> $(PKG_DIR)/DEBIAN/postinst
	echo "echo 'Run \"systemctl start vyom-middleware\" to begin.'" >> $(PKG_DIR)/DEBIAN/postinst
	chmod 0755 $(PKG_DIR)/DEBIAN/postinst
	
	# Build Deb
	dpkg-deb -Zxz --build $(PKG_DIR) $(DEB_NAME)
	@echo "🎉 Package Created: $(DEB_NAME)"
