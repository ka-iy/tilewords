# Squabble — top-level build rules.
#
# Run `make help` to list every target with a one-line description.
#
# Quick start (no licensed word lists required):
#   make gaddag-free && make   → downloads ENABLE (public domain) then builds
#
# Adding a dictionary: drop <name>.txt into wordlists/ (it is compiled to a GADDAG
# asset automatically) and register <name> in dictionary.AllDictNames so the game
# offers it in the new-game setup menu.
#
# First-time mobile setup:
#   make install-mobile-tools     # then set ANDROID_HOME and ANDROID_NDK_HOME

.PHONY: all build run test vet clean gaddag gaddag-free download-wordlists help \
        android android-arm64-v8a android-x86_64 android-armeabi-v7a android-universal \
        android-release android-release-arm64-v8a android-release-x86_64 \
        android-release-armeabi-v7a android-release-universal \
        ios install-mobile-tools install-desktop

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

# Android build-tools (apksigner lives here) — highest installed version. Expanded lazily,
# so it is only resolved when a debug APK is actually re-signed (see DEBUG_KEYSTORE).
ANDROID_BUILD_TOOLS ?= $(ANDROID_HOME)/build-tools/$(shell ls $(ANDROID_HOME)/build-tools 2>/dev/null | sort -V | tail -1)
APKSIGNER            = $(ANDROID_BUILD_TOOLS)/apksigner

# Debug keystore for the android-* debug APKs. If $(DEBUG_KEYSTORE) exists, those targets
# re-sign the APK with it; otherwise the APK keeps the fyne debug key/cert signature. It is
# not committed (see .gitignore). Create one with the conventional Android debug credentials:
#   keytool -genkeypair -v -keystore debug.keystore -storepass android -keypass android \
#     -alias androiddebugkey -dname "CN=Android Debug,O=Android,C=US" \
#     -keyalg RSA -keysize 2048 -validity 10000
DEBUG_KEYSTORE      ?= debug.keystore
DEBUG_KEYSTORE_PASS ?= android
DEBUG_KEY_ALIAS     ?= androiddebugkey

# Release signing (android-release*). Override on the command line. Generate the keystore:
#   keytool -genkey -v -keystore release.keystore \
#           -alias squabble -keyalg RSA -keysize 2048 -validity 10000
KEYSTORE      ?= release.keystore
KEYSTORE_PASS ?= changeme
KEY_ALIAS     ?= squabble

# iOS package output (a .app bundle, i.e. a directory).
IOS_APP := $(BINARY).app

# Mobile app metadata. fyne normally reads these from FyneApp.toml, but mobile builds
# (android/ios) must run from the main-package directory (cmd/squabble), where that
# file is not present — so they are passed explicitly. Keep in sync with FyneApp.toml.
APP_NAME    := Squabble
APP_ID      := net.squabble.app
APP_VERSION := 1.0.0
APP_BUILD   := 1
ICON        := $(CURDIR)/ui/Icon.png

# ── Help ──────────────────────────────────────────────────────────────────────
#
# Self-documenting: a '## text' comment after a target is its description, and a '##@ text'
# comment line starts a new section. Targets and sections are listed in file order.

##@ General
help: ## Show this help (targets grouped by section)
	@echo 'Squabble — make targets:'
	@awk 'BEGIN {FS = ":.*## "} \
		/^##@ / {printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next} \
		/^[a-zA-Z0-9][a-zA-Z0-9_-]*:.*## / {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}' \
		$(MAKEFILE_LIST)

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

gaddag: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Build a GADDAG for every wordlists/*.txt (plus ENABLE)

gaddag-free: $(GADDAG_ENABLE) ## Download ENABLE and build only its GADDAG asset

download-wordlists: $(WL_ENABLE) ## Download the free (ENABLE) word list

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

all: build ## Build the native desktop binary (default target)

build: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Build the native desktop binary
	go build -o $(BINARY) $(CMD)

run: build ## Build and launch the desktop app
	./$(BINARY)

# install-desktop registers the app with the local desktop environment so its icon shows
# in the taskbar/dock. This is separate from the window's own icon: Linux desktops
# (notably GNOME/Wayland) take the taskbar/dock icon from an installed .desktop entry
# matched to the window via StartupWMClass, NOT from the icon the app sets at runtime — so
# a bare `make run` binary shows a generic taskbar icon until the app is installed.
#
# `fyne install` reads the app name from FyneApp.toml, which lives at the repo root rather
# than in $(CMD), so we stage a copy there for the build (removed afterwards even on failure).
install-desktop: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Install the desktop app + .desktop entry (taskbar icon)
	cp FyneApp.toml $(CMD)/FyneApp.toml
	-cd $(CMD) && fyne install --release --icon $(ICON) --app-id $(APP_ID)
	rm -f $(CMD)/FyneApp.toml

test: ## Run all tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

clean: ## Remove build artefacts and generated GADDAG assets
	rm -f $(BINARY) $(BUILDGADDAG_BIN) $(BINARY).apk $(BINARY)-*.apk $(BINARY)-*.apk.idsig
	rm -f $(BINARY)-release*.aab
	rm -rf $(IOS_APP)
	rm -f $(DICT_DIR)/*.gob $(DICT_DIR)/*.gob.tmp

# ── Mobile tooling ────────────────────────────────────────────────────────────

# NOTE: Squabble's Android build needs a PATCHED fyne CLI (targets SDK 36 and adds v2/v3/v4
# signing + zipalign). Install it from the fork instead of the upstream line below, e.g.
#   (cd ~/FYNE-SOURCE/tools && go install ./cmd/fyne)
install-mobile-tools: ## Install the fyne + gomobile CLIs for mobile builds
	go install fyne.io/tools/cmd/fyne@latest
	go install golang.org/x/mobile/cmd/gomobile@latest
	ANDROID_HOME=$(ANDROID_HOME) ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) gomobile init

# ── Android ───────────────────────────────────────────────────────────────────
#
# Prerequisites:
#   • Android SDK platform-tools + NDK installed; ANDROID_HOME / ANDROID_NDK_HOME set.
#   • The (patched) fyne CLI and gomobile in PATH (see install-mobile-tools).
#
# `fyne package` has no -o flag, so each build runs from $(CMD) and writes $(APP_NAME).apk
# there; we move it (and the v4 .idsig sidecar) into place, labelled by ABI. App metadata is
# passed explicitly because FyneApp.toml is not in that directory.
#
# `fyne -os android/<goarch>` restricts a build to one ABI; `-os android` bundles all ABIs
# into one (universal) artifact (~4x the size). The patched fyne signs debug APKs (v1/v2/v3,
# required for targetSdk>=30) and emits an <apk>.idsig (v4); keeping that sidecar next to the
# APK makes `adb install` use the incremental path, which installs cleanly across images.
#
# Manifest: cmd/squabble/AndroidManifest.xml is picked up automatically by fyne. It grants
# only local-storage permission for save files and omits INTERNET — Squabble is offline.

# fyne-package-apk: build a debug APK. $(1)=fyne -os value, $(2)=ABI label for the output.
# The patched fyne signs the APK with its debug key/cert and emits a v4 .idsig. If a local
# $(DEBUG_KEYSTORE) exists, the APK is then re-signed with it (apksigner preserves the
# zipalignment and regenerates the .idsig for the new key); otherwise the fyne signature is
# kept as-is.
define fyne-package-apk
cd $(CMD) && \
ANDROID_HOME=$(ANDROID_HOME) \
ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
fyne package \
	-os $(1) \
	--name $(APP_NAME) \
	--app-id $(APP_ID) \
	--app-version $(APP_VERSION) \
	--app-build $(APP_BUILD) \
	--icon $(ICON)
mv $(CMD)/$(APP_NAME).apk $(BINARY)-$(2).apk
mv $(CMD)/$(APP_NAME).apk.idsig $(BINARY)-$(2).apk.idsig
if [ -f "$(DEBUG_KEYSTORE)" ]; then \
	echo ">> re-signing $(BINARY)-$(2).apk with $(DEBUG_KEYSTORE)"; \
	"$(APKSIGNER)" sign --ks "$(DEBUG_KEYSTORE)" --ks-pass pass:$(DEBUG_KEYSTORE_PASS) \
		--ks-key-alias $(DEBUG_KEY_ALIAS) --key-pass pass:$(DEBUG_KEYSTORE_PASS) \
		--v4-signing-enabled true "$(BINARY)-$(2).apk"; \
else \
	echo ">> $(DEBUG_KEYSTORE) not found — keeping the fyne debug key/cert signature on $(BINARY)-$(2).apk"; \
fi
endef

# fyne-release-aab: build a signed release App Bundle. $(1)=fyne -os value, $(2)=ABI label.
define fyne-release-aab
cd $(CMD) && \
ANDROID_HOME=$(ANDROID_HOME) \
ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
fyne release \
	-os $(1) \
	--name $(APP_NAME) \
	--app-id $(APP_ID) \
	--app-version $(APP_VERSION) \
	--app-build $(APP_BUILD) \
	--icon $(ICON) \
	-keyStore $(CURDIR)/$(KEYSTORE) \
	-keyStorePass $(KEYSTORE_PASS) \
	-keyName $(KEY_ALIAS)
mv $(CMD)/$(APP_NAME).aab $(BINARY)-release-$(2).aab
endef

##@ Android — debug APKs
# Signed with $(DEBUG_KEYSTORE) if present, else the fyne debug key/cert. `adb install`-able.
android-arm64-v8a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Debug APK for arm64-v8a (modern phones)
	$(call fyne-package-apk,android/arm64,arm64-v8a)

android-x86_64: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Debug APK for x86_64 (emulators / x86 devices)
	$(call fyne-package-apk,android/amd64,x86_64)

android-armeabi-v7a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Debug APK for armeabi-v7a (old 32-bit devices)
	$(call fyne-package-apk,android/arm,armeabi-v7a)

android-universal: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Debug APK for all ABIs (universal, ~4x size)
	$(call fyne-package-apk,android,universal)

android: android-arm64-v8a ## Debug APK for arm64-v8a (alias for android-arm64-v8a)

##@ Android — signed release App Bundles (.aab)
# Need KEYSTORE / KEYSTORE_PASS / KEY_ALIAS (see the release-signing config above).
android-release-arm64-v8a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Signed release .aab for arm64-v8a
	$(call fyne-release-aab,android/arm64,arm64-v8a)

android-release-x86_64: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Signed release .aab for x86_64
	$(call fyne-release-aab,android/amd64,x86_64)

android-release-armeabi-v7a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Signed release .aab for armeabi-v7a
	$(call fyne-release-aab,android/arm,armeabi-v7a)

android-release-universal: $(GADDAG_ENABLE) $(GADDAG_ASSETS) ## Signed release .aab for all ABIs (universal)
	$(call fyne-release-aab,android,universal)

android-release: android-release-universal ## Signed release .aab for all ABIs (alias for android-release-universal)

# ── iOS ───────────────────────────────────────────────────────────────────────
#
# macOS only: needs Xcode + command-line tools and a valid Apple developer certificate.
# fyne always generates the iOS Info.plist (it does not read a custom one);
# cmd/squabble/Info.plist documents the intended configuration.
##@ iOS
ios: ## Build an iOS .app bundle (macOS only)
	cd $(CMD) && \
	fyne package \
		-os ios \
		--name $(APP_NAME) \
		--app-id $(APP_ID) \
		--app-version $(APP_VERSION) \
		--app-build $(APP_BUILD) \
		--icon $(ICON)
	mv $(CMD)/$(APP_NAME).app $(IOS_APP)
