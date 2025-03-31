#!/bin/bash
# build_appstore.sh - Script to build and package Chapar for the Mac App Store

set -e

# Ensure required variables are set
if [ -z "$APPLE_TEAM_ID" ]; then
  echo "ERROR: APPLE_TEAM_ID environment variable is not set"
  exit 1
fi

# Version information
VERSION=$(echo "$1" | sed 's/^v//')  # Remove 'v' prefix if present
if [ -z "$VERSION" ]; then
  echo "ERROR: Version must be provided as the first argument (e.g., 0.1.0)"
  exit 1
fi

BUILD_NUMBER=$(date +%Y%m%d%H%M)  # Use date/time as build number

echo "Building Chapar version $VERSION (build $BUILD_NUMBER) for App Store..."

# Create directories
mkdir -p ./dist/appstore/amd64
mkdir -p ./dist/appstore/arm64
mkdir -p ./dist/appstore/universal/Chapar.app/Contents/MacOS
mkdir -p ./dist/appstore/universal/Chapar.app/Contents/Resources
mkdir -p ./build/appstore

# Create a proper Info.plist
cat > ./build/appstore/Info.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>Chapar</string>
	<key>CFBundleIdentifier</key>
	<string>rest.chapar.app</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>Chapar</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$VERSION</string>
	<key>CFBundleVersion</key>
	<string>$BUILD_NUMBER</string>
	<key>LSMinimumSystemVersion</key>
	<string>10.14</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSPrincipalClass</key>
	<string>NSApplication</string>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.developer-tools</string>
	<key>CFBundleIconFile</key>
  <string>appicon</string>
</dict>
</plist>
EOF

# Create entitlements file for App Store
cat > ./build/appstore/entitlements.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.app-sandbox</key>
  <true/>
  <key>com.apple.security.network.client</key>
  <true/>
  <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
  <true/>
  <key>com.apple.security.cs.allow-jit</key>
  <true/>
</dict>
</plist>
EOF

# Build for App Store (both architectures)
echo "Building for Intel (amd64)..."
gogio -ldflags="-X main.serviceVersion=$VERSION" -appid=rest.chapar.app -icon=./build/appicon.png -target=macos -arch=amd64 -o ./dist/appstore/amd64/Chapar.app .

echo "Building for Apple Silicon (arm64)..."
gogio -ldflags="-X main.serviceVersion=$VERSION" -appid=rest.chapar.app -icon=./build/appicon.png -target=macos -arch=arm64 -o ./dist/appstore/arm64/Chapar.app .

# Create universal app bundle structure
echo "Creating universal app structure..."
mkdir -p ./dist/appstore/universal/Chapar.app/Contents/Resources
cp -R ./dist/appstore/arm64/Chapar.app/Contents/Resources/* ./dist/appstore/universal/Chapar.app/Contents/Resources/ || true
cp ./dist/appstore/arm64/Chapar.app/Contents/Info.plist ./dist/appstore/universal/Chapar.app/Contents/
cp ./build/appstore/Info.plist ./dist/appstore/universal/Chapar.app/Contents/

# Copy the ICNS file to Resources
if [ -f ./build/appicon.icns ]; then
  echo "Copying ICNS file to app bundle..."
  cp ./build/appicon.icns ./dist/appstore/universal/Chapar.app/Contents/Resources/

  # Make sure Info.plist references the icon
  plutil -replace CFBundleIconFile -string "appicon" ./dist/appstore/universal/Chapar.app/Contents/Info.plist
else
  echo "ERROR: No ICNS file found at ./build/appicon.icns"
  exit 1
fi

# Use lipo to create universal binary
echo "Creating universal binary..."
lipo -create ./dist/appstore/amd64/Chapar.app/Contents/MacOS/Chapar ./dist/appstore/arm64/Chapar.app/Contents/MacOS/Chapar -output ./dist/appstore/universal/Chapar.app/Contents/MacOS/Chapar

# Sign the universal app with proper entitlements
echo "Signing universal app with entitlements..."
codesign --force --options runtime --entitlements ./build/appstore/entitlements.plist --deep --sign "3rd Party Mac Developer Application: Mohsen Mirzakhani ($APPLE_TEAM_ID)" ./dist/appstore/universal/Chapar.app

# Verify code signing
codesign -dvv ./dist/appstore/universal/Chapar.app

# Create a temporary directory for the flat package
mkdir -p ./dist/appstore/pkg_root/Applications
cp -R ./dist/appstore/universal/Chapar.app ./dist/appstore/pkg_root/Applications/

# Create a proper product archive using Xcode's productbuild
echo "Building App Store package..."
productbuild --component ./dist/appstore/universal/Chapar.app /Applications \
  --sign "3rd Party Mac Developer Installer: Mohsen Mirzakhani ($APPLE_TEAM_ID)" \
  ./dist/Chapar-v$VERSION.pkg

# Verify the package
echo "Verifying package..."
pkgutil --check-signature ./dist/Chapar-v$VERSION.pkg

echo "Package built: ./dist/Chapar-v$VERSION.pkg"
echo
echo "To upload to App Store Connect, run:"
echo "xcrun altool --upload-app -f ./dist/Chapar-v$VERSION.pkg -t macos -u \"\$APPLE_ID\" -p \"\$APPLE_APP_SPECIFIC_PASSWORD\" --verbose"
