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

.PHONY: all build run test vet clean gaddag gaddag-free download-wordlists defs help \
        android android-arm64-v8a android-x86_64 android-armeabi-v7a android-universal \
        android-release android-release-arm64-v8a android-release-x86_64 \
        android-release-armeabi-v7a android-release-universal \
        android-release-apk android-release-apk-arm64-v8a android-release-apk-x86_64 \
        android-release-apk-armeabi-v7a android-release-apk-universal \
        install-mobile-tools install-desktop

.DEFAULT_GOAL := all

# ── Config ────────────────────────────────────────────────────────────────────

BINARY  := tilewords
CMD     := ./cmd/tilewords
MODULE  := tilewords

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

# The migrated_fynedo build tag opts this binary into Fyne's fyne.Do threading model at
# compile time, so the standalone desktop binary is self-contained: it suppresses the
# launch-time "not migrated" warning without depending on FyneApp.toml being present on
# disk at runtime (plain `go build`/`go run` read that file from the filesystem, they do
# not embed it). The Android build gets the same tag automatically because `fyne release`
# translates FyneApp.toml's [Migrations] fyneDo=true into this tag.
build: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Build the native desktop binary
	go build -tags migrated_fynedo -o $(BINARY) $(CMD)

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
install-desktop: $(GADDAG_ENABLE) $(GADDAG_ASSETS) $(ABOUT_ASSET) ## Install the desktop app + .desktop entry (taskbar icon)
	cp FyneApp.toml $(CMD)/FyneApp.toml
	-cd $(CMD) && fyne install --release --icon $(ICON) --app-id $(APP_ID)
	rm -f $(CMD)/FyneApp.toml

# ── Development ───────────────────────────────────────────────────────────────
##@ Development

test: ## Run all tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

clean: ## Remove build artefacts and generated GADDAG assets
	rm -f $(BINARY) $(BUILDGADDAG_BIN) $(BINARY).apk $(BINARY)-*.apk $(BINARY)-*.apk.idsig
# Release bundles, plus the APK Set intermediate a failed bundletool run can leave.
	rm -f $(BINARY)-release*.aab $(BINARY)-release*.apks
	rm -f $(DICT_DIR)/*.gob $(DICT_DIR)/*.gob.tmp
	rm -f $(DEFS_ASSET) $(DEFS_ASSET).tmp
	rm -f $(ABOUT_ASSET)

# ── Mobile tooling ────────────────────────────────────────────────────────────
##@ Mobile tooling

# NOTE: TileWords's Android build needs a PATCHED fyne CLI (targets SDK 36 and adds v2/v3/v4
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
# Manifest: cmd/tilewords/AndroidManifest.xml is picked up automatically by fyne. It grants
# only local-storage permission for save files and omits INTERNET — TileWords is offline.

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
	"$(APKSIGNER)" $(APKSIGNER_JVM_OPTS) sign --ks "$(DEBUG_KEYSTORE)" --ks-pass pass:$(DEBUG_KEYSTORE_PASS) \
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
JAVA_TOOL_OPTIONS='$(JVM_NATIVE_ACCESS)' \
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

