APP_NAME := vyom-middleware
VERSION := 2.5.0
ARCH := arm64
PKG_DIR := deb_package_final

.PHONY: build-rpi build-local clean package-rpi

clean:
	rm -rf $(PKG_DIR)
	rm -f $(APP_NAME)
	rm -f middleware-arm64

build-rpi:
	export GOOS=linux GOARCH=arm64 CGO_ENABLED=0 && go build -o $(APP_NAME) .
	@echo "✅ Build Complete (ARM64): $(APP_NAME)"

build-local:
	go build -o $(APP_NAME) .
	@echo "✅ Build Complete (Local): $(APP_NAME)"

package-rpi: clean
	@echo "📦 Packaging for Raspberry Pi..."
	mkdir -p $(PKG_DIR)/usr/local/bin
	mkdir -p $(PKG_DIR)/etc/systemd/system
	mkdir -p $(PKG_DIR)/opt/vyom/ui
	mkdir -p $(PKG_DIR)/DEBIAN
	
	# Build Binary
	$(MAKE) build-rpi
	mv $(APP_NAME) $(PKG_DIR)/usr/local/bin/
	
	# Build UI
	cd ui && npm install && npm run build
	cp -r ui/dist/* $(PKG_DIR)/opt/vyom/ui/
	
	# Config
	cp vyom-middleware.service $(PKG_DIR)/etc/systemd/system/
	
	# Control File
	@echo "Package: $(APP_NAME)\nVersion: $(VERSION)\nSection: utils\nPriority: optional\nArchitecture: $(ARCH)\nMaintainer: Vyom <support@vyom.ai>\nDescription: Vyom Device Middleware\nDepends: gstreamer1.0-tools, gstreamer1.0-plugins-good, gstreamer1.0-plugins-bad, gstreamer1.0-libav, v4l-utils, libcamera-apps, rpicam-apps" > $(PKG_DIR)/DEBIAN/control
	
	# Post Inst
	@echo "#!/bin/bash\nchmod +x /usr/local/bin/$(APP_NAME)\nchmod 755 /opt/vyom/ui\nsystemctl daemon-reload\nsystemctl enable $(APP_NAME)\nsystemctl restart $(APP_NAME)" > $(PKG_DIR)/DEBIAN/postinst
	chmod 755 $(PKG_DIR)/DEBIAN/postinst
	
	# Build Deb
	dpkg-deb --build $(PKG_DIR) "$(APP_NAME)_$(VERSION)_$(ARCH).deb"
	@echo "🎉 Package Created: $(APP_NAME)_$(VERSION)_$(ARCH).deb"

package-local: clean
	@echo "📦 Packaging for Local Machine (AMD64)..."
	mkdir -p $(PKG_DIR)/usr/local/bin
	mkdir -p $(PKG_DIR)/etc/systemd/system
	mkdir -p $(PKG_DIR)/opt/vyom/ui
	mkdir -p $(PKG_DIR)/DEBIAN
	
	# Build Binary
	$(MAKE) build-local
	mv $(APP_NAME) $(PKG_DIR)/usr/local/bin/
	
	# Build UI
	cd ui && npm install && npm run build
	cp -r ui/dist/* $(PKG_DIR)/opt/vyom/ui/
	
	# Config
	cp vyom-middleware.service $(PKG_DIR)/etc/systemd/system/
	
	# Control File (AMD64, removed rpicam-apps dependency for local test)
	@echo "Package: $(APP_NAME)\nVersion: $(VERSION)\nSection: utils\nPriority: optional\nArchitecture: amd64\nMaintainer: Vyom <support@vyom.ai>\nDescription: Vyom Device Middleware (Local)\nDepends: gstreamer1.0-tools, gstreamer1.0-plugins-good, gstreamer1.0-plugins-bad, gstreamer1.0-libav" > $(PKG_DIR)/DEBIAN/control
	
	# Post Inst
	@echo "#!/bin/bash\nchmod +x /usr/local/bin/$(APP_NAME)\nchmod 755 /opt/vyom/ui\nsystemctl daemon-reload\nsystemctl enable $(APP_NAME)\nsystemctl restart $(APP_NAME)" > $(PKG_DIR)/DEBIAN/postinst
	chmod 755 $(PKG_DIR)/DEBIAN/postinst
	
	# Build Deb
	dpkg-deb --build $(PKG_DIR) "$(APP_NAME)_$(VERSION)_amd64.deb"
	@echo "🎉 Package Created: $(APP_NAME)_$(VERSION)_amd64.deb"
