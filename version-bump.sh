#!/usr/bin/env bash
#
# Bump the application version and build number across every file that carries them.
#
# Three files have to agree, and nothing in the build checks that they do — a mismatch
# surfaces only when the Play Store rejects an upload, or when a packaged app reports a
# version that is not the one that was built. This script is the single place that moves
# them together:
#
#   Makefile                            APP_VERSION := X.Y.Z    APP_BUILD   := N
#   FyneApp.toml                        Version = "X.Y.Z"       Build = N
#   cmd/tilewords/AndroidManifest.xml   android:versionName="X.Y.Z"
#                                       android:versionCode="N"
#
# The Makefile is read for the values in force; run with -h for what the options do.
#
# The semver parsing and comparison below are adapted from semver-tool by François
# Saint-Jacques (MIT licence): https://github.com/fsaintjacques/semver-tool
set -euo pipefail

REPO="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO
readonly MAKEFILE="$REPO/Makefile"
readonly FYNEAPP="$REPO/FyneApp.toml"
readonly MANIFEST="$REPO/cmd/tilewords/AndroidManifest.xml"

readonly PROG="${0##*/}"

DRY_RUN=0
INTERACTIVE=0
OPT_VERSION=""
OPT_BUILD=""

die() { printf 'version-bump: %s\n' "$*" >&2; exit 1; }
note() { printf '%s\n' "$*"; }

# ── semver, after semver-tool ─────────────────────────────────────────────────

readonly NAT='0|[1-9][0-9]*'
readonly ALPHANUM='[0-9]*[A-Za-z-][0-9A-Za-z-]*'
readonly IDENT="$NAT|$ALPHANUM"
readonly FIELD='[0-9A-Za-z-]+'
readonly SEMVER_REGEX="\
^[vV]?\
($NAT)\\.($NAT)\\.($NAT)\
(\\-(${IDENT})(\\.(${IDENT}))*)?\
(\\+${FIELD}(\\.${FIELD})*)?$"

# validate_version VERSION [ARRAY_NAME]
# With an array name, fills it with (major minor patch prerelease build).
# Returns non-zero rather than exiting, so a prompt can ask again.
validate_version() {
    local version=$1
    if [[ "$version" =~ $SEMVER_REGEX ]]; then
        if [ "$#" -eq 2 ]; then
            local major=${BASH_REMATCH[1]}
            local minor=${BASH_REMATCH[2]}
            local patch=${BASH_REMATCH[3]}
            local prere=${BASH_REMATCH[4]}
            local build=${BASH_REMATCH[8]}
            eval "$2=(\"$major\" \"$minor\" \"$patch\" \"$prere\" \"$build\")"
        fi
        return 0
    fi
    return 1
}

is_nat()  { [[ "$1" =~ ^($NAT)$ ]]; }
is_null() { [ -z "$1" ]; }

order_nat()    { [ "$1" -lt "$2" ] && { echo -1; return; }; [ "$1" -gt "$2" ] && { echo 1; return; }; echo 0; }
order_string() { [[ $1 < $2 ]] && { echo -1; return; }; [[ $1 > $2 ]] && { echo 1; return; }; echo 0; }

# compare_fields LEFT_ARRAY_NAME RIGHT_ARRAY_NAME -> -1 | 0 | 1
# Numeric identifiers compare numerically and rank below alphanumeric ones; a field that
# runs out ranks lower. This is the precedence rule from the semver spec.
compare_fields() {
    local l="$1[@]" r="$2[@]"
    local leftfield=( "${!l}" ) rightfield=( "${!r}" )
    local left right
    local i=-1 order=0

    while true; do
        [ "$order" -ne 0 ] && { echo "$order"; return; }
        : $(( i++ ))
        left="${leftfield[$i]:-}"
        right="${rightfield[$i]:-}"

        is_null "$left" && is_null "$right" && { echo 0;  return; }
        is_null "$left"                     && { echo -1; return; }
        is_null "$right"                    && { echo 1;  return; }

        is_nat "$left" && is_nat "$right" && { order=$(order_nat "$left" "$right"); continue; }
        is_nat "$left"                    && { echo -1; return; }
        is_nat "$right"                   && { echo 1;  return; }
        order=$(order_string "$left" "$right")
    done
}

# compare_version A B -> -1 (A<B) | 0 (A==B) | 1 (A>B). Build metadata is ignored, as
# the spec requires: 1.0.0+a and 1.0.0+b are the same version.
compare_version() {
    local order
    local -a V V_
    validate_version "$1" V  || die "not a valid semver version: $1"
    validate_version "$2" V_ || die "not a valid semver version: $2"

    local left=( "${V[0]}" "${V[1]}" "${V[2]}" )
    local right=( "${V_[0]}" "${V_[1]}" "${V_[2]}" )
    order=$(compare_fields left right)
    [ "$order" -ne 0 ] && { echo "$order"; return; }

    # A version with a prerelease ranks below the same version without one.
    local prerel="${V[3]:1}" prerel_="${V_[3]:1}"
    # shellcheck disable=SC2206  # splitting on '.' is intended: each identifier compares separately
    local left=( ${prerel//./ } ) right=( ${prerel_//./ } )

    is_null "$prerel" && is_null "$prerel_" && { echo 0;  return; }
    is_null "$prerel"                       && { echo 1;  return; }
    is_null "$prerel_"                      && { echo -1; return; }

    compare_fields left right
}

# ── reading what is there now ─────────────────────────────────────────────────

current_makefile_version() { sed -n 's/^APP_VERSION[[:space:]]*:=[[:space:]]*\(.*[^[:space:]]\)[[:space:]]*$/\1/p' "$MAKEFILE"; }
current_makefile_build()   { sed -n 's/^APP_BUILD[[:space:]]*:=[[:space:]]*\(.*[^[:space:]]\)[[:space:]]*$/\1/p' "$MAKEFILE"; }
current_fyne_version()     { sed -n 's/^Version[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' "$FYNEAPP"; }
current_fyne_build()       { sed -n 's/^Build[[:space:]]*=[[:space:]]*\([0-9][0-9]*\).*/\1/p' "$FYNEAPP"; }
current_manifest_version() { sed -n 's/.*android:versionName="\([^"]*\)".*/\1/p' "$MANIFEST"; }
current_manifest_build()   { sed -n 's/.*android:versionCode="\([^"]*\)".*/\1/p' "$MANIFEST"; }

# Reads the current values and refuses to go on unless all three files agree. Bumping
# from one file's value while another sits higher would silently downgrade that file.
read_current() {
    local f
    for f in "$MAKEFILE" "$FYNEAPP" "$MANIFEST"; do
        [ -r "$f" ] || die "cannot read $f"
    done

    local mv fv nv mb fb nb
    mv=$(current_makefile_version); fv=$(current_fyne_version); nv=$(current_manifest_version)
    mb=$(current_makefile_build);   fb=$(current_fyne_build);   nb=$(current_manifest_build)

    for pair in "Makefile APP_VERSION:$mv" "FyneApp.toml Version:$fv" "AndroidManifest versionName:$nv" \
                "Makefile APP_BUILD:$mb" "FyneApp.toml Build:$fb" "AndroidManifest versionCode:$nb"; do
        [ -n "${pair#*:}" ] || die "could not read ${pair%%:*} — has its format changed?"
    done

    if [ "$mv" != "$fv" ] || [ "$mv" != "$nv" ]; then
        die "the three files disagree on the version (Makefile=$mv FyneApp.toml=$fv AndroidManifest=$nv).
    Put them in sync by hand first: this script will not guess which one is right."
    fi
    if [ "$mb" != "$fb" ] || [ "$mb" != "$nb" ]; then
        die "the three files disagree on the build (Makefile=$mb FyneApp.toml=$fb AndroidManifest=$nb).
    Put them in sync by hand first: this script will not guess which one is right."
    fi

    CUR_VERSION=$mv
    CUR_BUILD=$mb
}

# ── validating what was asked for ─────────────────────────────────────────────

# Prints nothing and returns 0 when acceptable; prints the reason and returns 1 when not.
check_version() {
    local want=$1
    if ! validate_version "$want"; then
        echo "  '$want' is not a semver version. Expected major.minor.patch, e.g. 0.2.0."
        return 1
    fi
    # A version may hold steady across a rebuild, but it must never go backwards.
    if [ "$(compare_version "$want" "$CUR_VERSION")" -lt 0 ]; then
        echo "  $want is lower than the current $CUR_VERSION. The version must not go backwards."
        return 1
    fi
    return 0
}

check_build() {
    local want=$1
    if ! [[ "$want" =~ ^(0|[1-9][0-9]*)$ ]]; then
        echo "  '$want' is not a build number. Expected a single whole number, e.g. 4."
        return 1
    fi
    # Android refuses an upload whose versionCode is not above the last one, so this is
    # strict where the version itself is not.
    if [ "$want" -le "$CUR_BUILD" ]; then
        echo "  $want is not above the current build $CUR_BUILD. Every release needs a higher build number."
        return 1
    fi
    return 0
}

# ── defaults ──────────────────────────────────────────────────────────────────

# bump_patch VERSION -> VERSION with the patch level raised by one.
# Any prerelease or build metadata is dropped: 0.2.0-rc1 bumps to 0.2.1, not 0.2.1-rc1,
# since a prerelease tag does not carry over to a version it was never attached to.
bump_patch() {
    local -a V
    validate_version "$1" V || die "the current version is not valid semver: $1"
    printf '%s.%s.%s\n' "${V[0]}" "${V[1]}" "$(( V[2] + 1 ))"
}

# ── usage ─────────────────────────────────────────────────────────────────────

# The defaults are shown as the concrete values this working tree would get, so -h
# answers "what will a bare run do here?" and not merely "what does the option mean?".
usage() {
    local cv cb nv nb
    cv=$(current_makefile_version 2>/dev/null) || cv=""
    cb=$(current_makefile_build 2>/dev/null)   || cb=""
    if validate_version "$cv" 2>/dev/null; then nv=$(bump_patch "$cv"); else cv="?"; nv="?"; fi
    if [[ "$cb" =~ ^(0|[1-9][0-9]*)$ ]]; then nb=$(( cb + 1 )); else cb="?"; nb="?"; fi

    cat <<USAGE
Usage: $PROG [-t] [-a version] [-b number]
       $PROG [-t] -i
       $PROG -h

Raise the application version and build number in the three files that carry them, so
that they cannot drift apart:

  Makefile                            APP_VERSION, APP_BUILD
  FyneApp.toml                        Version, Build
  cmd/tilewords/AndroidManifest.xml   android:versionName, android:versionCode

The values in force are read from the Makefile. Given no options, the patch level and
the build number are each raised by one. For this working tree that is:

  version  $cv -> $nv
  build    $cb -> $nb

Options:
  -a, --app-version version   Use this version instead of the default. Semver, in
                              major.minor.patch form (for example 0.2.0).
  -b, --build-number number   Use this build number instead of the default. A single
                              whole number (for example $nb).
  -i, --interactive           Ask for each value that -a and -b did not supply, and
                              keep asking until the answer is usable.
  -t, --test                  Report what would change and write nothing. Combines
                              with -a, -b and -i.
  -h, --help                  Print this message and exit.

A version may repeat when the build number rises, but it may never fall: 0.2.0 will not
follow 0.3.0. The build number must rise every time, because the Play Store refuses an
upload whose versionCode is not above the one before it. The three files must agree on
the values in force before anything is changed; when they do not, the disagreement is
reported and nothing is written.

Examples:
  $PROG                     Raise $cv/$cb to $nv/$nb.
  $PROG -a 0.3.0            Release 0.3.0, with the build number raised to $nb.
  $PROG -b 12               Keep version $cv, set the build number to 12.
  $PROG -t -a 1.0.0         Show the 1.0.0 bump without touching a file.
  $PROG -i                  Be asked for both values.

Exit status:
  0   the files were changed, or with -t would have been
  1   the options were wrong, a value would not move forwards, or the files disagree
USAGE
}

# ── asking ────────────────────────────────────────────────────────────────────

# ask VARNAME OPTION PROMPT CHECK_FN — re-asks until the answer passes. OPTION is the
# flag a caller would otherwise pass, which is what the error must name when there is
# no terminal to ask on.
ask() {
    local varname=$1 option=$2 prompt=$3 check=$4 answer reason
    while true; do
        if ! [ -t 0 ]; then
            die "there is no terminal to ask on.
    Pass the value instead: $PROG $option ..."
        fi
        read -r -p "$prompt" answer
        answer="${answer#"${answer%%[![:space:]]*}"}"   # trim leading space
        answer="${answer%"${answer##*[![:space:]]}"}"   # trim trailing space
        if [ -z "$answer" ]; then
            echo "  Nothing entered."
            continue
        fi
        if reason=$("$check" "$answer"); then
            printf -v "$varname" '%s' "$answer"
            return 0
        fi
        printf '%s\n' "$reason"
    done
}
# ── writing ───────────────────────────────────────────────────────────────────

# edit FILE SED_EXPR EXPECTED_GREP DESCRIPTION
# Edits through a temp file and verifies the result before replacing the original, so a
# pattern that stops matching leaves the tree untouched rather than half-bumped.
edit() {
    local file=$1 expr=$2 expect=$3 what=$4 tmp
    tmp=$(mktemp "$file.XXXXXX")

    if ! sed "$expr" "$file" > "$tmp"; then
        rm -f "$tmp"
        die "$what: sed failed on $file. Nothing has been changed."
    fi
    if ! grep -qF -- "$expect" "$tmp"; then
        rm -f "$tmp"
        die "$what: the edit did not take in $file. Nothing has been changed."
    fi
    if [ "$DRY_RUN" -eq 0 ]; then
        cat "$tmp" > "$file"   # write through, so the file keeps its mode and inode
    fi
    rm -f "$tmp"
}

apply() {
    local version=$1 build=$2

    # Anchored at the start of the line so the prose in the manifest comment, which
    # mentions versionCode and versionName by name, is left alone.
    edit "$MAKEFILE" \
        "s/^\(APP_VERSION[[:space:]]*:=[[:space:]]*\).*/\1$version/" \
        "APP_VERSION" "Makefile APP_VERSION"
    edit "$MAKEFILE" \
        "s/^\(APP_BUILD[[:space:]]*:=[[:space:]]*\).*/\1$build/" \
        "APP_BUILD" "Makefile APP_BUILD"

    edit "$FYNEAPP" \
        "s/^\(Version[[:space:]]*=[[:space:]]*\)\"[^\"]*\"/\1\"$version\"/" \
        "Version = \"$version\"" "FyneApp.toml Version"
    edit "$FYNEAPP" \
        "s/^\(Build[[:space:]]*=[[:space:]]*\)[0-9][0-9]*/\1$build/" \
        "Build = $build" "FyneApp.toml Build"

    edit "$MANIFEST" \
        "s/\(android:versionName=\)\"[^\"]*\"/\1\"$version\"/" \
        "android:versionName=\"$version\"" "AndroidManifest versionName"
    edit "$MANIFEST" \
        "s/\(android:versionCode=\)\"[^\"]*\"/\1\"$build\"/" \
        "android:versionCode=\"$build\"" "AndroidManifest versionCode"
}

verify() {
    local version=$1 build=$2 bad=0
    local got
    for got in "$(current_makefile_version)" "$(current_fyne_version)" "$(current_manifest_version)"; do
        [ "$got" = "$version" ] || { echo "version is '$got', expected '$version'" >&2; bad=1; }
    done
    for got in "$(current_makefile_build)" "$(current_fyne_build)" "$(current_manifest_build)"; do
        [ "$got" = "$build" ] || { echo "build is '$got', expected '$build'" >&2; bad=1; }
    done
    [ "$bad" -eq 0 ] || die "the files did not end up as expected — check them before building."
}

# ── main ──────────────────────────────────────────────────────────────────────

# Long options accept both `--app-version X` and `--app-version=X`; the =-form is split
# here so a single case arm handles each option.
args=()
for arg in "$@"; do
    case "$arg" in
        --*=*) args+=( "${arg%%=*}" "${arg#*=}" ) ;;
        *)     args+=( "$arg" ) ;;
    esac
done
set -- ${args[@]+"${args[@]}"}

while [ $# -gt 0 ]; do
    case "$1" in
        -h|--help)         usage; exit 0 ;;
        -t|--test)         DRY_RUN=1 ;;
        -i|--interactive)  INTERACTIVE=1 ;;
        -a|--app-version)
            [ $# -ge 2 ] || die "$1 needs a version (try --help)"
            OPT_VERSION=$2; shift ;;
        -b|--build-number)
            [ $# -ge 2 ] || die "$1 needs a number (try --help)"
            OPT_BUILD=$2; shift ;;
        --) shift; break ;;
        -*) die "unknown option: $1 (try --help)" ;;
        *)  die "unexpected argument: $1 (try --help)" ;;
    esac
    shift
done
[ $# -eq 0 ] || die "unexpected argument: $1 (try --help)"

read_current
note "Current: version $CUR_VERSION, build $CUR_BUILD"

# Precedence for each value, independently: an explicit -a/-b wins; failing that -i
# asks; failing that the default bump applies. So -i tops up whichever value was not
# given on the command line rather than conflicting with it.
if [ -n "$OPT_VERSION" ]; then
    NEW_VERSION=$OPT_VERSION
    reason=$(check_version "$NEW_VERSION") || die "--app-version $NEW_VERSION is not usable.
$reason"
elif [ "$INTERACTIVE" -eq 1 ]; then
    note ""
    ask NEW_VERSION --app-version \
        "New version (semver, major.minor.patch, e.g. $(bump_patch "$CUR_VERSION"); current $CUR_VERSION): " \
        check_version
else
    NEW_VERSION=$(bump_patch "$CUR_VERSION")
fi

if [ -n "$OPT_BUILD" ]; then
    NEW_BUILD=$OPT_BUILD
    reason=$(check_build "$NEW_BUILD") || die "--build-number $NEW_BUILD is not usable.
$reason"
elif [ "$INTERACTIVE" -eq 1 ]; then
    ask NEW_BUILD --build-number \
        "New build number (a single whole number, e.g. $((CUR_BUILD + 1)); current $CUR_BUILD): " \
        check_build
else
    NEW_BUILD=$(( CUR_BUILD + 1 ))
fi

# Holding the version steady is allowed as long as the build moves; changing neither
# would produce a release indistinguishable from the last one. The default bump moves
# both, so this can only be reached through -a or -b.
if [ "$(compare_version "$NEW_VERSION" "$CUR_VERSION")" -eq 0 ] && [ "$NEW_BUILD" = "$CUR_BUILD" ]; then
    die "version and build are both unchanged — nothing to do."
fi

note ""
note "  version  $CUR_VERSION -> $NEW_VERSION"
note "  build    $CUR_BUILD -> $NEW_BUILD"
note ""

apply "$NEW_VERSION" "$NEW_BUILD"

if [ "$DRY_RUN" -eq 1 ]; then
    note "Test run: the edits above all applied cleanly, but nothing was written."
    exit 0
fi

verify "$NEW_VERSION" "$NEW_BUILD"
note "Updated Makefile, FyneApp.toml and cmd/tilewords/AndroidManifest.xml."
