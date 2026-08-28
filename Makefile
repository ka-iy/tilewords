# TileWords — top-level build rules.
#
# Run 'make help' to list every target with a one-line description.
#
# Quick start: run 'make' to build the Linux binary. It compiles a GADDAG for each word
# list first, and downloads any shipped list that is missing.
#
# Adding a dictionary: drop <name>.txt into wordlists/ (it is compiled to a GADDAG
# asset automatically) and register <name> in dictionary.AllDictNames so the game
# offers it in the new-game setup menu.
#
# First-time mobile setup:
#   make install-mobile-tools     # then set ANDROID_HOME and ANDROID_NDK_HOME

.PHONY: linux linux-release test vet vet32 clean clean-all-the-things clean-defs-sources \
        gaddag download-wordlists defs defs-asset help \
        debug-all release-all \
        windows-debug windows-release \
        android-debug android-debug-arm64-v8a android-debug-x86_64 \
        android-debug-armeabi-v7a android-debug-universal \
        android-release android-release-arm64-v8a android-release-x86_64 \
        android-release-armeabi-v7a android-release-universal \
        android-release-apk android-release-apk-arm64-v8a android-release-apk-x86_64 \
        android-release-apk-armeabi-v7a android-release-apk-universal \
        install-mobile-tools install-desktop

.DEFAULT_GOAL := linux

# ── Infra ─────────────────────────────────────────────────────────────────────

# Get the module name and directory. Note that CURDIR is NOT defined here: it is a make
# built-in holding the absolute working directory, which is what the $(CURDIR) uses below
# need (ICON and the keystore paths are handed to tools that run from another directory, so
# they must be absolute). Assigning it would shadow the built-in and break them.
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# repoPath resolves a path against this Makefile's own directory, leaving an absolute one
# alone. Every destructive command below goes through it, because a recipe's relative path is
# interpreted against make's WORKING directory, not the Makefile's: 'make -f /path/to/TileWords/
# Makefile clean-defs-sources' run from elsewhere would otherwise delete that directory's
# wordlists/ tree and leave the repo's untouched.
repoPath = $(if $(filter /%,$(1)),$(1),$(MAKEFILE_DIR)$(1))
MODNAME := $(shell grep ^module $(MAKEFILE_DIR)go.mod | awk '{print $$2}')

# ----- Build info to embed into the built binary -----
# NOTE: BUILD_TIMESTAMP specifically uses a Unix-style date command formatted
# using standard strftime format specifiers. Building using shell systems
# without support for this (e.g. Windows cmd/powershell) will result in
# undefined behavior. MSYS2 on Windows supports the GNU coreutils date command.
BUILD_TIMESTAMP := $(shell date -u +"%Y-%m-%d_%H:%M:%S_%Z")
BUILD_VERSION := $(shell git -C $(MAKEFILE_DIR) describe --tags --long --dirty --always 2>/dev/null)

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

# Host architecture, used as the default for the Windows cross build. The GOOS of every build
# target is stated by the target itself, so only the architecture is taken from the host.
HOST_GOARCH := $(shell go env GOARCH)

# The GADDAG builder is normally invoked with 'go run' (see BUILDGADDAG below), but a
# manual 'go build ./tools/buildgaddag' leaves this binary in the repo root; clean it.
BUILDGADDAG_BIN := buildgaddag

# Android SDK/NDK roots — override on the command line or export in your shell.
ANDROID_HOME     ?= $(HOME)/Android/Sdk
ANDROID_NDK_HOME ?= $(ANDROID_HOME)/ndk/$(shell ls $(ANDROID_HOME)/ndk 2>/dev/null | sort -V | tail -1)

# Android build-tools (apksigner lives here) — highest installed version. Expanded lazily,
# so it is only resolved when a debug APK is actually re-signed (see DEBUG_KEYSTORE).
ANDROID_BUILD_TOOLS ?= $(ANDROID_HOME)/build-tools/$(shell ls $(ANDROID_HOME)/build-tools 2>/dev/null | sort -V | tail -1)
APKSIGNER            = $(ANDROID_BUILD_TOOLS)/apksigner

# Silencing two JDK warnings the Android toolchain raises. Both are the JDK tightening access
# to its internals, neither reports anything this project does, and both are suppressed the
# same way: as a JVM flag, either through apksigner's -J hook or exported in JAVA_TOOL_OPTIONS
# for the tools that have no such hook ('fyne' and 'bundletool' are bare 'java -jar' wrappers).
# The JVM prints one benign "Picked up JAVA_TOOL_OPTIONS: …" line per spawned tool in exchange.
#
# Override any of these to empty on a JDK that does not recognise its flag: an unrecognised
# JVM option stops the JVM from starting at all, which is a build failure rather than a warning.

# APKSIGNER_JVM_OPTS silences the JDK 24+ JEP 472 warning ("restricted method …
# System::loadLibrary … by org.conscrypt … apksigner.jar") for the re-signing call below:
# apksigner's Conscrypt provider loads a native library the JVM warns about unless native
# access is granted. The apksigner wrapper forwards a leading -J<opt> to the JVM adding one
# dash, so this reaches it as --enable-native-access=ALL-UNNAMED; it must precede the
# apksigner subcommand.
APKSIGNER_JVM_OPTS ?= -J-enable-native-access=ALL-UNNAMED

# JVM_NATIVE_ACCESS is that same JEP 472 flag in the plain form JAVA_TOOL_OPTIONS takes, for
# the apksigner that 'fyne package', 'fyne release' and 'bundletool' spawn internally — the
# usual source of the warning, since that copy is launched with no -J hook to forward.
JVM_NATIVE_ACCESS ?= --enable-native-access=ALL-UNNAMED

# JVM_UNSAFE_ACCESS silences the JDK 24+ JEP 498 warning ("A terminally deprecated method in
# sun.misc.Unsafe has been called … by com.google.protobuf.UnsafeUtil$MemoryAccessor"), which
# bundletool raises while reading an .aab: the protobuf runtime shaded into bundletool-all.jar
# reaches into arrays through sun.misc.Unsafe. It is protobuf's to fix and ours only to
# silence, so 'allow' restores the pre-JDK-24 quiet until a bundletool ships a protobuf that
# no longer does it. A JDK that has removed those methods will reject 'allow' rather than
# ignore it, which is the point at which this has to go empty and the warning become real.
JVM_UNSAFE_ACCESS ?= --sun-misc-unsafe-memory-access=allow

# JVM_TOOL_OPTS is what the JAVA_TOOL_OPTIONS-based recipes export: both plain flags above, so
# a tool spawned by fyne or bundletool inherits every suppression rather than some of them.
JVM_TOOL_OPTS = $(JVM_NATIVE_ACCESS) $(JVM_UNSAFE_ACCESS)

# bundletool — required by the android-release* (.aab) and android-release-apk* targets.
# Expected on PATH (e.g. 'brew install bundletool'); override if it lives elsewhere.
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

# Release signing (android-release*). Generate the keystore:
#   keytool -genkey -v -keystore release.keystore \
#           -alias tilewords -keyalg RSA -keysize 2048 -validity 10000
#
# Prefer KEYSTORE_PASS_FILE over KEYSTORE_PASS: a password given on the make command line is
# visible in the process list and is kept in shell history, whereas a mode-0600 file is not.
#   make android-release KEYSTORE_PASS_FILE=~/.tilewords-keystore-pass
# When only KEYSTORE_PASS is set, the signing recipes copy it into a temporary 0600 file and
# delete that on exit, so the password still never reaches a tool's argument vector.
KEYSTORE           ?= release.keystore
KEYSTORE_PASS      ?= changeme
KEYSTORE_PASS_FILE ?=
KEY_ALIAS          ?= tilewords

# Mobile app metadata. fyne normally reads these from FyneApp.toml, but Android builds
# must run from the main-package directory (cmd/tilewords), where that file is not
# present — so they are passed explicitly.
#
# APP_VERSION and APP_BUILD are carried by three files, which must agree:
#
#   - this Makefile                       APP_VERSION, APP_BUILD
#   - FyneApp.toml                        Version, Build
#   - cmd/tilewords/AndroidManifest.xml   android:versionName, android:versionCode
#
# Nothing in the build checks that they agree, so an edit to one alone surfaces only when a
# packaged app reports a version it was not built as, or the Play Store rejects an upload.
# Use ./version-bump.sh rather than editing them here: it moves all three together and
# refuses a value that would go backwards. It has to be told what to do -- run with no
# options, it only prints its usage.
#
#   ./version-bump.sh -a           raise the patch level and the build number by one
#   ./version-bump.sh -u           take the version from the most recent tag, patch + 1
#   ./version-bump.sh -v 0.3.0     set the version; the build number still rises
#   ./version-bump.sh -b 12        set the build number; the version holds
#   ./version-bump.sh -i           be asked for the values
#   ./version-bump.sh -d ...       show what would change, write nothing
#   ./version-bump.sh -h           explain all of the above
APP_NAME    := TileWords
APP_ID      := fyi.tilewords.game
APP_VERSION := 0.2.0
APP_BUILD   := 5
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
# the Android package. The Android leg is the arm64-v8a build (the 'android'/'android-release'
# aliases), not the universal one, so a run does not pay for four ABIs.
#
# Each leg needs its platform's toolchain — the mingw-w64 cross compiler for Windows, the
# Android SDK/NDK for Android, and a release keystore for release-all. make stops at the
# first leg that fails, leaving the later ones unbuilt, so build a single target directly
# when it is one platform you care about.

debug-all: linux windows-debug android-debug ## Build every debug artifact (Linux, Windows, Android)

release-all: linux-release windows-release android-release ## Build every release artifact (Linux, Windows, Android)

# ── Assets ────────────────────────────────────────────────────────────────────
##@ Assets
#
# Everything the game embeds, and the sources it is built from. Three assets, each with its
# own subsection below: the GADDAG dictionaries, the definitions asset, and the About text.
#
# The word lists are committed; the definitions asset is not (F-Droid does not accept a
# prebuilt binary asset in a source tree), so a build produces one when it is absent. None of
# these targets has to be run by hand — they exist to fetch sources and to regenerate
# deliberately. Anything a build needs is a prerequisite of the build targets themselves, so
# it is produced on demand.

# ── GADDAG assets ─────────────────────────────────────────────────────────────
#
# Every word list found at $(WORDLISTS_DIR)/<name>.txt is compiled to a matching
# GADDAG asset at $(DICT_DIR)/<name>.bin by the single pattern rule below. To add a
# new dictionary, just drop its <name>.txt into wordlists/ — it is discovered and
# built automatically, with no new Makefile rule required. (Also register <name> in
# dictionary.AllDictNames so the game offers it in the setup menu.)

WORDLISTS_DIR ?= wordlists
DICT_DIR      := dictionary/assets/dictionaries
BUILDGADDAG   := go run ./tools/buildgaddag

# The three shipped word lists, each openly licensed and each fetchable from upstream. All
# three are committed, so these rules normally do nothing: a rule fires only when its list
# is missing, which means a fetch never silently replaces the committed copy. Any other list
# must still be supplied by placing it under wordlists/.
#
# ENABLE2K — public domain (Alan Beale and others), the 2K edition. Distributed gzipped.
WL_ENABLE_URL := https://raw.githubusercontent.com/BartMassey/wordlists/main/enable2k.txt.gz
WL_ENABLE     ?= $(WORDLISTS_DIR)/enable.txt

# Wordnik word list — MIT. Upstream quotes every word, so the quotes are stripped out.
WL_WORDNIK_URL := https://raw.githubusercontent.com/wordnik/wordlist/main/wordlist-20210729.txt
WL_WORDNIK     ?= $(WORDLISTS_DIR)/wordnik.txt

# atebits "Words" — CC0. The list behind Letterpress, taken as-is.
WL_ATEBITS_URL := https://raw.githubusercontent.com/atebits/Words/master/Words/en.txt
WL_ATEBITS     ?= $(WORDLISTS_DIR)/atebits-letterpress.txt

WORDLIST_DOWNLOADS := $(WL_ENABLE) $(WL_WORDNIK) $(WL_ATEBITS)

# Discover every word list currently present and map each to its GADDAG asset:
#   wordlists/<name>.txt  →  dictionary/assets/dictionaries/<name>.bin
WORDLIST_SRCS := $(wildcard $(WORDLISTS_DIR)/*.txt)
GADDAG_ASSETS := $(patsubst $(WORDLISTS_DIR)/%.txt,$(DICT_DIR)/%.bin,$(WORDLIST_SRCS))

# The three shipped lists' assets, named explicitly rather than left to $(GADDAG_ASSETS).
# That wildcard is expanded when make parses this file, so it cannot see a list that is
# still to be downloaded; naming these means one `make` on a checkout missing a list fetches
# it, compiles its GADDAG and builds, instead of quietly producing a binary without it.
# Anything else under wordlists/ is picked up by $(GADDAG_ASSETS) as before.
GADDAG_ENABLE  := $(DICT_DIR)/enable.bin
GADDAG_WORDNIK := $(DICT_DIR)/wordnik.bin
GADDAG_ATEBITS := $(DICT_DIR)/atebits-letterpress.bin

GADDAG_SHIPPED := $(GADDAG_ENABLE) $(GADDAG_WORDNIK) $(GADDAG_ATEBITS)

gaddag: $(GADDAG_SHIPPED) $(GADDAG_ASSETS) ## Build a GADDAG for every wordlists/*.txt (plus the shipped lists)

download-wordlists: $(WORDLIST_DOWNLOADS) ## Download any missing shipped word list (ENABLE, Wordnik, atebits)

# Ensure the source and output directories exist before they are written to.
$(WORDLISTS_DIR) $(DICT_DIR):
	mkdir -p $@

# Each list is fetched, transformed if it needs it, and only then moved into place, so an
# interrupted transfer cannot leave a half a word list that later builds would compile as if
# it were whole. Each step is its own recipe line so make stops on the first one to fail —
# piping curl straight into gunzip or tr would hide a truncated transfer behind the exit
# status of the last command in the pipeline. Every temporary ends in .part, which
# .gitignore covers.
$(WL_ENABLE): | $(WORDLISTS_DIR)
	curl -fsSL -o $@.gz.part $(WL_ENABLE_URL)
	gunzip -c $@.gz.part > $@.part
	mv $@.part $@
	rm -f $@.gz.part

$(WL_WORDNIK): | $(WORDLISTS_DIR)
	curl -fsSL -o $@.raw.part $(WL_WORDNIK_URL)
	tr -d '"' < $@.raw.part > $@.part
	mv $@.part $@
	rm -f $@.raw.part

$(WL_ATEBITS): | $(WORDLISTS_DIR)
	curl -fsSL -o $@.part $(WL_ATEBITS_URL)
	mv $@.part $@

# Words a shipped list carries that no dictionary defines. buildgaddag drops each one
# from every asset it compiles, so the game does not accept as a play a word it could
# only ever show with no meaning. It is a prerequisite of the rule below, so editing it
# recompiles every GADDAG.
EXCLUDE_WORDS := defs/possibly_invalid_words.txt

# Compile any word list into its GADDAG asset. The stem ($*) is the dictionary name.
# $< is the word list alone, so naming $(EXCLUDE_WORDS) second leaves -input unaffected.
$(DICT_DIR)/%.bin: $(WORDLISTS_DIR)/%.txt $(EXCLUDE_WORDS) | $(DICT_DIR)
	$(BUILDGADDAG) -input $< -output $@ -name $* -exclude $(EXCLUDE_WORDS)

# ── Definitions asset ─────────────────────────────────────────────────────────
#
# The definitions asset holds the word meanings shown during gameplay, filtered down to
# just the words the shipped lists can form. 'make defs' builds the complete asset in one
# pass from every source recorded in $(DEFS_DIR)/SOURCES.md, which is the authoritative
# record of what goes in and under which licence:
#   - Wiktionary, via the kaikki.org wiktextract JSONL extract (CC BY-SA 4.0). The primary
#     source: most headwords, and every inflection edge.
#   - Webster's Revised Unabridged Dictionary, 1913 (public domain). Archaic and technical
#     headwords Wiktionary does not define.
#   - Princeton WordNet 3.1 (WordNet licence). Headwords neither source above covers.
#   - $(SUPP_GLOSSARY): curated public-domain glosses (Jamieson's Scots dictionary, a
#     Spenser glossary). Hand-checked and reviewable, so it is committed to the repository
#     rather than scraped from Gutenberg/Wikisource at build time.
#
# The first three are downloaded on demand and are NOT committed. Each is its own rule, so
# a source already present is never re-fetched; point KAIKKI_EXTRACT / WEBSTER_JSON /
# WORDNET_DICT at existing copies to reuse them. The kaikki extract alone is several GB, so
# a build that has to fetch it takes a long while.
#
# Two tools split the work, each parsing the formats it knows: builddefs reads the kaikki
# JSONL into the base asset, then mergedefs folds in the three supplements (JSON, the
# WordNet data files, and TSV respectively). A supplement only ever ADDS a word the primary
# source cannot resolve, and only when the word is itself a headword there, so no gloss is
# inferred from a near-spelling — see the precedence notes in SOURCES.md.
#
# The asset is NOT committed, so a build makes one: every artifact target requires it through
# the defs-asset guard below. The module still compiles without it — the embed pattern matches
# the committed directory, and the game reports that definitions are unavailable — which is
# why 'make test' and 'make vet' do not ask for it. Fetching the sources for a first build
# takes a long while; the kaikki extract alone is several GB.

DEFS_DIR   := defs/assets/definitions
DEFS_ASSET := $(DEFS_DIR)/definitions.bin.gz
BUILDDEFS  := go run ./tools/builddefs
MERGEDEFS  := go run ./tools/mergedefs
MISSAUDIT  := go run ./tools/missaudit

# Staging path for the asset while it is built. Both stages write here and it is moved into
# place only once all of them have succeeded (see the $(DEFS_ASSET) rule).
#
# It deliberately sits BESIDE $(DEFS_DIR) rather than inside it, because defs/loader.go embeds
# that directory with '//go:embed all:', and the 'all:' prefix takes dot-prefixed files too. Both
# asset writers stage next to their output — builddefs writes <output>.tmp, mergedefs its own
# .mergedefs-*.tmp — so staging inside the directory puts those temporaries where the compiler
# collects them. A defs build killed part-way then leaves them behind, 'make defs-asset' only
# tests that the finished asset exists, and the next APK silently ships megabytes of build
# scratch with no error anywhere. Staging one level up keeps the embedded directory holding
# only what is meant to be shipped.
DEFS_STAGE := $(dir $(DEFS_DIR))definitions.bin.gz.stage

# Where a download puts each source when the variables below are left at their defaults.
# clean-defs-sources deletes these paths and only these: they are what this Makefile
# fetched, so they are its to remove. A KAIKKI_EXTRACT / WEBSTER_JSON / WORDNET_DICT
# pointed somewhere else is a copy you supplied, and is never deleted.
DEFS_SRC_KAIKKI  := $(WORDLISTS_DIR)/kaikki-en.jsonl
DEFS_SRC_WEBSTER := $(WORDLISTS_DIR)/webster1913.json
DEFS_SRC_WORDNET := $(WORDLISTS_DIR)/wordnet

# Source locations. Override any of these to point at a copy you already have.
KAIKKI_EXTRACT ?= $(DEFS_SRC_KAIKKI)
WEBSTER_JSON   ?= $(DEFS_SRC_WEBSTER)
WORDNET_DICT   ?= $(DEFS_SRC_WORDNET)/dict

# Committed, reviewable glossary of the smaller curated public-domain sources.
SUPP_GLOSSARY := defs/supplemental-glossary.tsv

KAIKKI_URL  := https://kaikki.org/dictionary/English/kaikki.org-dictionary-English.jsonl
WEBSTER_URL := https://raw.githubusercontent.com/matthewreagan/WebstersEnglishDictionary/master/dictionary_compact.json
WORDNET_URL := https://wordnetcode.princeton.edu/wn3.1.dict.tar.gz

# comma/space let the space-separated $(WORDLIST_SRCS) be passed as one
# comma-separated -input argument.
comma := ,
empty :=
space := $(empty) $(empty)

$(DEFS_DIR):
	mkdir -p $@

# Source downloads. Each writes to a temporary file and moves it into place only on
# success, so an interrupted transfer cannot leave a truncated source that the next build
# would treat as complete and quietly parse.
$(KAIKKI_EXTRACT): | $(WORDLISTS_DIR)
	@echo ">> fetching the Wiktionary (kaikki) extract — several GB, this will take a while"
	curl -fsSL -o $@.part $(KAIKKI_URL) && mv $@.part $@

$(WEBSTER_JSON): | $(WORDLISTS_DIR)
	curl -fsSL -o $@.part $(WEBSTER_URL) && mv $@.part $@

# The archive holds a top-level dict/ directory, so it is extracted into the parent of
# $(WORDNET_DICT) and then moved into place.
#
# Downloading to a temporary file and extracting into a staging directory is what makes this
# rule honour the all-or-nothing promise above. Piping curl straight into tar reports only
# the exit status of the last command in the pipeline, so a truncated transfer looks like a
# successful extraction; and because the target is a directory, an interrupted run leaves a
# partially populated dict/ that make then considers up to date — a later build would fold an
# incomplete WordNet into the shipped asset with no warning. --no-same-owner and
# --no-same-permissions keep the archive's own uid/gid and modes from being applied when the
# build happens to run as root.
$(WORDNET_DICT):
	mkdir -p $(dir $@)
	curl -fsSL -o $(dir $@)wn.tar.gz.part $(WORDNET_URL)
	rm -rf $(dir $@)stage
	mkdir -p $(dir $@)stage
	tar -xzf $(dir $@)wn.tar.gz.part --no-same-owner --no-same-permissions -C $(dir $@)stage
	test -d $(dir $@)stage/dict
	mv $(dir $@)stage/dict $@
	rm -rf $(dir $@)stage $(dir $@)wn.tar.gz.part

defs: $(DEFS_ASSET) ## Build the complete definitions asset, fetching each source if absent

# defs-asset guarantees the asset exists, and does nothing else. It is what the build targets
# depend on, rather than $(DEFS_ASSET) itself.
#
# The asset is not committed, so a build has to be able to produce one; but depending on the
# file directly would pull every build into the asset's own prerequisite graph. After
# clean-defs-sources — the ordinary way to reclaim the several GB the sources occupy — a plain
# 'make linux' would then re-fetch the Wiktionary extract, because a missing prerequisite is
# built before make can decide whether the asset is stale, and the fresh download would then
# be newer than the asset sitting right there. Existence is the condition that matters for a
# build, so that is the only thing tested here.
#
# Rebuilding an asset that already exists is what 'make defs' is for: it depends on the file,
# so it still regenerates when a word list or a source is newer.
defs-asset: ## Build the definitions asset if it is missing (what the build targets use)
	@test -f $(call repoPath,$(DEFS_ASSET)) || $(MAKE) -C $(MAKEFILE_DIR) $(DEFS_ASSET)

# Both stages write to $(DEFS_STAGE), which is moved into place only once every stage has
# succeeded — a failure part-way through must not leave a partial asset that make would
# then consider up to date.
#
# Every source is a real prerequisite, so fetching one is what makes the asset out of date
# and triggers this rule. They are deliberately NOT order-only: order-only prerequisites are
# still built, but do not mark the target stale, which would mean a first run downloading
# gigabytes and then rebuilding nothing.
#
# This rule fires when the asset is absent — a fresh checkout, or after clean-all-the-things —
# and when a source or word list is newer than an asset that exists. A build reaches it only
# through defs-asset, which asks for the file only when it is missing; 'make defs' names the
# file itself, so that is what regenerates a stale one.
$(DEFS_ASSET): $(WORDLIST_SRCS) $(SUPP_GLOSSARY) $(KAIKKI_EXTRACT) $(WEBSTER_JSON) $(WORDNET_DICT) | $(DEFS_DIR)
	$(BUILDDEFS) -kaikki "$(KAIKKI_EXTRACT)" \
	  -input "$(subst $(space),$(comma),$(WORDLIST_SRCS))" -output $(DEFS_STAGE)
	$(MERGEDEFS) -db $(DEFS_STAGE) \
	  -webster "$(WEBSTER_JSON)" -wordnet "$(WORDNET_DICT)" \
	  -glossary "$(SUPP_GLOSSARY)" \
	  -lists "$(subst $(space),$(comma),$(WORDLIST_SRCS))" -output $(DEFS_STAGE)
	mv $(DEFS_STAGE) $@

# ── About-dialog text ─────────────────────────────────────────────────────────
#
# The About dialog's text is generated from the top-level ABOUT.txt, FEATURES.txt and
# LEXICON.txt (in that order) so those files stay the single source of truth. Each becomes
# a section headed by its upper-cased file name. The results are written into the ui
# package where they are embedded (a Go //go:embed directive cannot reach a parent
# directory, so the generated copies must live alongside the code that embeds them).
ABOUT_SRCS  := ABOUT.txt FEATURES.txt LEXICON.txt
ABOUT_ASSET := ui/about.txt

# The copyright and licence notice is generated separately from the sectioned text because
# the dialog shows it above the runtime-composed BUILD INFO section, which the Makefile
# cannot emit; see aboutDialogText.
COPYRIGHT_SRC   := COPYRIGHT.txt
COPYRIGHT_ASSET := ui/copyright.txt

# TEXT_ASSETS is what the build targets depend on: both generated text assets are embedded
# in the ui package, so a build needs each one present.
TEXT_ASSETS := $(COPYRIGHT_ASSET) $(ABOUT_ASSET)

# COMPILE_ASSETS is what any compilation of this module needs, because the packages that
# embed them do not build without them: an embed pattern matching no file is a compile error.
# 'make test' and 'make vet' need exactly this much.
COMPILE_ASSETS := $(GADDAG_SHIPPED) $(GADDAG_ASSETS) $(TEXT_ASSETS)

# BUILD_PREREQS is what producing a shippable artifact needs: the above, plus the definitions
# asset. The definitions asset is deliberately not in COMPILE_ASSETS — the embed pattern
# matches its committed directory, so the module compiles without it and only the game's
# Definitions tab is affected, reporting that definitions are unavailable in this build. That
# is fine for a test run and not fine for something a player installs, so every artifact
# target requires it. See defs-asset for why the guard is used rather than the file itself.
BUILD_PREREQS := $(COMPILE_ASSETS) defs-asset

$(ABOUT_ASSET): $(ABOUT_SRCS)
	@: > $@
	@for f in $(ABOUT_SRCS); do \
	  name=$$(basename "$$f" .txt | tr '[:lower:]' '[:upper:]'); \
	  printf '==============================\n  %s\n==============================\n\n' "$$name" >> $@; \
	  { cat "$$f"; printf '\n\n'; } >> $@; \
	done
	@echo "generated $@ from $(ABOUT_SRCS)"

$(COPYRIGHT_ASSET): $(COPYRIGHT_SRC)
	cp $< $@
	@echo "generated $@ from $(COPYRIGHT_SRC)"

.PHONY: defs-audit
defs-audit: ## Report per-list definition coverage and the deduplicated set of undefined words
	@test -f "$(DEFS_ASSET)" || { echo "make defs-audit: $(DEFS_ASSET) not found; run 'make defs' first."; exit 1; }
	$(MISSAUDIT) -db $(DEFS_ASSET) $(WORDLIST_SRCS)

# clean-defs-sources drops the definition sources this Makefile downloaded, at the default
# locations recorded in DEFS_SRC_* — several GB, the Wiktionary extract being nearly all of
# it. The next 'make defs' fetches them again, so run this to reclaim the space rather than
# as part of an ordinary rebuild. The committed word lists and glossary are untouched: they
# are repository content, not downloads. The generated assets are not touched either — that
# is clean-all-the-things, under Development, which folds this target in.
clean-defs-sources: ## Remove the downloaded definition sources (frees GBs; 'make defs' re-fetches)
	rm -f $(call repoPath,$(DEFS_SRC_KAIKKI)) $(call repoPath,$(DEFS_SRC_KAIKKI)).part
	rm -f $(call repoPath,$(DEFS_SRC_WEBSTER)) $(call repoPath,$(DEFS_SRC_WEBSTER)).part
	rm -rf $(call repoPath,$(DEFS_SRC_WORDNET))

# ── Linux desktop ─────────────────────────────────────────────────────────────
##@ Linux desktop

# Artifact names. The desktop binaries carry the platform they were built for, and every
# debug artifact ends in -debug, so a debug and a release build (and builds for different
# platforms) sit side by side instead of overwriting each other.
# Target platform for these targets. GOOS is pinned because they build for Linux and nothing
# else: on a Windows or macOS host they cross-compile rather than silently producing a binary
# for that host, and every target in this Makefile now states the GOOS it builds for. GOARCH
# defaults to amd64 and is overridable, so an arm64 Linux machine builds for itself with
# 'make LINUX_GOARCH=arm64 linux' — pinning that too would make such a host cross-compile
# unusably.
LINUX_GOOS   := linux
LINUX_GOARCH ?= amd64

DESKTOP_BIN       := $(BINARY)-$(LINUX_GOOS)-$(LINUX_GOARCH)
DESKTOP_BIN_DEBUG := $(DESKTOP_BIN)-debug

# These use the fyne CLI, as the Windows and Android targets do, so every artifact in this
# Makefile is produced the same way. 'fyne build' compiles a plain executable (unlike
# 'fyne package -os linux', which wraps it in a .tar.xz); it takes the build-info linker
# flags from GOFLAGS, and for --release strips the binary and builds it with -trimpath — so
# the GOFLAGS forms deliberately carry no -s/-w of their own.
#
# -o is absolute because fyne runs the compiler from the source directory, which would
# otherwise be what a relative path is written next to.
#
# The migrated_fynedo build tag opts this binary into Fyne's fyne.Do threading model at
# compile time, so the standalone desktop binary is self-contained: it suppresses the
# launch-time "not migrated" warning without depending on FyneApp.toml being present on
# disk at runtime (plain 'go build'/'go run' read that file from the filesystem, they do
# not embed it). It is passed explicitly rather than relying on fyne to translate
# FyneApp.toml's [Migrations] fyneDo=true, which requires that file to be found from the
# main-package directory being built.
linux: $(BUILD_PREREQS) ## Build the Linux desktop binary (debug)
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) GOFLAGS="$(BUILD_INFO_GOFLAGS_DEBUG)" fyne build \
		--src $(CMD) --tags migrated_fynedo -o $(CURDIR)/$(DESKTOP_BIN_DEBUG)

# linux-release differs from linux in stamping the binary as a production build and having fyne
# strip it (-s -w) and build it with -trimpath. It is named to match every other release target
# (windows-release, android-release-*).
#
# Code that branches on buildinfo.IsProductionBuild() only takes its production path in a
# binary built this way, so this is the target to use when testing that behavior.
linux-release: $(BUILD_PREREQS) ## Build the Linux desktop binary (production, stripped)
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) GOFLAGS="$(BUILD_INFO_GOFLAGS_PROD)" fyne build \
		--src $(CMD) --tags migrated_fynedo --release -o $(CURDIR)/$(DESKTOP_BIN)

# install-desktop registers the app with the local desktop environment so its icon shows
# in the taskbar/dock. This is separate from the window's own icon: Linux desktops
# (notably GNOME/Wayland) take the taskbar/dock icon from an installed .desktop entry
# matched to the window via StartupWMClass, NOT from the icon the app sets at runtime — so
# a bare 'make run' binary shows a generic taskbar icon until the app is installed.
#
# 'fyne install' reads the app name from FyneApp.toml, which lives at the repo root rather
# than in $(CMD), so we stage a copy there for the build (removed afterwards even on failure).
install-desktop: $(BUILD_PREREQS) ## Install the Linux desktop app + .desktop entry (taskbar icon)
	cp FyneApp.toml $(CMD)/FyneApp.toml
	-cd $(CMD) && GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) GOFLAGS="$(BUILD_INFO_GOFLAGS_PROD)" fyne install --release --icon $(ICON) --app-id $(APP_ID)
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
# Two things come from packaging with the fyne CLI rather than a bare 'go build': it links
# the binary with -H=windowsgui, so Windows does not open a console window behind the GUI,
# and it embeds $(ICON) as the executable's own icon resource.
#
# 'fyne package' has no -o flag, so the build runs from $(CMD) and writes $(APP_NAME).exe
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
#   $(1) = extra 'fyne package' flags (--release for a production build, empty otherwise)
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
$(clear-staged)
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

windows-debug: $(BUILD_PREREQS) ## Cross-compile a Windows .exe (debug)
	$(call fyne-package-windows,,$(BUILD_INFO_GOFLAGS_DEBUG),$(WINDOWS_BIN_DEBUG))

# --release has fyne strip the binary (-s -w) and build it with -trimpath. It does not sign
# the .exe: signing is what 'fyne release -os windows' does, and that wants a Microsoft
# Store developer identity and certificate.
windows-release: $(BUILD_PREREQS) ## Cross-compile a Windows .exe (production, stripped)
	$(call fyne-package-windows,--release,$(BUILD_INFO_GOFLAGS_PROD),$(WINDOWS_BIN))

# ── Development ───────────────────────────────────────────────────────────────
##@ Development

# These carry the compile-time assets because those are a compile-time dependency, not just a
# runtime one: ui, dictionary and defs reach them with //go:embed, and an embed pattern that
# matches no file fails the build of every package that imports them. Without these, 'make
# test' and 'make vet' break after clean-all-the-things rather than regenerating what they
# need.
#
# They stop short of $(BUILD_PREREQS): the definitions asset is not needed to compile or to
# test, and requiring it would put a multi-gigabyte download in front of a test run on a
# checkout that has not built one. Artifact targets require it; these do not.
test: $(COMPILE_ASSETS) ## Run all tests with the race detector
	go test -race ./...

vet: $(COMPILE_ASSETS) ## Run go vet
	go vet ./...

# The 32-bit ABIs (armeabi-v7a, and any 386 Windows build) have a 32-bit int, so a constant
# or size computation that only fits a 64-bit int is a compile error there and nowhere else.
# An amd64 host never sees it, and the Android targets that would are the slowest to build,
# so type-check the pure-Go packages for a 32-bit arch here instead. The UI packages are
# excluded because they need cgo and an NDK toolchain to build at all.
vet32: ## Type-check the portable packages for 32-bit targets (catches 64-bit-only constants)
	GOOS=android GOARCH=arm go build ./dictionary ./defs ./engine ./cpu ./buildinfo
	GOOS=windows GOARCH=386 go build ./dictionary ./defs ./engine ./cpu ./buildinfo

# FYNE_STAGED are the names the fyne CLI writes into whichever directory it is run from.
# 'fyne package' has no -o flag, so each packaging target builds inside $(CMD) and moves the
# result out afterwards; a run that fails before that move leaves its output sitting there.
# An Android build is the usual casualty, being by far the longest.
#
# All but one are $(APP_NAME)'s own names, and no artifact this Makefile keeps is named that
# way — those are all named for $(BINARY) — so removing one by name cannot reach a finished
# build output. base.zip is the exception: it is the module archive fyne hands to bundletool,
# it carries no app name, and fyne removes it only when the build gets far enough to reach the
# deferred cleanup. No file in this repository is named base.zip.
FYNE_STAGED := $(APP_NAME).apk $(APP_NAME).apk.idsig $(APP_NAME).aab $(APP_NAME).exe base.zip

# clear-staged empties the staging directory before a packaging run.
#
# Neither fyne nor the bundletool it calls will overwrite an artifact that is already there:
# 'bundletool build-bundle' takes no --overwrite flag, so it stops with "File 'TileWords.aab'
# already exists". A run that fails anywhere after the bundle is written — the signing step, or
# make itself stopping before the mv — therefore poisons every later release build until the
# file is removed by hand. Starting from an empty directory makes each packaging target
# repeatable no matter how the one before it ended.
clear-staged = rm -f $(addprefix $(CMD)/,$(FYNE_STAGED))

# clean removes what a build produces from source that is already on disk, plus every
# temporary a failed or interrupted run leaves behind. Nothing it deletes has to be fetched
# again: it never touches a completed download, and the partial ones it does take are never
# resumed anyway (every curl here runs without -C), so the next attempt restarts from zero
# whether or not clean ran. The generated assets are left alone — see clean-all-the-things.
clean: ## Remove built binaries and packages (cheap to rebuild)
	rm -f $(call repoPath,$(BINARY)) $(call repoPath,$(BUILDGADDAG_BIN)) $(call repoPath,$(BINARY)).apk $(call repoPath,$(BINARY))-*.apk $(call repoPath,$(BINARY))-*.apk.idsig
# Desktop binaries for this host, plus every Windows .exe (any architecture, either type).
	rm -f $(call repoPath,$(DESKTOP_BIN)) $(call repoPath,$(DESKTOP_BIN_DEBUG)) $(call repoPath,$(BINARY))-windows-*.exe
# Release bundles, plus the APK Set intermediate a failed bundletool run can leave.
	rm -f $(call repoPath,$(BINARY))-release*.aab $(call repoPath,$(BINARY))-release*.apks
# Packaging leftovers, wherever fyne was run. The whole tree is searched rather than $(CMD)
# alone, so a second main package added later is covered without a change here. Each hit is
# printed: a file disappearing from a source directory should not be silent.
#
# Every sweep below is rooted at $(MAKEFILE_DIR), the directory holding this Makefile, and
# never at '.'. A recursive -delete rooted at the working directory would follow 'make -f
# /path/to/TileWords/Makefile clean' out of the repo and delete matching files under whatever
# directory make happened to be started from — and '.part', swept further down, is the
# in-progress-download suffix used by Firefox and wget. Anchoring to the Makefile keeps clean
# operating on this repo no matter where it is invoked from.
	@for f in $(FYNE_STAGED); do \
		find "$(MAKEFILE_DIR)" -type f -name "$$f" -print -delete; \
	done
# The FyneApp.toml that install-desktop stages into $(CMD) for the length of the build. That
# recipe removes it even when the build itself fails, but not when make is interrupted first.
# The search starts below the repo root so it cannot reach the repo's own copy, which is the
# file being staged rather than a leftover.
	@find "$(MAKEFILE_DIR)" -mindepth 2 -type f -name FyneApp.toml -print -delete
# Staging files the asset tools write beside their output and rename into place on success:
# buildgaddag and builddefs write <output>.tmp, mergedefs its own .mergedefs-*.tmp. Each run
# writes one from scratch rather than continuing an earlier one, so a leftover is dead weight.
	rm -f $(call repoPath,$(DICT_DIR))/*.bin.tmp $(call repoPath,$(DEFS_ASSET)).tmp $(call repoPath,$(DEFS_STAGE)) $(call repoPath,$(DEFS_STAGE)).tmp
	rm -f $(call repoPath,$(dir $(DEFS_DIR)))/.mergedefs-*.tmp $(call repoPath,$(DEFS_DIR))/.mergedefs-*.tmp
# Partial downloads. Every fetch stages through a .part and renames only once complete, so a
# .part is always this Makefile's own temporary wherever it sits, never a source you supplied.
# The whole tree is swept because they are not all in one place: the word lists leave theirs
# in wordlists/, the WordNet fetch leaves one a level further down, and an overridden
# KAIKKI_EXTRACT / WEBSTER_JSON leaves one whereever it points.
	@find "$(MAKEFILE_DIR)" -type f -name '*.part' -print -delete
# Those two overrides can name a path outside the repo, which the sweep above cannot reach.
# Quoted because an override may contain a space, which would otherwise split into two
# unrelated rm targets.
	rm -f "$(call repoPath,$(KAIKKI_EXTRACT)).part" "$(call repoPath,$(WEBSTER_JSON)).part"
# The directory the WordNet fetch untars into before moving dict/ out of it. Its rule clears
# and recreates it on every run, so a leftover is never an input to anything.
	rm -rf $(call repoPath,$(DEFS_SRC_WORDNET))/stage

# clean-all-the-things additionally drops the generated assets — every GADDAG, the
# definitions asset and the About text — and, via clean-defs-sources, the downloaded
# definition sources. Nothing here is cheap to restore: the next 'make defs' re-downloads
# several GB and reprocesses it. Use plain 'clean' unless the assets themselves are what you
# need to rebuild.
clean-all-the-things: clean clean-defs-sources ## Remove the above PLUS every generated asset and downloaded source
	rm -f $(call repoPath,$(DICT_DIR))/*.bin
# The asset itself. The staging files that sit beside it are already gone: clean is a
# prerequisite of this target, and takes them along with every other build temporary.
	rm -f $(call repoPath,$(DEFS_ASSET))
# Partial downloads are already gone, wherever they sat: clean is a prerequisite of this
# target and sweeps them along with every other build temporary.
	rm -f $(foreach a,$(TEXT_ASSETS),$(call repoPath,$(a)))
	@echo ''
	@echo '>> WARNING: every generated asset is now gone, along with the downloaded definition'
	@echo '>>   sources. The next build recompiles a GADDAG for each wordlists/*.txt, and'
	@echo '>>   'make defs' re-downloads the Wiktionary extract, Webster'"'"'s 1913 and WordNet'
	@echo '>>   before reprocessing them — several GB of transfer, the extract being nearly all'
	@echo '>>   of it. A source you supplied yourself (an overridden KAIKKI_EXTRACT,'
	@echo '>>   WEBSTER_JSON or WORDNET_DICT) has been left alone and is reused as-is.'

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
# 'fyne package' has no -o flag, so each build runs from $(CMD) and writes $(APP_NAME).apk
# there; we move it into place, labelled by ABI. App metadata is passed explicitly because
# FyneApp.toml is not in that directory.
#
# 'fyne -os android/<goarch>' restricts a build to one ABI; '-os android' bundles all ABIs
# into one (universal) artifact (~4x the size). The patched fyne signs debug APKs (v1/v2/v3,
# required for targetSdk>=30) and emits an <apk>.idsig (v4); keeping that sidecar next to the
# APK makes 'adb install' use the incremental path, which installs cleanly across images.
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
# environment for the 'go build' that gomobile runs, where the go command applies it itself.
#
# A fyne that consumes GOFLAGS without forwarding it does lose the metadata, and only for
# the release artifact: measured against an upstream CLI, the debug APKs came out with the
# values embedded but the .aab did not, whereas the patched CLI embeds them in both. Losing
# them never fails the build — the app just omits the About dialog's build section, having
# no version to report.
#
# To check an artifact, look for the injected values themselves (e.g. the version string) in
# its lib/*/*.so, NOT for an -ldflags entry in 'go version -m' output: the release build
# passes -trimpath, which suppresses that entry whether or not linker flags were applied.

# fyne-package-apk: build a debug APK. $(1)=fyne -os value, $(2)=ABI label for the output.
# The patched fyne signs the APK with its debug key/cert and emits a v4 .idsig. If a local
# $(DEBUG_KEYSTORE) exists, the APK is then re-signed with it (apksigner preserves the
# zipalignment and regenerates the .idsig for the new key); otherwise the fyne signature is
# kept as-is.
define fyne-package-apk
$(clear-staged)
cd $(CMD) && \
ANDROID_HOME=$(ANDROID_HOME) \
ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
JAVA_TOOL_OPTIONS='$(JVM_TOOL_OPTS)' \
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

# keystore-pass-file emits shell that leaves the release keystore password in a mode-0600
# file named by $$KSPASS. When KEYSTORE_PASS_FILE is set that file is used directly;
# otherwise KEYSTORE_PASS is written to a temporary file which is removed when the shell
# exits, however it exits. Either way the password is never a command-line argument, where
# any other process on the machine could read it from the process list.
define keystore-pass-file
KSPASS='$(KEYSTORE_PASS_FILE)'; \
if [ -z "$$KSPASS" ]; then \
	KSPASS=$$(mktemp) && chmod 600 "$$KSPASS" && \
	trap 'rm -f "$$KSPASS"' EXIT HUP INT TERM && \
	printf '%s' '$(KEYSTORE_PASS)' > "$$KSPASS"; \
elif [ ! -r "$$KSPASS" ]; then \
	echo "KEYSTORE_PASS_FILE=$$KSPASS is not readable" >&2; exit 1; \
fi
endef

# fyne-release-aab: build a signed release App Bundle. $(1)=fyne -os value, $(2)=ABI label.
#
# The recipe is prefixed with @ so make does not echo it: the command line is where the
# password used to end up in terminal scrollback and CI logs. fyne reads -keyStorePass from
# its argument vector only, and documents that it takes the password from stdin when the flag
# is omitted, so the password is piped in rather than passed.
define fyne-release-aab
$(clear-staged)
@$(keystore-pass-file); \
cd $(CMD) && \
ANDROID_HOME=$(ANDROID_HOME) \
ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) \
JAVA_TOOL_OPTIONS='$(JVM_TOOL_OPTS)' \
GOFLAGS="$(BUILD_INFO_GOFLAGS_PROD)" \
fyne release \
	-os $(1) \
	--name $(APP_NAME) \
	--app-id $(APP_ID) \
	--app-version $(APP_VERSION) \
	--app-build $(APP_BUILD) \
	--icon $(ICON) \
	-keyStore $(CURDIR)/$(KEYSTORE) \
	-keyName $(KEY_ALIAS) \
	< "$$KSPASS"
mv $(CMD)/$(APP_NAME).aab $(BINARY)-release-$(2).aab
# A release build writes an intermediate .apk beside the bundle and never removes it, so it
# would be left to collide with the next build (a debug package, or the pre-clear above).
rm -f $(CMD)/$(APP_NAME).apk $(CMD)/base.zip
endef

# bundletool-release-apk: convert a signed release .aab into a signed release APK.
# $(1)=ABI label, matching the .aab built by the corresponding android-release-* target.
# --mode=universal emits one APK carrying every ABI present in the bundle, so a per-ABI
# bundle yields that single ABI and the universal bundle yields all of them. bundletool
# writes an APK Set (.apks, a zip holding universal.apk); we extract it and drop the set.
# The recipe is prefixed with @ so make does not echo the signing command. bundletool accepts
# a password as pass:<literal> or file:<path>; file: is used because a literal would be
# readable in the process list for as long as signing takes.
define bundletool-release-apk
@$(keystore-pass-file); \
JAVA_TOOL_OPTIONS='$(JVM_TOOL_OPTS)' $(BUNDLETOOL) build-apks \
	--bundle=$(BINARY)-release-$(1).aab \
	--output=$(BINARY)-release-$(1).apks \
	--mode=universal \
	--overwrite \
	--ks=$(CURDIR)/$(KEYSTORE) \
	--ks-pass=file:"$$KSPASS" \
	--ks-key-alias=$(KEY_ALIAS) \
	--key-pass=file:"$$KSPASS"
unzip -p $(BINARY)-release-$(1).apks universal.apk > $(BINARY)-release-$(1).apk
rm -f $(BINARY)-release-$(1).apks
endef

##@ Android — debug APKs
# Signed with $(DEBUG_KEYSTORE) if present, else the fyne debug key/cert. 'adb install'-able.
android-debug-arm64-v8a: $(BUILD_PREREQS) ## One debug APK, arm64-v8a only (modern phones)
	$(call fyne-package-apk,android/arm64,arm64-v8a)

android-debug-x86_64: $(BUILD_PREREQS) ## One debug APK, x86_64 only (emulators / x86)
	$(call fyne-package-apk,android/amd64,x86_64)

# vet32 runs first on every target that includes the 32-bit ABI: a constant or size
# computation that only fits a 64-bit int fails to compile for android/arm and nowhere else,
# and catching that in a two-second type-check beats discovering it part-way through an APK
# build that needs the NDK.
android-debug-armeabi-v7a: vet32 $(BUILD_PREREQS) ## One debug APK, armeabi-v7a only (old 32-bit)
	$(call fyne-package-apk,android/arm,armeabi-v7a)

# Universal bundles include android/arm, so they need the 32-bit check too.
android-debug-universal: vet32 $(BUILD_PREREQS) ## One debug APK holding every ABI (~4x the size)
	$(call fyne-package-apk,android,universal)

# android-debug, android-release and android-release-apk each build every ABI. Each per-ABI target
# stays available on its own for a quicker edit-and-test cycle.
#
# Do NOT run these with -j. Every packaging target stages through the same fixed filenames in
# $(CMD) (see FYNE_STAGED), so two builds at once overwrite each other's artifact.
android-debug: android-debug-arm64-v8a android-debug-x86_64 android-debug-armeabi-v7a android-debug-universal ## Runs the 4 targets above: 4 separate APKs

##@ Android — signed release App Bundles (.aab)
# Need KEYSTORE / KEYSTORE_PASS / KEY_ALIAS (see the release-signing config above).
android-release-arm64-v8a: $(BUILD_PREREQS) ## One signed .aab, arm64-v8a only
	$(call fyne-release-aab,android/arm64,arm64-v8a)

android-release-x86_64: $(BUILD_PREREQS) ## One signed .aab, x86_64 only
	$(call fyne-release-aab,android/amd64,x86_64)

android-release-armeabi-v7a: vet32 $(BUILD_PREREQS) ## One signed .aab, armeabi-v7a only
	$(call fyne-release-aab,android/arm,armeabi-v7a)

android-release-universal: vet32 $(BUILD_PREREQS) ## One signed .aab holding every ABI
	$(call fyne-release-aab,android,universal)

android-release: android-release-arm64-v8a android-release-x86_64 android-release-armeabi-v7a android-release-universal ## Runs the 4 targets above: 4 separate .aab files

# ── Android release APKs ──────────────────────────────────────────────────────
#
# Signed, sideloadable release APKs (for distribution outside Google Play, which wants the
# .aab). fyne cannot emit a release APK directly — its release build is only produced as an
# App Bundle — so each target here first builds the matching .aab via the corresponding
# android-release-* target, then converts that bundle to an APK. The APK is therefore a real
# release build (symbols stripped), not a debug build re-signed.
#
# REQUIRES bundletool on PATH (https://github.com/google/bundletool), e.g.
# 'brew install bundletool'; override with BUNDLETOOL=/path/to/bundletool. bundletool is
# needed by the android-release-* .aab targets themselves too, so it is not an extra
# dependency for this section alone.
#
# These targets sign with the release key, so they need KEYSTORE / KEYSTORE_PASS / KEY_ALIAS
# (see the release-signing config above) just like the .aab targets.

##@ Android — release APKs (requires bundletool)
android-release-apk-arm64-v8a: android-release-arm64-v8a ## One signed release APK, arm64-v8a only
	$(call bundletool-release-apk,arm64-v8a)

android-release-apk-x86_64: android-release-x86_64 ## One signed release APK, x86_64 only
	$(call bundletool-release-apk,x86_64)

android-release-apk-armeabi-v7a: android-release-armeabi-v7a ## One signed release APK, armeabi-v7a only
	$(call bundletool-release-apk,armeabi-v7a)

android-release-apk-universal: android-release-universal ## One signed release APK holding every ABI
	$(call bundletool-release-apk,universal)

android-release-apk: android-release-apk-arm64-v8a android-release-apk-x86_64 android-release-apk-armeabi-v7a android-release-apk-universal ## Runs the 4 targets above: 4 separate APKs

