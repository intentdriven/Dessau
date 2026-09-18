#!/bin/bash
# Build GropiusChat for the iPad from the same Swift files build.sh compiles
# for the Mac: no Xcode project, one script. Needs the installed Xcode's
# toolchain (`xcrun`) and its iOS 27 SDK — as on macOS, the 27 SDK's SwiftUI is
# implemented with compiler macros whose plugin ships only inside Xcode.
#
#   SIM=1 ./build-ipad.sh    build for the simulator, install it and launch it
#   ./build-ipad.sh          build for a device; needs a signing identity
#
# A device build is signed with the maintainer's own free personal team, for
# their own iPad over USB (itd-2609180943290800). Nothing here is published.
set -euo pipefail
cd "$(dirname "$0")"

APP="GropiusChat"
BUNDLE="dist-ipad/$APP.app"
OBJ="dist-ipad/obj"
# The client's floor on this system; client/Info-iPad.plist declares the same
# value as MinimumOSVersion. A target above the bundle minimum is the dangerous
# direction: the system admits the iPad and dyld then kills the app at exec.
DEPLOYMENT_TARGET="27.0"

if [ -n "${SIM:-}" ]; then
    SDK_NAME="iphonesimulator"
    TARGET="arm64-apple-ios$DEPLOYMENT_TARGET-simulator"
    ACTOOL_PLATFORM="iphonesimulator"
else
    SDK_NAME="iphoneos"
    TARGET="arm64-apple-ios$DEPLOYMENT_TARGET"
    ACTOOL_PLATFORM="iphoneos"
    # Checked before anything is built rather than after: a bundle that cannot
    # be signed is a bundle no iPad will install, and finding that out at the
    # end of a build reads as a signing hiccup rather than a missing setup step.
    if [ -z "${IPAD_SIGNING_IDENTITY:-}" ] || [ -z "${IPAD_PROFILE:-}" ]; then
        cat >&2 <<'MSG'
error: a device build needs a development signing identity and a provisioning
profile. An iPad refuses an ad-hoc signature, so this script will not write a
bundle it knows no iPad can install.

One sign-in mints both: open Xcode > Settings > Accounts, add your Apple ID,
and let the personal team issue an "Apple Development" certificate; the
matching profile for dev.gropius.chat is in
~/Library/Developer/Xcode/UserData/Provisioning Profiles.

Then:
  IPAD_SIGNING_IDENTITY="Apple Development: <your name> (XXXXXXXXXX)" \
  IPAD_PROFILE=/path/to/profile.mobileprovision \
  ./build-ipad.sh

Or build for the simulator instead, which needs neither:  SIM=1 ./build-ipad.sh
MSG
        exit 1
    fi
fi

SDK="$(xcrun --sdk "$SDK_NAME" --show-sdk-path)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Compiling $APP against $(basename "$SDK")..."
rm -rf "$BUNDLE" "$OBJ"
# An iOS bundle is flat: the executable, the Info.plist and the resources sit
# at the top of the .app, where a macOS bundle has Contents/.
mkdir -p "$BUNDLE" "$OBJ"

# One module, whole-module: -emit-const-values writes the compile-time
# constants the App Intents metadata processor reads (which intents exist,
# their parameters and phrases) beside the object file, gathered for the
# protocols named in the list. The same flags as the Mac build.
xcrun swiftc -O -wmo -c -parse-as-library \
    -swift-version 6 -default-isolation MainActor \
    -sdk "$SDK" -target "$TARGET" -module-name "$APP" \
    -emit-const-values \
    -Xfrontend -const-gather-protocols-file -Xfrontend appintents-protocols.json \
    -o "$OBJ/$APP.o" \
    GropiusChat/*.swift
# The link step drives clang, which otherwise reaches for the host's macOS
# sysroot and says so on every build; -isysroot points it at the same SDK the
# frontend compiled against.
xcrun swiftc -sdk "$SDK" -target "$TARGET" \
    -Xclang-linker -isysroot -Xclang-linker "$SDK" \
    -o "$BUNDLE/$APP" "$OBJ/$APP.o"

# App Intents: the system lists an app's actions from Metadata.appintents in
# its bundle, which Xcode writes at build time and this script writes the same
# way. A bundle without it has no actions in Shortcuts or Spotlight and says
# nothing about why, so a processor that writes nothing fails the build here.
# The platform family is the one difference from the Mac build: metadata
# extracted as macOS describes a system this bundle will never run on.
CONSTVALS="$OBJ/$APP.swiftconstvalues"
if [ ! -f "$CONSTVALS" ]; then
    # `|| true` so that a glob matching nothing reaches the message below
    # rather than killing the script under `set -e` with no explanation.
    CONSTVALS="$(ls "$OBJ"/*.swiftconstvalues 2>/dev/null | sed -n '1p' || true)"
fi
[ -n "$CONSTVALS" ] && [ -f "$CONSTVALS" ] ||
    { echo "error: the compiler wrote no .swiftconstvalues; the App Intents metadata would describe nothing" >&2; exit 1; }
printf '%s\n' "$CONSTVALS" > "$OBJ/constvals.txt"
ls "$PWD"/GropiusChat/*.swift > "$OBJ/sources.txt"
xcrun appintentsmetadataprocessor \
    --output "$BUNDLE" \
    --toolchain-dir "$(xcode-select -p)/Toolchains/XcodeDefault.xctoolchain" \
    --module-name "$APP" \
    --sdk-root "$SDK" \
    --xcode-version "$(xcodebuild -version | awk '/Build version/{print $3}')" \
    --platform-family iOS \
    --deployment-target "$DEPLOYMENT_TARGET" \
    --target-triple "$TARGET" \
    --source-file-list "$OBJ/sources.txt" \
    --swift-const-vals-list "$OBJ/constvals.txt" \
    --force
[ -f "$BUNDLE/Metadata.appintents/extract.actionsdata" ] ||
    { echo "error: the App Intents metadata was not written; Shortcuts would list nothing" >&2; exit 1; }

cp Info-iPad.plist "$BUNDLE/Info.plist"
if [ -n "${VERSION:-}" ]; then
    /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${VERSION#v}" \
        "$BUNDLE/Info.plist"
fi

# The icon. iOS reads a compiled asset catalogue (Assets.car) plus the icon
# keys actool writes into the plist; the .icns the Mac bundle carries means
# nothing here. The catalogue is written from the same source art rather than
# committed, so there is one icon source for both bundles.
ICON_PNG="icon/icon-1024.png"
if [ ! -f "$ICON_PNG" ]; then
    if [ -f icon/AppIcon.icns ]; then
        # The committed .icns already holds the 1024 rendering; unpacking it
        # needs no extra tool, where re-rendering the SVG needs rsvg-convert.
        iconutil --convert iconset --output "$TMP/AppIcon.iconset" icon/AppIcon.icns
        ICON_PNG="$TMP/AppIcon.iconset/icon_512x512@2x.png"
    else
        ./mkicon.sh
        ICON_PNG="icon/icon-1024.png"
    fi
fi
mkdir -p "$TMP/Assets.xcassets/AppIcon.appiconset"
cp "$ICON_PNG" "$TMP/Assets.xcassets/AppIcon.appiconset/icon-1024.png"
cat > "$TMP/Assets.xcassets/AppIcon.appiconset/Contents.json" <<'JSON'
{
  "images" : [
    {
      "filename" : "icon-1024.png",
      "idiom" : "universal",
      "platform" : "ios",
      "size" : "1024x1024"
    }
  ],
  "info" : { "author" : "gropius", "version" : 1 }
}
JSON
cat > "$TMP/Assets.xcassets/Contents.json" <<'JSON'
{ "info" : { "author" : "gropius", "version" : 1 } }
JSON
xcrun actool --compile "$BUNDLE" \
    --platform "$ACTOOL_PLATFORM" \
    --minimum-deployment-target "$DEPLOYMENT_TARGET" \
    --target-device ipad \
    --app-icon AppIcon \
    --output-partial-info-plist "$TMP/icon-partial.plist" \
    "$TMP/Assets.xcassets" > "$TMP/actool.log" 2>&1 ||
    { echo "error: actool could not compile the icon:" >&2; cat "$TMP/actool.log" >&2; exit 1; }
# actool reports which icon it compiled in a partial plist; without those keys
# merged in, the home screen shows the generic placeholder — and the app's own
# empty state, which reads the same keys, falls back to a symbol. A silent icon
# step is the one the App Intents step above refuses to be.
/usr/libexec/PlistBuddy -c "Merge $TMP/icon-partial.plist" "$BUNDLE/Info.plist"
[ -f "$BUNDLE/Assets.car" ] ||
    { echo "error: actool wrote no Assets.car; the app would carry no icon" >&2
      cat "$TMP/actool.log" >&2; exit 1; }
/usr/libexec/PlistBuddy -c "Print :CFBundleIcons:CFBundlePrimaryIcon:CFBundleIconFiles" \
    "$BUNDLE/Info.plist" > /dev/null 2>&1 ||
    { echo "error: the compiled icon's keys were not merged into Info.plist; the home screen would show a placeholder" >&2; exit 1; }

if [ -z "${SIM:-}" ]; then
    # Development signing. The profile is what the iPad checks the signature
    # against, and it carries the team that owns the application identifier, so
    # the entitlements are read out of the profile rather than typed in twice.
    cp "$IPAD_PROFILE" "$BUNDLE/embedded.mobileprovision"
    security cms -D -i "$IPAD_PROFILE" > "$TMP/profile.plist"
    /usr/libexec/PlistBuddy -x -c "Print :Entitlements" "$TMP/profile.plist" > "$TMP/entitlements.plist"
    TEAM_ID="$(/usr/libexec/PlistBuddy -c "Print :com.apple.developer.team-identifier" "$TMP/entitlements.plist")"
    /usr/libexec/PlistBuddy -c "Set :application-identifier $TEAM_ID.dev.gropius.chat" "$TMP/entitlements.plist"
    # get-task-allow is what lets a debugger attach; a personal team's profile
    # is a development profile, and this is the entitlement it is issued for.
    /usr/libexec/PlistBuddy -c "Set :get-task-allow true" "$TMP/entitlements.plist" 2>/dev/null ||
        /usr/libexec/PlistBuddy -c "Add :get-task-allow bool true" "$TMP/entitlements.plist"
    codesign --force --sign "$IPAD_SIGNING_IDENTITY" \
        --entitlements "$TMP/entitlements.plist" \
        --timestamp=none "$BUNDLE"
fi

rm -rf "$OBJ"
echo "Built $BUNDLE"

if [ -z "${SIM:-}" ]; then
    if [ -n "${IPAD_DEVICE:-}" ]; then
        echo "Installing on $IPAD_DEVICE..."
        xcrun devicectl device install app --device "$IPAD_DEVICE" "$BUNDLE"
    else
        echo "Install it with:  xcrun devicectl device install app --device <udid> $BUNDLE"
    fi
    exit 0
fi

# The simulator run: the build Mac's own check that the bundle starts. The
# simulator borrows the Mac's language model, so the on-device path can be
# exercised here even where no iPad this project owns is eligible for it.
# A simulator is named or identified; the udid is taken from the listing
# because an iPad's name carries parentheses of its own ("iPad Pro 11-inch
# (M5)") and only the udid is unambiguous. It is also what every simctl call
# below names: "booted" would be whichever simulator happens to be running,
# which on a Mac with an iPhone simulator open is not this app's iPad at all.
DEVICE="${IPAD_SIM:-}"
if [ -z "$DEVICE" ]; then
    # Only the iOS 27 section of the listing: an iPad simulator on an older
    # runtime cannot run a bundle whose minimum is 27.0, and `|| true` leaves
    # an empty result to the message below rather than ending the script here.
    LISTED="$(xcrun simctl list devices available |
        awk -v want="-- iOS ${DEPLOYMENT_TARGET%%.*}" \
            'index($0, "--") == 1 { inside = index($0, want) == 1; next }
             inside && /iPad/ { print }' |
        sed -n '1p' || true)"
    DEVICE="$(printf '%s' "$LISTED" | sed -E 's/.*\(([0-9A-Fa-f-]{36})\).*/\1/')"
    NAME="$(printf '%s' "$LISTED" | sed -E 's/^[[:space:]]*//; s/[[:space:]]*\([0-9A-Fa-f-]{36}\).*//')"
else
    NAME="$DEVICE"
fi
[ -n "$DEVICE" ] ||
    { echo "error: no iPad simulator on iOS $DEPLOYMENT_TARGET is available; install one in Xcode > Settings > Components, or name one in IPAD_SIM" >&2
      exit 1; }

echo "Booting \"$NAME\"..."
# Booting an already-booted simulator is an error rather than a no-op, and a
# second run of this script is the ordinary case.
xcrun simctl boot "$DEVICE" 2>/dev/null || true
xcrun simctl bootstatus "$DEVICE" > /dev/null
xcrun simctl install "$DEVICE" "$BUNDLE"

echo "Launching dev.gropius.chat..."
LOG="$TMP/launch.log"
xcrun simctl launch --console-pty "$DEVICE" dev.gropius.chat > "$LOG" 2>&1 &
LAUNCHED=$!
sleep 8
# Asked while the console is still attached, because detaching it takes the app
# with it. The system lists a running app as a UIKitApplication job; nothing
# there means it crashed on start, which is the one failure a build cannot see
# for itself.
# Read into a variable rather than piped into grep: `set -o pipefail` and a
# grep that stops at the first match make a successful search look like a
# failed pipeline.
JOBS="$(xcrun simctl spawn "$DEVICE" launchctl list 2>/dev/null || true)"
if [ "${JOBS#*UIKitApplication:dev.gropius.chat}" != "$JOBS" ]; then
    echo "Launched on \"$NAME\" and still running after 8 seconds."
    STATUS=0
else
    echo "error: the app did not stay running on \"$NAME\"" >&2
    STATUS=1
fi
kill "$LAUNCHED" 2>/dev/null || true
wait "$LAUNCHED" 2>/dev/null || true
xcrun simctl terminate "$DEVICE" dev.gropius.chat 2>/dev/null || true
[ -s "$LOG" ] && { echo "--- console ---"; cat "$LOG"; }
exit "$STATUS"
