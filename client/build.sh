#!/bin/bash
# Build DessauChat.app without an Xcode project: several Swift files, one
# script. Needs the installed Xcode's toolchain (`xcrun`) and its macOS 27 SDK —
# the 27 SDK's SwiftUI is implemented with compiler macros whose plugin ships
# only inside Xcode, so the Command Line Tools alone cannot build this app.
set -euo pipefail
cd "$(dirname "$0")"

APP="DessauChat"
BUNDLE="dist/$APP.app"
MACOS="$BUNDLE/Contents/MacOS"
RES="$BUNDLE/Contents/Resources"
# The deployment target is the client's floor; client/Info.plist declares the
# same value and a test in the server's suite holds the two together. macOS 27
# runs on no Intel Mac, so there is one slice.
TARGET="arm64-apple-macos27.0"
SDK="$(xcrun --sdk macosx --show-sdk-path)"
OBJ="dist/obj"

echo "Compiling $APP against $(basename "$SDK")..."
rm -rf "$BUNDLE" "$OBJ"
mkdir -p "$MACOS" "$RES" "$OBJ"

# One module, whole-module: -emit-const-values writes the compile-time
# constants the App Intents metadata processor reads (which intents exist,
# their parameters and phrases) beside the object file, gathered for the
# protocols named in the list.
xcrun swiftc -O -wmo -c -parse-as-library \
    -swift-version 6 -default-isolation MainActor \
    -sdk "$SDK" -target "$TARGET" -module-name "$APP" \
    -emit-const-values \
    -Xfrontend -const-gather-protocols-file -Xfrontend appintents-protocols.json \
    -o "$OBJ/$APP.o" \
    DessauChat/*.swift
xcrun swiftc -sdk "$SDK" -target "$TARGET" -o "$MACOS/$APP" "$OBJ/$APP.o"

# App Intents: the system lists an app's actions from Metadata.appintents in
# its bundle, which Xcode writes at build time and this script writes the same
# way. A bundle without it has no actions in Shortcuts or Spotlight and says
# nothing about why, so a processor that writes nothing fails the build here.
CONSTVALS="$OBJ/$APP.swiftconstvalues"
[ -f "$CONSTVALS" ] || CONSTVALS="$(ls "$OBJ"/*.swiftconstvalues | head -1)"
printf '%s\n' "$CONSTVALS" > "$OBJ/constvals.txt"
ls "$PWD"/DessauChat/*.swift > "$OBJ/sources.txt"
xcrun appintentsmetadataprocessor \
    --output "$RES" \
    --toolchain-dir "$(xcode-select -p)/Toolchains/XcodeDefault.xctoolchain" \
    --module-name "$APP" \
    --sdk-root "$SDK" \
    --xcode-version "$(xcodebuild -version | awk '/Build version/{print $3}')" \
    --platform-family macOS \
    --deployment-target "${TARGET##*macos}" \
    --target-triple "$TARGET" \
    --source-file-list "$OBJ/sources.txt" \
    --swift-const-vals-list "$OBJ/constvals.txt" \
    --force
[ -f "$RES/Metadata.appintents/extract.actionsdata" ] ||
    { echo "error: the App Intents metadata was not written; Shortcuts would list nothing" >&2; exit 1; }

cp Info.plist "$BUNDLE/Contents/Info.plist"
if [ -n "${VERSION:-}" ]; then
    /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${VERSION#v}" \
        "$BUNDLE/Contents/Info.plist"
fi

if [ ! -f icon/AppIcon.icns ]; then ./mkicon.sh; fi
cp icon/AppIcon.icns "$RES/AppIcon.icns"

codesign --force --identifier sh.intentdriven.dessau.chat --sign - "$BUNDLE"
rm -rf "$OBJ"
echo "Built $BUNDLE"
echo "Run it with:  open $BUNDLE"
