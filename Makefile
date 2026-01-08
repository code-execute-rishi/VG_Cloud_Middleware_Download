.PHONY: all api zerotier telemetry livekit camera ui clean package

VERSION := 1.0.23
ARCH ?= amd64
PKG_NAME := vyom-middleware
PKG_DIR := deb_package
DEB_NAME := $(PKG_NAME)_$(VERSION)_$(ARCH).deb

EXTRA_DEPENDS :=
GO_ARCH := $(ARCH)
GO_ARM :=

ifeq ($(ARCH),arm64)
	EXTRA_DEPENDS := gstreamer1.0-libcamera,
	GO_ARCH := arm64
endif
ifeq ($(ARCH),armhf)
	EXTRA_DEPENDS := gstreamer1.0-libcamera,
	GO_ARCH := arm
	GO_ARM := 7
endif

.PHONY: package-all package-arm64 package-armhf package-amd64 clean clean-bin

package-all: package-amd64 package-arm64 package-armhf

package-arm64:
	$(MAKE) clean-bin
	@echo "BUILDING ARM64..."
	$(MAKE) package ARCH=arm64

package-armhf:
	$(MAKE) clean-bin
	@echo "BUILDING ARMHF..."
	$(MAKE) package ARCH=armhf

package-amd64:
	$(MAKE) clean-bin
	@echo "BUILDING AMD64..."
	$(MAKE) package ARCH=amd64

all: api zerotier telemetry livekit camera

api:
	GOOS=linux GOARCH=$(GO_ARCH) GOARM=$(GO_ARM) go build -o bin/vyom-api cmd/api/main.go

zerotier:
	GOOS=linux GOARCH=$(GO_ARCH) GOARM=$(GO_ARM) go build -o bin/vyom-zerotier cmd/zerotier/main.go

telemetry:
	GOOS=linux GOARCH=$(GO_ARCH) GOARM=$(GO_ARM) go build -o bin/vyom-telemetry cmd/telemetry/main.go

livekit:
	GOOS=linux GOARCH=$(GO_ARCH) GOARM=$(GO_ARM) go build -o bin/vyom-livekit cmd/livekit/main.go

camera:
	GOOS=linux GOARCH=$(GO_ARCH) GOARM=$(GO_ARM) go build -o bin/vyom-camera cmd/camera/main.go

ui:
	cd ui && npm install && npm run build

clean:
	rm -rf bin/ $(PKG_DIR) *.deb

clean-bin:
	rm -rf bin/ $(PKG_DIR)

package: all ui
	@echo "📦 Packaging $(PKG_NAME)..."
	mkdir -p $(PKG_DIR)/opt/vyom/bin
	mkdir -p $(PKG_DIR)/opt/vyom/ui
	mkdir -p $(PKG_DIR)/etc/systemd/system
	mkdir -p $(PKG_DIR)/DEBIAN
	
	# Copy Binaries
	cp bin/* $(PKG_DIR)/opt/vyom/bin/
	
	# Copy UI
	if [ -d "ui/dist" ]; then \
		cp -r ui/dist $(PKG_DIR)/opt/vyom/ui/; \
	else \
		echo "❌ UI dist not found. Build failed."; \
		exit 1; \
	fi
	
	# Copy Systemd Services
	cp vyom-api.service $(PKG_DIR)/etc/systemd/system/
	cp vyom-zerotier.service $(PKG_DIR)/etc/systemd/system/
	cp vyom-telemetry.service $(PKG_DIR)/etc/systemd/system/
	cp vyom-livekit.service $(PKG_DIR)/etc/systemd/system/
	cp vyom-camera.service $(PKG_DIR)/etc/systemd/system/
	
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
	echo "Conflicts: vyom-middleware" >> $(PKG_DIR)/DEBIAN/control
	echo "Replaces: vyom-middleware" >> $(PKG_DIR)/DEBIAN/control
	echo "Provides: vyom-middleware" >> $(PKG_DIR)/DEBIAN/control
	
	# Create Post-Install Script
	echo "#!/bin/bash" > $(PKG_DIR)/DEBIAN/postinst
	echo "set -e" >> $(PKG_DIR)/DEBIAN/postinst
	echo "chmod +x /opt/vyom/bin/*" >> $(PKG_DIR)/DEBIAN/postinst
	echo "systemctl daemon-reload" >> $(PKG_DIR)/DEBIAN/postinst
	echo "systemctl daemon-reload" >> $(PKG_DIR)/DEBIAN/postinst
	echo "mkdir -p /var/log/vyom" >> $(PKG_DIR)/DEBIAN/postinst
	echo "chmod 755 /var/log/vyom" >> $(PKG_DIR)/DEBIAN/postinst
	echo "mkdir -p /etc/vyom" >> $(PKG_DIR)/DEBIAN/postinst
	echo "chmod 777 /etc/vyom" >> $(PKG_DIR)/DEBIAN/postinst
	
	# Cleanup Legacy
	# Cleanup Legacy (Quiet)
	echo "if [ -f /etc/systemd/system/vyom-middleware.service ]; then" >> $(PKG_DIR)/DEBIAN/postinst
	echo "  systemctl stop vyom-middleware || true" >> $(PKG_DIR)/DEBIAN/postinst
	echo "  systemctl disable vyom-middleware || true" >> $(PKG_DIR)/DEBIAN/postinst
	echo "  rm -f /etc/systemd/system/vyom-middleware.service" >> $(PKG_DIR)/DEBIAN/postinst
	echo "fi" >> $(PKG_DIR)/DEBIAN/postinst
	
	# Enable and Restart New Services
	echo "systemctl daemon-reload" >> $(PKG_DIR)/DEBIAN/postinst
	echo "systemctl enable vyom-api vyom-zerotier vyom-telemetry vyom-livekit vyom-camera" >> $(PKG_DIR)/DEBIAN/postinst
	echo "systemctl restart vyom-api vyom-zerotier vyom-telemetry vyom-livekit vyom-camera" >> $(PKG_DIR)/DEBIAN/postinst
	
	echo "echo '✅ Vyom Middleware Installed and Started.'" >> $(PKG_DIR)/DEBIAN/postinst
	chmod 0755 $(PKG_DIR)/DEBIAN/postinst
	
	# Build Deb
	dpkg-deb --build $(PKG_DIR) $(DEB_NAME)
	@echo "🎉 Package Created: $(DEB_NAME)"
