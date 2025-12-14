#!/bin/bash
set -e

APP_NAME="vyom-middleware"
VERSION="2.3.0"
ARCH="arm64"
PKG_DIR="deb_package"

echo "📦 Starting Debian Package Build for $APP_NAME v$VERSION ($ARCH)..."

# 1. Clean & Prepare Directories
echo "🧹 Cleaning up..."
rm -rf $PKG_DIR
mkdir -p $PKG_DIR/usr/local/bin
mkdir -p $PKG_DIR/etc/systemd/system
mkdir -p $PKG_DIR/opt/vyom/ui
mkdir -p $PKG_DIR/DEBIAN

# 2. Build Middleware (Go)
echo "🔨 Building Middleware Binary (ARM64)..."
export GOOS=linux
export GOARCH=arm64
go build -o $PKG_DIR/usr/local/bin/$APP_NAME .

# 3. Build UI (React)
echo "⚛️  Building Local UI..."
cd ui
npm install
npm run build
cd ..
cp -r ui/dist/* $PKG_DIR/opt/vyom/ui/

# 4. Config & Service Files
echo "📄 Copying Service File..."
cp vyom-middleware.service $PKG_DIR/etc/systemd/system/

# 5. Create Control File
echo "📝 Creating Control File..."
cat <<EOF > $PKG_DIR/DEBIAN/control
Package: $APP_NAME
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Maintainer: Vyom <support@vyom.ai>
Description: Vyom Device Middleware
 Key bridge between Pixhawk, Camera, and Vyom Cloud.
 Auto-detects hardware and manages telemetry.
Depends: gstreamer1.0-tools, gstreamer1.0-plugins-good, gstreamer1.0-plugins-bad, v4l-utils
EOF

# 6. Create Post-Install Script
echo "🔧 Creating Post-Install Script..."
cat <<EOF > $PKG_DIR/DEBIAN/postinst
#!/bin/bash
# Set Permissions
chmod +x /usr/local/bin/$APP_NAME
chmod 755 /opt/vyom/ui

# Reload Systemd
systemctl daemon-reload
systemctl enable $APP_NAME
systemctl restart $APP_NAME

echo "✅ Vyom Middleware Installed & Started!"
EOF
chmod 755 $PKG_DIR/DEBIAN/postinst

# 7. Build Package
echo "📦 Building .deb..."
dpkg-deb --build $PKG_DIR "${APP_NAME}_${VERSION}_${ARCH}.deb"

echo "🎉 Build Complete: ${APP_NAME}_${VERSION}_${ARCH}.deb"
