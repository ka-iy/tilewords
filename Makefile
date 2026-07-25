# TileWords — top-level build rules.
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

.PHONY: all build build-prod test vet clean clean-all-the-things gaddag gaddag-free download-wordlists defs help \
        debug-all release-all \
        windows-debug windows-release \
        android android-arm64-v8a android-x86_64 android-armeabi-v7a android-universal \
        android-release android-release-arm64-v8a android-release-x86_64 \
        android-release-armeabi-v7a android-release-universal \
        android-release-apk android-release-apk-arm64-v8a android-release-apk-x86_64 \
        android-release-apk-armeabi-v7a android-release-apk-universal \
        install-mobile-tools install-desktop

.DEFAULT_GOAL := all

# ── Infra ─────────────────────────────────────────────────────────────────────

# Get the module name and directory. Note that CURDIR is NOT defined here: it is a make
# built-in holding the absolute working directory, which is what the $(CURDIR) uses below
# need (ICON and the keystore paths are handed to tools that run from another directory, so
# they must be absolute). Assigning it would shadow the built-in and break them.
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
MODNAME := $(shell grep ^module $(MAKEFILE_DIR)go.mod | awk '{print $$2}')

# ----- Build info to embed into the built binary -----
# NOTE: BUILD_TIMESTAMP specifically uses a Unix-style date command formatted
# using standard strftime format specifiers. Building using shell systems
# without support for this (e.g. Windows cmd/powershell) will result in
# undefined behavior. MSYS2 on Windows supports the GNU coreutils date command.
BUILD_TIMESTAMP := $(shell date -u +"%Y-%m-%d_%H:%M:%S_%Z")
BUILD_VERSION := $(shell git describe --tags --long --dirty --always)

# Linker ldflags directives for build info.
# The bits after -X to the left of the "=" refer to variables defined in the 
# Go code. Gomobile does support the standard Go build ldflags directive.
#
# Note that if setting variables NOT in the "main" package, the -X flag
# requires that the package in which the variable lives must be fully
# qualified starting from the main module name. See:
#   https://stackoverflow.com/questions/47509272
#
# None of these carry -s (omit symbol table) or -w (omit DWARF debugging info): fyne adds
# what a release build needs itself, as the GOFLAGS forms below explain.
BUILD_INFO_LDFLAGS_COMMON := -X '$(MODNAME)/buildinfo.buildVersion=$(BUILD_VERSION)' -X '$(MODNAME)/buildinfo.buildTimestamp=$(BUILD_TIMESTAMP)'
BUILD_INFO_LDFLAGS_PROD := $(BUILD_INFO_LDFLAGS_COMMON) -X '$(MODNAME)/buildinfo.buildType=production'
BUILD_INFO_LDFLAGS_DEBUG := $(BUILD_INFO_LDFLAGS_COMMON) -X '$(MODNAME)/buildinfo.buildType=debug'

# GOFLAGS forms of the same directives, which is how every build target here passes them:
# all of them go through the fyne CLI, and it has no --ldflags option — it reads the linker
# flags out of GOFLAGS and forwards them to the build it runs, the Android/gomobile build
# included.
#
# GOFLAGS holds a list of entries, and a quoted run counts as a single entry. The whole
# space-separated '-ldflags=<value>' therefore has to be wrapped in one quoted run, or only
# its first word would be taken as the -ldflags value and the remaining -X directives would
# be parsed as separate (invalid) flags. The wrapping quotes are escaped for the shell, so
# these variables MUST be expanded inside a double-quoted recipe word:
#   GOFLAGS="$(BUILD_INFO_GOFLAGS_DEBUG)"
# The -X values are single words, so the inner quoting of the BUILD_INFO_LDFLAGS_* forms
# survives the round trip: the go command splits an -ldflags value on quotes of its own.
#
# Both forms are built on the variants that carry neither -s nor -w, because fyne supplies
# those itself where they apply:
#   - it appends '-s -w' to a --release desktop build, and '-w' to an Android release;
#   - an Android build cannot use -s at all (its packaging step reads the symbol table, so
#     fyne appends '-s=false' to undo one).
BUILD_INFO_GOFLAGS_DEBUG := \"-ldflags=$(BUILD_INFO_LDFLAGS_DEBUG)\"
BUILD_INFO_GOFLAGS_PROD  := \"-ldflags=$(BUILD_INFO_LDFLAGS_PROD)\"


# ── Config ────────────────────────────────────────────────────────────────────

BINARY  := tilewords
CMD     := ./cmd/tilewords
MODULE  := tilewords

# Host platform, used to label the native desktop artifacts and as the default architecture
# for the Windows cross build. Both come from one `go env` call, which prints one value per
# line — $(shell) turns those into a space-separated pair.
HOST_PLATFORM := $(shell go env GOOS GOARCH)
HOST_GOOS     := $(word 1,$(HOST_PLATFORM))
HOST_GOARCH   := $(word 2,$(HOST_PLATFORM))

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

# Silencing the JDK 24+ JEP 472 warning ("restricted method … System::loadLibrary … by
# org.conscrypt … apksigner.jar"): apksigner's Conscrypt provider loads a native library the
# JVM warns about unless native access is granted. This affects apksigner however it is
# launched — the Makefile's own re-signing call below, and (the usual source of the warning)
# the apksigner that `fyne package`/`fyne release` and `bundletool` spawn internally. The same
# flag is applied in two forms:
#   - APKSIGNER_JVM_OPTS: the apksigner wrapper forwards a leading -J<opt> to the JVM adding
#     one dash, so -J-enable-native-access=ALL-UNNAMED reaches it as
#     --enable-native-access=ALL-UNNAMED. It must precede the apksigner subcommand.
#   - JVM_NATIVE_ACCESS: the plain flag, exported via JAVA_TOOL_OPTIONS to fyne and bundletool
#     (bare `java -jar` wrappers with no -J hook) so the signer they spawn inherits it. The JVM
#     prints one benign "Picked up JAVA_TOOL_OPTIONS: …" line per spawned tool in exchange.
# Override either to empty on a JDK that does not recognise the flag.
APKSIGNER_JVM_OPTS ?= -J-enable-native-access=ALL-UNNAMED
JVM_NATIVE_ACCESS  ?= --enable-native-access=ALL-UNNAMED

# bundletool — required by the android-release* (.aab) and android-release-apk* targets.
# Expected on PATH (e.g. `brew install bundletool`); override if it lives elsewhere.
BUNDLETOOL ?= bundletool

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
#           -alias tilewords -keyalg RSA -keysize 2048 -validity 10000
KEYSTORE      ?= release.keystore
KEYSTORE_PASS ?= changeme
KEY_ALIAS     ?= tilewords

# Mobile app metadata. fyne normally reads these from FyneApp.toml, but Android builds
# must run from the main-package directory (cmd/tilewords), where that file is not
# present — so they are passed explicitly. Keep in sync with FyneApp.toml.
APP_NAME    := TileWords
APP_ID      := fyi.tilewords.game
APP_VERSION := 1.0.0
APP_BUILD   := 1
ICON        := $(CURDIR)/ui/Icon.png

# ── Help ──────────────────────────────────────────────────────────────────────
#
# Self-documenting: a '## text' comment after a target is its description, and a '##@ text'
# comment line starts a new section. Targets and sections are listed in file order.

##@ General
help: ## Show this help (targets grouped by section)
	@echo 'TileWords — make targets:'
	@awk 'BEGIN {FS = ":.*## "} \
		/^##@ / {n++; kind[n]="s"; text[n]=substr($$0, 5); next} \
		/^[a-zA-Z0-9][a-zA-Z0-9_-]*:.*## / {n++; kind[n]="t"; name[n]=$$1; desc[n]=$$2; \
			if (length($$1) > w) w = length($$1)} \
		END {fmt = "  \033[36m%-" w "s\033[0m  %s\n"; \
			for (i = 1; i <= n; i++) \
				if (kind[i] == "s") printf "\n\033[1m%s\033[0m\n", text[i]; \
				else printf fmt, name[i], desc[i]}' \
		$(MAKEFILE_LIST)

# ── Build everything ──────────────────────────────────────────────────────────
#
# One artifact per platform in a single run: the native desktop binary, the Windows .exe and
# the Android package. The Android leg is the arm64-v8a build (the `android`/`android-release`
# aliases), not the universal one, so a run does not pay for four ABIs.
#
# Each leg needs its platform's toolchain — the mingw-w64 cross compiler for Windows, the
# Android SDK/NDK for Android, and a release keystore for release-all. make stops at the
# first leg that fails, leaving the later ones unbuilt, so build a single target directly
# when it is one platform you care about.

debug-all: build windows-debug android ## Build every debug artifact (desktop, Windows, Android)

release-all: build-prod windows-release android-release ## Build every release artifact (desktop, Windows, Android)

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

# ENABLE2K — public domain word list (Alan Beale and others), the 2K edition.
# Downloaded automatically (and decompressed) when its source is absent; every other
# list must be supplied by placing it under wordlists/.
WL_ENABLE_URL := https://raw.githubusercontent.com/BartMassey/wordlists/main/enable2k.txt.gz
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

# Download ENABLE2K from GitHub (a gzip-compressed public-domain distribution) and
# decompress it into the plain-text word list the build consumes.
$(WL_ENABLE): | $(WORDLISTS_DIR)
	curl -fsSL $(WL_ENABLE_URL) | gunzip -c > $@

# Compile any word list into its GADDAG asset. The stem ($*) is the dictionary name.
$(DICT_DIR)/%.gob: $(WORDLISTS_DIR)/%.txt | $(DICT_DIR)
	$(BUILDGADDAG) -input $< -output $@ -name $*

# ── Definitions asset ─────────────────────────────────────────────────────────
#
# The definitions asset holds the word meanings shown during gameplay, filtered
# from a Wiktionary extract down to just the words the shipped lists can form.
# The extract is ~3 GB and is NOT committed; download it once and point
# KAIKKI_EXTRACT at it:
#
#   https://kaikki.org/dictionary/English/kaikki.org-dictionary-English.jsonl
#
# `make defs` is opt-in: the game builds and runs without it (definitions are
# simply unavailable), so `build` does not depend on it.

DEFS_DIR       := defs/assets/definitions
DEFS_ASSET     := $(DEFS_DIR)/definitions.gob.gz
BUILDDEFS      := go run ./tools/builddefs
KAIKKI_EXTRACT ?= wordlists/kaikki-en.jsonl

# comma/space let the space-separated $(WORDLIST_SRCS) be passed as one
# comma-separated -input argument.
comma := ,
empty :=
space := $(empty) $(empty)

$(DEFS_DIR):
	mkdir -p $@

# ── About-dialog text ─────────────────────────────────────────────────────────
#
# The About dialog's text is generated from the top-level ABOUT.txt and LEXICON.txt
# (in that order) so those files stay the single source of truth. Each becomes a
# section headed by its upper-cased file name. The result is written into the ui
# package where it is embedded (a Go //go:embed directive cannot reach a parent
# directory, so the generated copy must live alongside the code that embeds it).
ABOUT_SRCS  := ABOUT.txt FEATURES.txt LEXICON.txt
ABOUT_ASSET := ui/about.txt

$(ABOUT_ASSET): $(ABOUT_SRCS)
	@: > $@
	@for f in $(ABOUT_SRCS); do \
	  name=$$(basename "$$f" .txt | tr '[:lower:]' '[:upper:]'); \
	  { printf '==============================\n  %s\n==============================\n\n' "$$name"; \
	    cat "$$f"; printf '\n\n'; } >> $@; \
	done
	@echo "generated $@ from $(ABOUT_SRCS)"

defs: $(DEFS_ASSET) ## Build the definitions asset from KAIKKI_EXTRACT (Wiktionary extract)

# Rebuild when any word list changes. Fails with guidance if the extract is absent.
$(DEFS_ASSET): $(WORDLIST_SRCS) | $(DEFS_DIR)
	@test -f "$(KAIKKI_EXTRACT)" || { \
	  echo "make defs: Wiktionary extract not found at '$(KAIKKI_EXTRACT)'."; \
	  echo "  Download it from https://kaikki.org/dictionary/English/ and set KAIKKI_EXTRACT=<path>."; \
	  exit 1; }
	$(BUILDDEFS) -kaikki "$(KAIKKI_EXTRACT)" \
	  -input "$(subst $(space),$(comma),$(WORDLIST_SRCS))" -output $@

# ── Definitions augmentation (secondary public-domain sources) ─────────────────
#
# `make defs` builds the base asset from Wiktionary. `make defs-augment` then folds
# in glosses from Webster's 1913 (public domain) and WordNet (permissive) for the
# words Wiktionary does not define, closing part of the coverage gap. It edits
# $(DEFS_ASSET) in place and is idempotent: existing (Wiktionary) definitions always
# win, and only a word that is itself a headword in a source is added, so no
# definition is invented from a near-spelling.
#
# Run it after `make defs` (a fresh `make defs` drops the supplements, so re-run
# this to fold them back in). Both sources are external downloads and NOT committed:
#   - Webster's 1913 JSON:
#       https://raw.githubusercontent.com/matthewreagan/WebstersEnglishDictionary/master/dictionary_compact.json
#   - WordNet 3.1 dict (extract the archive; point WORDNET_DICT at its 'dict' folder):
#       https://wordnetcode.princeton.edu/wn3.1.dict.tar.gz
MERGEDEFS    := go run ./tools/mergedefs
MISSAUDIT    := go run ./tools/missaudit
WEBSTER_JSON ?= wordlists/webster1913.json
WORDNET_DICT ?= wordlists/wordnet/dict
# Committed, reviewable glossary of smaller curated public-domain sources (Jamieson's
# Scots dictionary, Spenser glossaries). See defs/assets/definitions/SOURCES.md.
SUPP_GLOSSARY := defs/supplemental-glossary.tsv

.PHONY: defs-augment
defs-augment: ## Fold Webster's 1913 + WordNet + the supplemental glossary into the defs asset for uncovered words
	@test -f "$(DEFS_ASSET)" || { echo "make defs-augment: $(DEFS_ASSET) not found; run 'make defs' first."; exit 1; }
	@test -f "$(WEBSTER_JSON)" || { \
	  echo "make defs-augment: Webster's 1913 JSON not found at '$(WEBSTER_JSON)'."; \
	  echo "  Download it (see the Makefile comment above defs-augment) and set WEBSTER_JSON=<path>."; \
	  exit 1; }
	@test -d "$(WORDNET_DICT)" || { \
	  echo "make defs-augment: WordNet dict directory not found at '$(WORDNET_DICT)'."; \
	  echo "  Download+extract wn3.1.dict.tar.gz and set WORDNET_DICT=<path-to-dict>."; \
	  exit 1; }
	$(MERGEDEFS) -db $(DEFS_ASSET) \
	  -webster "$(WEBSTER_JSON)" -wordnet "$(WORDNET_DICT)" \
	  -glossary "$(SUPP_GLOSSARY)" \
	  -lists "$(subst $(space),$(comma),$(WORDLIST_SRCS))" -output $(DEFS_ASSET)

.PHONY: defs-audit
defs-audit: ## Report per-list definition coverage and the deduplicated set of undefined words
	@test -f "$(DEFS_ASSET)" || { echo "make defs-audit: $(DEFS_ASSET) not found; run 'make defs' first."; exit 1; }
	$(MISSAUDIT) -db $(DEFS_ASSET) $(WORDLIST_SRCS)

# ── Native desktop ────────────────────────────────────────────────────────────
##@ Native desktop

all: build ## Build the native desktop binary (default target)

# Artifact names. The desktop binaries carry the platform they were built for, and every
# debug artifact ends in -debug, so a debug and a release build (and builds for different
# platforms) sit side by side instead of overwriting each other.
DESKTOP_BIN       := $(BINARY)-$(HOST_GOOS)-$(HOST_GOARCH)
DESKTOP_BIN_DEBUG := $(DESKTOP_BIN)-debug

# These use the fyne CLI, as the Windows and Android targets do, so every artifact in this
# Makefile is produced the same way. `fyne build` compiles a plain executable (unlike
# `fyne package -os linux`, which wraps it in a .tar.xz); it takes the build-info linker
# flags from GOFLAGS, and for --release strips the binary and builds it with -trimpath — so
# the GOFLAGS forms deliberately carry no -s/-w of their own.
#
# -o is absolute because fyne runs the compiler from the source directory, which would
# otherwise be what a relative path is written next to.
#
# The migrated_fynedo build tag opts this binary into Fyne's fyne.Do threading model at
# compile time, so the standalone desktop binary is self-contained: it suppresses the
# launch-time "not migrated" warning without depending on FyneApp.toml being present on
# disk at runtime (plain `go build`/`go run` read that file from the filesystem, they do
# not embed it). It is passed explicitly rather than relying on fyne to translate
# FyneApp.toml's [Migrations] fyneDo=true, which requires that file to be found from the
# main-package directory being built.
build: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Build the native desktop binary (debug)
	GOFLAGS="$(BUILD_INFO_GOFLAGS_DEBUG)" fyne build \
		--src $(CMD) --tags migrated_fynedo -o $(CURDIR)/$(DESKTOP_BIN_DEBUG)

# build-prod differs from build in stamping the binary as a production build and having fyne
# strip it (-s -w) and build it with -trimpath.
#
# Code that branches on buildinfo.IsProductionBuild() only takes its production path in a
# binary built this way, so this is the target to use when testing that behavior.
build-prod: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Build the native desktop binary (production, stripped)
	GOFLAGS="$(BUILD_INFO_GOFLAGS_PROD)" fyne build \
		--src $(CMD) --tags migrated_fynedo --release -o $(CURDIR)/$(DESKTOP_BIN)

# install-desktop registers the app with the local desktop environment so its icon shows
# in the taskbar/dock. This is separate from the window's own icon: Linux desktops
# (notably GNOME/Wayland) take the taskbar/dock icon from an installed .desktop entry
# matched to the window via StartupWMClass, NOT from the icon the app sets at runtime — so
# a bare `make run` binary shows a generic taskbar icon until the app is installed.
#
# `fyne install` reads the app name from FyneApp.toml, which lives at the repo root rather
# than in $(CMD), so we stage a copy there for the build (removed afterwards even on failure).
install-desktop: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Install the desktop app + .desktop entry (taskbar icon)
	cp FyneApp.toml $(CMD)/FyneApp.toml
	-cd $(CMD) && GOFLAGS="$(BUILD_INFO_GOFLAGS_PROD)" fyne install --release --icon $(ICON) --app-id $(APP_ID)
	rm -f $(CMD)/FyneApp.toml

# ── Windows (cross-compiled) ──────────────────────────────────────────────────
##@ Windows (cross-compiled)
#
# Builds a Windows .exe on any host the fyne CLI runs on, Linux included. Fyne's GUI is
# C-backed (GLFW/OpenGL), so this needs cgo and a mingw-w64 cross compiler — but nothing
# beyond it, because the Windows libraries the GUI links (opengl32, gdi32, user32) ship with
# mingw-w64. Install it with, e.g.:
#   sudo apt install gcc-mingw-w64-x86-64       # Debian/Ubuntu
#
# Two things come from packaging with the fyne CLI rather than a bare `go build`: it links
# the binary with -H=windowsgui, so Windows does not open a console window behind the GUI,
# and it embeds $(ICON) as the executable's own icon resource.
#
# `fyne package` has no -o flag, so the build runs from $(CMD) and writes $(APP_NAME).exe
# there; we move it into place. App metadata is passed explicitly because FyneApp.toml is
# not in that directory (as for the Android targets).

# Cross compiler for the Windows target. Override for a different toolchain or a 32-bit
# build, which needs WINDOWS_GOARCH=386 and its own compiler (i686-w64-mingw32-gcc).
WINDOWS_CC ?= x86_64-w64-mingw32-gcc

# Architecture for the Windows build. fyne sets GOOS for the target it is given but leaves
# GOARCH alone, so this is passed explicitly — that keeps the artifact's name honest about
# what is inside it. Must match what $(WINDOWS_CC) emits.
WINDOWS_GOARCH ?= $(HOST_GOARCH)

WINDOWS_BIN       := $(BINARY)-windows-$(WINDOWS_GOARCH).exe
WINDOWS_BIN_DEBUG := $(BINARY)-windows-$(WINDOWS_GOARCH)-debug.exe

# fyne-package-windows: cross-compile a Windows .exe.
#   $(1) = extra `fyne package` flags (--release for a production build, empty otherwise)
#   $(2) = GOFLAGS value carrying the build-info linker flags
#   $(3) = name to give the finished .exe
# The compiler is checked first, so a missing toolchain reports what to install instead of
# failing partway through a cgo build.
define fyne-package-windows
@command -v $(WINDOWS_CC) >/dev/null 2>&1 || { \
	echo "windows build: cross compiler '$(WINDOWS_CC)' not found on PATH."; \
	echo "  Install it (Debian/Ubuntu: sudo apt install gcc-mingw-w64-x86-64),"; \
	echo "  or point WINDOWS_CC at your own toolchain."; \
	exit 1; }
cd $(CMD) && \
CGO_ENABLED=1 \
CC=$(WINDOWS_CC) \
GOARCH=$(WINDOWS_GOARCH) \
GOFLAGS="$(2)" \
fyne package \
	-os windows \
	--name $(APP_NAME) \
	--app-id $(APP_ID) \
	--app-version $(APP_VERSION) \
	--app-build $(APP_BUILD) \
	--icon $(ICON) \
	$(1)
mv $(CMD)/$(APP_NAME).exe $(3)
endef

windows-debug: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Cross-compile a Windows .exe (debug)
	$(call fyne-package-windows,,$(BUILD_INFO_GOFLAGS_DEBUG),$(WINDOWS_BIN_DEBUG))

# --release has fyne strip the binary (-s -w) and build it with -trimpath. It does not sign
# the .exe: signing is what `fyne release -os windows` does, and that wants a Microsoft
# Store developer identity and certificate.
windows-release: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Cross-compile a Windows .exe (production, stripped)
	$(call fyne-package-windows,--release,$(BUILD_INFO_GOFLAGS_PROD),$(WINDOWS_BIN))

# ── Development ───────────────────────────────────────────────────────────────
##@ Development

# These carry the same asset prerequisites as the build targets because the assets are a
# compile-time dependency, not just a runtime one: ui, dictionary and defs reach them with
# //go:embed, and an embed pattern that matches no file fails the build of every package
# that imports them. Without these, `make test` and `make vet` break after
# clean-all-the-things rather than regenerating what they need.
test: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Run all tests with the race detector
	go test -race ./...

vet: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Run go vet
	go vet ./...

# clean removes only what a build produces from source that is already on disk, so
# everything it deletes can be rebuilt with no downloads. The generated assets are left
# alone — see clean-all-the-things for those.
clean: ## Remove built binaries and packages (cheap to rebuild)
	rm -f $(BINARY) $(BUILDGADDAG_BIN) $(BINARY).apk $(BINARY)-*.apk $(BINARY)-*.apk.idsig
# Desktop binaries for this host, plus every Windows .exe (any architecture, either type).
	rm -f $(DESKTOP_BIN) $(DESKTOP_BIN_DEBUG) $(BINARY)-windows-*.exe
# Release bundles, plus the APK Set intermediate a failed bundletool run can leave.
	rm -f $(BINARY)-release*.aab $(BINARY)-release*.apks

# clean-all-the-things additionally drops the generated assets: every GADDAG, the
# definitions asset and the About text. Those are NOT cheap to restore — rebuilding them
# needs the source word lists and the multi-gigabyte Wiktionary extract to still be on disk,
# and anything missing has to be fetched again. Use plain `clean` unless the assets
# themselves are what you need to rebuild.
clean-all-the-things: clean ## Remove the above PLUS every generated asset (needs re-downloads)
	rm -f $(DICT_DIR)/*.gob $(DICT_DIR)/*.gob.tmp
	rm -f $(DEFS_ASSET) $(DEFS_ASSET).tmp
	rm -f $(ABOUT_ASSET)
	@echo ''
	@echo '>> WARNING: every generated asset is now gone. The next build recompiles a GADDAG'
	@echo '>>   for each wordlists/*.txt, and `make defs` rebuilds the definitions asset from'
	@echo '>>   the Wiktionary extract ($(KAIKKI_EXTRACT)). Both need their sources present, so'
	@echo '>>   whatever is no longer on disk must be downloaded again — the extract alone is'
	@echo '>>   several GB. See the GADDAG and definitions sections of this Makefile.'

# ── Mobile tooling ────────────────────────────────────────────────────────────
##@ Mobile tooling

# NOTE: TileWords's Android build needs a PATCHED fyne CLI (targets SDK 36, adds v2/v3/v4
# signing + zipalign, and forwards the GOFLAGS linker flags to the gomobile build so the
# build-info metadata is embedded). Install it from the fork instead of the upstream line
# below, e.g.
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
# there; we move it into place, labelled by ABI. App metadata is passed explicitly because
# FyneApp.toml is not in that directory.
#
# `fyne -os android/<goarch>` restricts a build to one ABI; `-os android` bundles all ABIs
# into one (universal) artifact (~4x the size). The patched fyne signs debug APKs (v1/v2/v3,
# required for targetSdk>=30) and emits an <apk>.idsig (v4); keeping that sidecar next to the
# APK makes `adb install` use the incremental path, which installs cleanly across images.
# Only a fyne carrying that work emits one, so the sidecar is treated as optional: it is
# moved alongside the APK when present, and a stale one is removed when it is not (an .idsig
# that does not match the APK next to it would fail the install it is meant to speed up).
#
# Manifest: cmd/tilewords/AndroidManifest.xml is picked up automatically by fyne. It grants
# only local-storage permission for save files and omits INTERNET — TileWords is offline.
#
# Build info: the Android builds embed the same metadata as the desktop build, passed in
# GOFLAGS (see BUILD_INFO_GOFLAGS_DEBUG / _PROD) because the fyne CLI takes linker flags
# only from there. This does not depend on a patched CLI, which is why GOFLAGS is set
# unconditionally: the whole -ldflags value is one quoted GOFLAGS entry, so a fyne that
# forwards it hands it to the build, and a fyne that ignores GOFLAGS still leaves it in the
# environment for the `go build` that gomobile runs, where the go command applies it itself.
#
# A fyne that consumes GOFLAGS without forwarding it does lose the metadata, and only for
# the release artifact: measured against an upstream CLI, the debug APKs came out with the
# values embedded but the .aab did not, whereas the patched CLI embeds them in both. Losing
# them never fails the build — the app just omits the About dialog's build section, having
# no version to report.
#
# To check an artifact, look for the injected values themselves (e.g. the version string) in
# its lib/*/*.so, NOT for an -ldflags entry in `go version -m` output: the release build
# passes -trimpath, which suppresses that entry whether or not linker flags were applied.

# fyne-package-apk: build a debug APK. $(1)=fyne -os value, $(2)=ABI label for the output.
# The patched fyne signs the APK with its debug key/cert and emits a v4 .idsig. If a local
# $(DEBUG_KEYSTORE) exists, the APK is then re-signed with it (apksigner preserves the
# zipalignment and regenerates the .idsig for the new key); otherwise the fyne signature is
# kept as-is.
define fyne-package-apk
cd $(CMD) && \
ANDROID_HOME=$(ANDROID_HOME) \
ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
JAVA_TOOL_OPTIONS='$(JVM_NATIVE_ACCESS)' \
GOFLAGS="$(BUILD_INFO_GOFLAGS_DEBUG)" \
fyne package \
	-os $(1) \
	--name $(APP_NAME) \
	--app-id $(APP_ID) \
	--app-version $(APP_VERSION) \
	--app-build $(APP_BUILD) \
	--icon $(ICON)
mv $(CMD)/$(APP_NAME).apk $(BINARY)-$(2)-debug.apk
if [ -f "$(CMD)/$(APP_NAME).apk.idsig" ]; then \
	mv $(CMD)/$(APP_NAME).apk.idsig $(BINARY)-$(2)-debug.apk.idsig; \
else \
	echo ">> no v4 signature sidecar emitted — 'adb install' will use its normal path"; \
	rm -f $(BINARY)-$(2)-debug.apk.idsig; \
fi
if [ -f "$(DEBUG_KEYSTORE)" ]; then \
	echo ">> re-signing $(BINARY)-$(2)-debug.apk with $(DEBUG_KEYSTORE)"; \
	"$(APKSIGNER)" $(APKSIGNER_JVM_OPTS) sign --ks "$(DEBUG_KEYSTORE)" --ks-pass pass:$(DEBUG_KEYSTORE_PASS) \
		--ks-key-alias $(DEBUG_KEY_ALIAS) --key-pass pass:$(DEBUG_KEYSTORE_PASS) \
		--v4-signing-enabled true "$(BINARY)-$(2)-debug.apk"; \
else \
	echo ">> $(DEBUG_KEYSTORE) not found — keeping the fyne debug key/cert signature on $(BINARY)-$(2)-debug.apk"; \
fi
endef

# fyne-release-aab: build a signed release App Bundle. $(1)=fyne -os value, $(2)=ABI label.
define fyne-release-aab
cd $(CMD) && \
ANDROID_HOME=$(ANDROID_HOME) \
ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
JAVA_TOOL_OPTIONS='$(JVM_NATIVE_ACCESS)' \
GOFLAGS="$(BUILD_INFO_GOFLAGS_PROD)" \
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

# bundletool-release-apk: convert a signed release .aab into a signed release APK.
# $(1)=ABI label, matching the .aab built by the corresponding android-release-* target.
# --mode=universal emits one APK carrying every ABI present in the bundle, so a per-ABI
# bundle yields that single ABI and the universal bundle yields all of them. bundletool
# writes an APK Set (.apks, a zip holding universal.apk); we extract it and drop the set.
define bundletool-release-apk
JAVA_TOOL_OPTIONS='$(JVM_NATIVE_ACCESS)' $(BUNDLETOOL) build-apks \
	--bundle=$(BINARY)-release-$(1).aab \
	--output=$(BINARY)-release-$(1).apks \
	--mode=universal \
	--overwrite \
	--ks=$(CURDIR)/$(KEYSTORE) \
	--ks-pass=pass:$(KEYSTORE_PASS) \
	--ks-key-alias=$(KEY_ALIAS) \
	--key-pass=pass:$(KEYSTORE_PASS)
unzip -p $(BINARY)-release-$(1).apks universal.apk > $(BINARY)-release-$(1).apk
rm -f $(BINARY)-release-$(1).apks
endef

##@ Android — debug APKs
# Signed with $(DEBUG_KEYSTORE) if present, else the fyne debug key/cert. `adb install`-able.
android-arm64-v8a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Debug APK for arm64-v8a (modern phones)
	$(call fyne-package-apk,android/arm64,arm64-v8a)

android-x86_64: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Debug APK for x86_64 (emulators / x86 devices)
	$(call fyne-package-apk,android/amd64,x86_64)

android-armeabi-v7a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Debug APK for armeabi-v7a (old 32-bit devices)
	$(call fyne-package-apk,android/arm,armeabi-v7a)

android-universal: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Debug APK for all ABIs (universal, ~4x size)
	$(call fyne-package-apk,android,universal)

android: android-arm64-v8a ## Debug APK for arm64-v8a (alias for android-arm64-v8a)

##@ Android — signed release App Bundles (.aab)
# Need KEYSTORE / KEYSTORE_PASS / KEY_ALIAS (see the release-signing config above).
android-release-arm64-v8a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Signed release .aab for arm64-v8a
	$(call fyne-release-aab,android/arm64,arm64-v8a)

android-release-x86_64: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Signed release .aab for x86_64
	$(call fyne-release-aab,android/amd64,x86_64)

android-release-armeabi-v7a: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Signed release .aab for armeabi-v7a
	$(call fyne-release-aab,android/arm,armeabi-v7a)

android-release-universal: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Signed release .aab for all ABIs (universal)
	$(call fyne-release-aab,android,universal)

android-release: android-release-universal ## Signed release .aab for all ABIs (alias for android-release-universal)

# ── Android release APKs ──────────────────────────────────────────────────────
#
# Signed, sideloadable release APKs (for distribution outside Google Play, which wants the
# .aab). fyne cannot emit a release APK directly — its release build is only produced as an
# App Bundle — so each target here first builds the matching .aab via the corresponding
# android-release-* target, then converts that bundle to an APK. The APK is therefore a real
# release build (symbols stripped), not a debug build re-signed.
#
# REQUIRES bundletool on PATH (https://github.com/google/bundletool), e.g.
# `brew install bundletool`; override with BUNDLETOOL=/path/to/bundletool. bundletool is
# needed by the android-release-* .aab targets themselves too, so it is not an extra
# dependency for this section alone.
#
# These targets sign with the release key, so they need KEYSTORE / KEYSTORE_PASS / KEY_ALIAS
# (see the release-signing config above) just like the .aab targets.

##@ Android — release APKs (requires bundletool)
android-release-apk-arm64-v8a: android-release-arm64-v8a ## Signed release APK for arm64-v8a
	$(call bundletool-release-apk,arm64-v8a)

android-release-apk-x86_64: android-release-x86_64 ## Signed release APK for x86_64
	$(call bundletool-release-apk,x86_64)

android-release-apk-armeabi-v7a: android-release-armeabi-v7a ## Signed release APK for armeabi-v7a
	$(call bundletool-release-apk,armeabi-v7a)

android-release-apk-universal: android-release-universal ## Signed release APK for all ABIs (universal)
	$(call bundletool-release-apk,universal)

android-release-apk: android-release-apk-universal ## Signed release APK, all ABIs (alias for android-release-apk-universal)

