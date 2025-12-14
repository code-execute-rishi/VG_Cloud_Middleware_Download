#!/bin/bash
set -e

APP_NAME="vyom-middleware-local"
VERSION="2.2.0"
ARCH="amd64"
PKG_DIR="deb_package_local"

echo "📦 Starting Local Debian Package Build for $APP_NAME v$VERSION ($ARCH)..."

# 1. Clean & Prepare Directories
echo "🧹 Cleaning up..."
rm -rf $PKG_DIR
mkdir -p $PKG_DIR/usr/local/bin
mkdir -p $PKG_DIR/etc/systemd/system
mkdir -p $PKG_DIR/opt/vyom/ui
mkdir -p $PKG_DIR/DEBIAN

# 2. Build Middleware (Go) - FOR LOCAL SYSTEM
echo "🔨 Building Middleware Binary (AMD64)..."
export GOOS=linux
export GOARCH=amd64
go build -o $PKG_DIR/usr/local/bin/vyom-middleware .

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
Description: Vyom Device Middleware (Local Test)
 Key bridge between Pixhawk, Camera, and Vyom Cloud.
 Auto-detects hardware and manages telemetry.
EOF

# 6. Create Post-Install Script
echo "🔧 Creating Post-Install Script..."
cat <<EOF > $PKG_DIR/DEBIAN/postinst
#!/bin/bash
# Set Permissions
chmod +x /usr/local/bin/vyom-middleware
chmod 755 /opt/vyom/ui

# Reload Systemd
systemctl daemon-reload
# WARNING: We do NOT auto-enable/start on local dev machine to avoid conflicts
# systemctl enable vyom-middleware
# systemctl restart vyom-middleware

echo "✅ Vyom Middleware Installed (Local Test)!"
echo "ℹ️  Service is NOT started automatically to prevent port conflicts."
echo "ℹ️  To start: sudo systemctl start vyom-middleware"
EOF
chmod 755 $PKG_DIR/DEBIAN/postinst

# 7. Build Package
echo "📦 Building .deb..."
dpkg-deb --build $PKG_DIR "${APP_NAME}_${VERSION}_${ARCH}.deb"

echo "🎉 Local Build Complete: ${APP_NAME}_${VERSION}_${ARCH}.deb"
