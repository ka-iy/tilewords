# Squabble — top-level build rules.
#
# Desktop (Linux/macOS/Windows):
#   make                  → build native binary (uses committed .gob assets)
#   make run              → build and launch
#   make test             → run all tests with race detector
#   make vet              → run go vet
#   make gaddag           → build a GADDAG for every wordlists/*.txt present, plus ENABLE
#   make gaddag-free      → download ENABLE word list and build its GADDAG asset only
#   make clean            → remove build artefacts and generated GADDAG assets
#
# Adding a dictionary: drop <name>.txt into wordlists/ (it is compiled to a GADDAG
# asset automatically) and register <name> in dictionary.AllDictNames so the game
# offers it in the new-game setup menu.
#
# Quick start (no licensed word lists required):
#   make gaddag-free && make   → downloads ENABLE (public domain) then builds
#
# Licensed word lists (not committed — supply your own):
#   Place <name>.txt under wordlists/ and it is compiled automatically by `make gaddag`.
#
# Android (debug APKs are single-ABI; the ABI name is appended to the file name):
#   make android              → arm64-v8a   APK (squabble-arm64-v8a.apk; modern phones)
#   make android-x86_64       → x86_64      APK (squabble-x86_64.apk; emulators / x86)
#   make android-armeabi-v7a  → armeabi-v7a APK (squabble-armeabi-v7a.apk; old 32-bit)
#   make android-release      → signed multi-ABI App Bundle (squabble-release.aab; needs KEYSTORE_*)
#
# iOS  (macOS only):
#   make ios              → build an iOS app bundle   (squabble.app)
#
# First-time mobile setup:
#   make install-mobile-tools
#   # then set ANDROID_HOME and ANDROID_NDK_HOME in your shell

.PHONY: all build run test vet clean gaddag gaddag-free download-wordlists \
        android android-x86_64 android-armeabi-v7a android-release ios \
        install-mobile-tools install-desktop

.DEFAULT_GOAL := all

# ── Config ────────────────────────────────────────────────────────────────────

BINARY  := squabble
CMD     := ./cmd/squabble
MODULE  := squabble

# The GADDAG builder is normally invoked with `go run` (see BUILDGADDAG below), but a
# manual `go build ./tools/buildgaddag` leaves this binary in the repo root; clean it.
BUILDGADDAG_BIN := buildgaddag

# Android SDK/NDK roots — override on the command line or export in your shell.
ANDROID_HOME     ?= $(HOME)/Android/Sdk
ANDROID_NDK_HOME ?= $(ANDROID_HOME)/ndk/$(shell ls $(ANDROID_HOME)/ndk 2>/dev/null | sort -V | tail -1)

# Android build-tools directory (highest installed version). apksigner and zipalign live
# here; the android* debug targets use them to add a v2+ signature to the APK (see the
# "Debug APK signing" note in the Android section).
ANDROID_BUILD_TOOLS ?= $(ANDROID_HOME)/build-tools/$(shell ls $(ANDROID_HOME)/build-tools 2>/dev/null | sort -V | tail -1)
APKSIGNER           := $(ANDROID_BUILD_TOOLS)/apksigner
ZIPALIGN            := $(ANDROID_BUILD_TOOLS)/zipalign

# Debug keystore used to add the mandatory v2 signature to debug APKs. It is generated
# on first use with the conventional Android debug credentials (password "android", alias
# "androiddebugkey"); it is not committed (see .gitignore). Keep the same keystore across
# builds so an updated debug APK installs over a previously installed one.
DEBUG_KEYSTORE      ?= debug.keystore
DEBUG_KEYSTORE_PASS ?= android
DEBUG_KEY_ALIAS     ?= androiddebugkey

# Release signing — override on the command line.
KEYSTORE      ?= release.keystore
KEYSTORE_PASS ?= changeme
KEY_ALIAS     ?= squabble

# Output artefacts. Debug APKs are single-ABI and named $(BINARY)-<abi>.apk by the
# android* targets. A signed Android release is an App Bundle (.aab); an iOS package
# is a .app bundle (a directory).
AAB_RELEASE := $(BINARY)-release.aab
IOS_APP     := $(BINARY).app

# Mobile app metadata. fyne normally reads these from FyneApp.toml, but mobile builds
# (android/ios) must run from the main-package directory (cmd/squabble), where that
# file is not present — so they are passed explicitly. Keep in sync with FyneApp.toml.
APP_NAME    := Squabble
APP_ID      := net.squabble.app
APP_VERSION := 1.0.0
ICON        := $(CURDIR)/ui/Icon.png

# ── GADDAG assets ─────────────────────────────────────────────────────────────
#
# Every word list found at $(WORDLISTS_DIR)/<name>.txt is compiled to a matching
# GADDAG asset at $(DICT_DIR)/<name>.gob by the single pattern rule below. To add a
# new dictionary, just drop its <name>.txt into wordlists/ — it is discovered and
# built automatically, with no new Makefile rule required. (Also register <name> in
# dictionary.AllDictNames so the game offers it in the setup menu.)

WORDLISTS_DIR ?= wordlists
DICT_DIR      := dictionary/assets/dictionaries
BUILDGADDAG   := go run ./tools/buildgaddag

# ENABLE — public domain word list (Alan Beale).  Downloaded automatically when its
# source is absent; every other list must be supplied by placing it under wordlists/.
WL_ENABLE_URL := https://raw.githubusercontent.com/dolph/dictionary/master/enable1.txt
WL_ENABLE     ?= $(WORDLISTS_DIR)/enable.txt

# Discover every word list currently present and map each to its GADDAG asset:
#   wordlists/<name>.txt  →  dictionary/assets/dictionaries/<name>.gob
WORDLIST_SRCS := $(wildcard $(WORDLISTS_DIR)/*.txt)
GADDAG_ASSETS := $(patsubst $(WORDLISTS_DIR)/%.txt,$(DICT_DIR)/%.gob,$(WORDLIST_SRCS))

# ENABLE's asset, named explicitly so gaddag-free/gaddag can request it even before
# its source has been downloaded (it is absent at wildcard-expansion time).
GADDAG_ENABLE := $(DICT_DIR)/enable.gob

# gaddag: build a GADDAG for every word list present, plus ENABLE (downloading it if
# necessary). Licensed lists are built only when you have supplied their source file.
gaddag: $(GADDAG_ENABLE) $(GADDAG_ASSETS)

# gaddag-free: download ENABLE and build its GADDAG asset only.
gaddag-free: $(GADDAG_ENABLE)

# download-wordlists: alias that ensures the free word lists are present.
download-wordlists: $(WL_ENABLE)

# Ensure the source and output directories exist before they are written to.
$(WORDLISTS_DIR) $(DICT_DIR):
	mkdir -p $@

# Download ENABLE from GitHub (public domain mirror of the original list).
$(WL_ENABLE): | $(WORDLISTS_DIR)
	curl -fsSL $(WL_ENABLE_URL) -o $@

# Compile any word list into its GADDAG asset. The stem ($*) is the dictionary name.
$(DICT_DIR)/%.gob: $(WORDLISTS_DIR)/%.txt | $(DICT_DIR)
	$(BUILDGADDAG) -input $< -output $@ -name $*

# ── Desktop ───────────────────────────────────────────────────────────────────

all: build

build: $(GADDAG_ENABLE) $(GADDAG_ASSETS)
	go build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

# install-desktop registers the app with the local desktop environment so its icon shows
# in the taskbar/dock. This is separate from the window's own icon: Linux desktops
# (notably GNOME/Wayland) take the taskbar/dock icon from an installed .desktop entry
# matched to the window via StartupWMClass, NOT from the icon the app sets at runtime — so
# a bare `make run` binary shows a generic taskbar icon until the app is installed.
#
# `fyne install` builds and installs the binary plus a .desktop entry (with the icon) into
# ~/.local/. It reads the app name from FyneApp.toml, which lives at the repo root rather
# than in $(CMD), so we stage a copy there for the build (removed afterwards even on
# failure). --release embeds the metadata so the installed app's window class matches the
# .desktop StartupWMClass.
install-desktop: $(GADDAG_ENABLE) $(GADDAG_ASSETS)
	cp FyneApp.toml $(CMD)/FyneApp.toml
	-cd $(CMD) && fyne install --release --icon $(ICON) --app-id $(APP_ID)
	rm -f $(CMD)/FyneApp.toml

test:
	go test -race ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY) $(BUILDGADDAG_BIN) $(BINARY).apk $(BINARY)-*.apk $(AAB_RELEASE)
	rm -f $(BINARY)-*.apk.idsig $(BINARY)-*.aligned.apk
	rm -rf $(IOS_APP)
	rm -f $(DICT_DIR)/*.gob $(DICT_DIR)/*.gob.tmp

# ── Mobile tooling ────────────────────────────────────────────────────────────

# The UI is built with Fyne, whose `fyne` CLI packages Android and iOS apps
# (driving gomobile under the hood). Run once after cloning on a machine that
# will do mobile builds.
install-mobile-tools:
	go install fyne.io/tools/cmd/fyne@latest
	go install golang.org/x/mobile/cmd/gomobile@latest
	ANDROID_HOME=$(ANDROID_HOME) ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) gomobile init

# ── Android ───────────────────────────────────────────────────────────────────
#
# Prerequisites:
#   • Android Studio / SDK platform-tools installed.
#   • NDK installed via SDK Manager.
#   • ANDROID_HOME and ANDROID_NDK_HOME set (defaults above, or override).
#   • fyne and gomobile in PATH (make install-mobile-tools).
#
# Note: `fyne package` has no -o flag and rejects --src for mobile, so the build runs
# from the main-package dir and writes $(APP_NAME).apk there; we move it into place,
# appending the ABI name. App metadata is passed explicitly because FyneApp.toml is
# not in that directory.
#
# Manifest: cmd/squabble/AndroidManifest.xml is picked up automatically by fyne (it uses a
# custom AndroidManifest.xml when one is present in the main-package dir). It grants only
# local-storage permissions for save files and deliberately omits INTERNET — Squabble is
# offline. iOS has no equivalent (see cmd/squabble/Info.plist and the iOS section below).
#
# Debug APK signing (why the extra zipalign + apksigner steps below):
#   - The fyne packager signs debug APKs with only the v1 (JAR) signature scheme.
#   - Android's installer requires a minimum signature scheme based on the target SDK:
#     targetSdkVersion >= 30 (Android 11+) makes an APK Signature Scheme v2 block MANDATORY,
#     and a v1-only APK is rejected at install with INSTALL_PARSE_FAILED_NO_CERTIFICATES
#     ("No APK Signature Scheme v2 signature in package").
#   - Our manifest sets targetSdkVersion=36, so the fyne-produced v1-only APK will not
#     install. We therefore re-sign it with apksigner (which adds v2/v3) after packaging.
#     zipalign runs first because apksigner requires an already-aligned APK.
#
# Debug APKs, one CPU ABI per target. `fyne -os android/<goarch>` restricts the build
# to a single ABI (the default `-os android` bundles all four ABIs, quadrupling the
# APK size). Each target sets the goarch fyne wants and the Android ABI name used for
# the output file:
#
#   target               goarch  →  Android ABI
#   android              arm64      arm64-v8a     (virtually all modern phones)
#   android-x86_64       amd64      x86_64        (emulators / x86 devices)
#   android-armeabi-v7a  arm        armeabi-v7a   (old 32-bit devices)
android:             FYNE_ARCH := arm64
android:             ABI_LABEL := arm64-v8a
android-x86_64:      FYNE_ARCH := amd64
android-x86_64:      ABI_LABEL := x86_64
android-armeabi-v7a: FYNE_ARCH := arm
android-armeabi-v7a: ABI_LABEL := armeabi-v7a

# Generate the debug keystore on first use (conventional Android debug credentials).
$(DEBUG_KEYSTORE):
	keytool -genkeypair -v -keystore $@ \
		-storepass $(DEBUG_KEYSTORE_PASS) -keypass $(DEBUG_KEYSTORE_PASS) \
		-alias $(DEBUG_KEY_ALIAS) -keyalg RSA -keysize 2048 -validity 10000 \
		-dname "CN=Android Debug,O=Android,C=US"

android android-x86_64 android-armeabi-v7a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(DEBUG_KEYSTORE)
	cd $(CMD) && \
	ANDROID_HOME=$(ANDROID_HOME) \
	ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
	fyne package \
		-os android/$(FYNE_ARCH) \
		--name $(APP_NAME) \
		--app-id $(APP_ID) \
		--app-version $(APP_VERSION) \
		--icon $(ICON)
	mv $(CMD)/$(APP_NAME).apk $(BINARY)-$(ABI_LABEL).apk
	# Re-sign with a v2/v3 signature so targetSdkVersion=36 APKs install (see note above).
	# zipalign must run before apksigner; -p page-aligns the uncompressed native libraries.
	$(ZIPALIGN) -p -f 4 $(BINARY)-$(ABI_LABEL).apk $(BINARY)-$(ABI_LABEL).aligned.apk
	mv $(BINARY)-$(ABI_LABEL).aligned.apk $(BINARY)-$(ABI_LABEL).apk
	$(APKSIGNER) sign \
		--ks $(DEBUG_KEYSTORE) --ks-pass pass:$(DEBUG_KEYSTORE_PASS) \
		--ks-key-alias $(DEBUG_KEY_ALIAS) --key-pass pass:$(DEBUG_KEYSTORE_PASS) \
		$(BINARY)-$(ABI_LABEL).apk
	$(APKSIGNER) verify --verbose $(BINARY)-$(ABI_LABEL).apk

# Signed release App Bundle (.aab).  Generate a keystore first:
#   keytool -genkey -v -keystore release.keystore \
#           -alias squabble -keyalg RSA -keysize 2048 -validity 10000
android-release: $(GADDAG_ENABLE) $(GADDAG_ASSETS)
	cd $(CMD) && \
	ANDROID_HOME=$(ANDROID_HOME) \
	ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
	fyne release \
		-os android \
		--name $(APP_NAME) \
		--app-id $(APP_ID) \
		--app-version $(APP_VERSION) \
		--icon $(ICON) \
		-keyStore $(CURDIR)/$(KEYSTORE) \
		-keyStorePass $(KEYSTORE_PASS) \
		-keyName $(KEY_ALIAS)
	mv $(CMD)/$(APP_NAME).aab $(AAB_RELEASE)

# ── iOS ───────────────────────────────────────────────────────────────────────
#
# Prerequisites:
#   • macOS with Xcode and command-line tools installed.
#   • The fyne CLI in PATH (make install-mobile-tools).
#   • Xcode command-line tools and a valid Apple developer certificate in Keychain.
#
# `fyne package -os ios` writes a $(APP_NAME).app bundle; we move it into place.
#
# Manifest: unlike Android, fyne always generates the iOS Info.plist and does not read a
# custom one. cmd/squabble/Info.plist documents the intended configuration. iOS needs no
# change to satisfy "storage yes, network no": app storage is sandboxed (no permission) and
# iOS has no internet permission to declare — the generated plist requests no networking
# entitlements, matching AndroidManifest.xml's omission of INTERNET.
ios:
	cd $(CMD) && \
	fyne package \
		-os ios \
		--name $(APP_NAME) \
		--app-id $(APP_ID) \
		--app-version $(APP_VERSION) \
		--icon $(ICON)
	mv $(CMD)/$(APP_NAME).app $(IOS_APP)
