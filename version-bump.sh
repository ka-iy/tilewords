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
# The Makefile is read for the values in force. Something has to be asked for: a run
# with no options prints the usage and does nothing. Run with -h for what the options do.
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
AUTO=0
FROM_TAG=0
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

# Each takes the file to read as $1. They are used both on the real files and on a
# staged rewrite, which is how a rewrite is checked before it goes anywhere near the
# real thing -- the check then uses the very parser the rest of the script trusts.
makefile_version() { sed -n 's/^APP_VERSION[[:space:]]*:=[[:space:]]*\(.*[^[:space:]]\)[[:space:]]*$/\1/p' "$1"; }
makefile_build()   { sed -n 's/^APP_BUILD[[:space:]]*:=[[:space:]]*\(.*[^[:space:]]\)[[:space:]]*$/\1/p' "$1"; }
fyne_version()     { sed -n 's/^Version[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' "$1"; }
fyne_build()       { sed -n 's/^Build[[:space:]]*=[[:space:]]*\([0-9][0-9]*\).*/\1/p' "$1"; }
manifest_version() { sed -n 's/.*android:versionName="\([^"]*\)".*/\1/p' "$1"; }
manifest_build()   { sed -n 's/.*android:versionCode="\([^"]*\)".*/\1/p' "$1"; }

# Reads the current values and refuses to go on unless all three files agree. Bumping
# from one file's value while another sits higher would silently downgrade that file.
read_current() {
    local f
    for f in "$MAKEFILE" "$FYNEAPP" "$MANIFEST"; do
        [ -r "$f" ] || die "cannot read $f"
    done

    local mv fv nv mb fb nb
    mv=$(makefile_version "$MAKEFILE"); fv=$(fyne_version "$FYNEAPP"); nv=$(manifest_version "$MANIFEST")
    mb=$(makefile_build "$MAKEFILE");   fb=$(fyne_build "$FYNEAPP");   nb=$(manifest_build "$MANIFEST")

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


# ── the current tag ───────────────────────────────────────────────────────────

# latest_tag -> the most recent tag reachable from HEAD, as git prints it.
# Returns non-zero instead of exiting when there is no answer -- no git, no repository,
# no tag -- so that the usage text can offer the tag when there is one and say so when
# there is not, while -u itself turns the same failure into an error.
latest_tag() {
    local tag
    command -v git >/dev/null 2>&1 || return 1
    tag=$( cd -- "$REPO" && git describe --tags --abbrev=0 2>/dev/null ) || return 1
    [ -n "$tag" ] || return 1
    printf '%s\n' "$tag"
}


# ── usage ─────────────────────────────────────────────────────────────────────

# The defaults are shown as the concrete values this working tree would get, so -h
# answers "what will this do here?" and not merely "what does the option mean?".
usage() {
    local cv cb nv nb tag tv
    cv=$(makefile_version "$MAKEFILE" 2>/dev/null) || cv=""
    cb=$(makefile_build "$MAKEFILE" 2>/dev/null)   || cb=""
    if validate_version "$cv" 2>/dev/null; then nv=$(bump_patch "$cv"); else cv="?"; nv="?"; fi
    if [[ "$cb" =~ ^(0|[1-9][0-9]*)$ ]]; then nb=$(( cb + 1 )); else cb="?"; nb="?"; fi
    tag=$(latest_tag) || tag=""
    if [ -n "$tag" ] && validate_version "$tag"; then tv=$(bump_patch "$tag"); else tag="none"; tv="?"; fi

    cat <<USAGE
Usage: $PROG [-d] -a
       $PROG [-d] -u [-b number]
       $PROG [-d] [-i] [-v version] [-b number]
       $PROG -h

Raise the application version and build number in the three files that carry them, so
that they cannot drift apart:

  Makefile                            APP_VERSION, APP_BUILD
  FyneApp.toml                        Version, Build
  cmd/tilewords/AndroidManifest.xml   android:versionName, android:versionCode

The values in force are read from the Makefile, and here are version $cv and build $cb.
Something has to be asked for: a run with no options prints this message and changes
nothing.

Options:
  -a, --auto                  Raise the patch level and the build number by one; here,
                              version $cv -> $nv and build $cb -> $nb. This is the whole
                              instruction, so it takes no other option but -d.
  -u, --update-from-tag       Take the base version from the most recent tag reachable
                              from HEAD rather than from the Makefile, and raise its
                              patch level by one; here the tag is $tag, so version
                              -> $tv. The build number rises by one, to $nb. It is an
                              error if no tag can be read.
  -v, --app-version version   Use this version. Semver, in major.minor.patch form (for
                              example 0.2.0). The build number still rises, to $nb.
  -b, --build-number number   Use this build number: a single whole number, for example
                              $nb. On its own it holds the version at $cv, that being a
                              rebuild of the version in force rather than a new one.
  -i, --interactive           Ask for each value the options above did not supply, and
                              keep asking until the answer is usable.
  -d, --dry-run               Report what would change and write nothing. Combines with
                              every option above.
  -h, --help                  Print this message and exit.

A version may repeat when the build number rises, but it may never fall: 0.2.0 will not
follow 0.3.0, and neither will a version taken from an old tag. The build number must
rise every time, because the Play Store refuses an upload whose versionCode is not above
the one before it. The three files must agree on the values in force before anything is
changed; when they do not, the disagreement is reported and nothing is written.

Examples:
  $PROG -a                  Raise $cv/$cb to $nv/$nb.
  $PROG -u                  Follow the tag $tag: version $tv, build $nb.
  $PROG -v 0.3.0            Release 0.3.0, with the build number raised to $nb.
  $PROG -b 12               Keep version $cv, set the build number to 12.
  $PROG -d -v 1.0.0         Show the 1.0.0 bump without touching a file.
  $PROG -i                  Be asked for both values.

Exit status:
  0   the files were changed, or with -d would have been
  1   nothing was asked for, an option was wrong, a value would not move forwards, the
      tag could not be read, or the files disagree
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

# Temp files holding the rewritten content, and the file each is destined for. The two
# arrays are parallel: STAGED[i] is the new content of STAGED_FOR[i].
STAGED=()
STAGED_FOR=()

trap 'rm -f ${STAGED[@]+"${STAGED[@]}"}' EXIT

# stage FILE LABEL VERSION BUILD VERSION_READER BUILD_READER SED_VERSION SED_BUILD
# Rewrites FILE into a temp file and confirms, by reading that temp back with the same
# accessor that reads the real file, that both values landed. FILE itself is not
# touched. Staging every file before committing any of them is what keeps a failure
# from leaving some files bumped and others not -- which is the very drift this script
# exists to prevent, and which no part of the build would notice.
stage() {
    local file=$1 label=$2 version=$3 build=$4 vread=$5 bread=$6 sedv=$7 sedb=$8
    local tmp got

    tmp=$(mktemp "$file.XXXXXX")
    STAGED+=( "$tmp" )
    STAGED_FOR+=( "$file" )

    sed -e "$sedv" -e "$sedb" "$file" > "$tmp" \
        || die "$label: sed failed. No file has been changed."

    got=$("$vread" "$tmp")
    [ "$got" = "$version" ] \
        || die "$label: the version did not take (found '$got', wanted '$version'). No file has been changed."
    got=$("$bread" "$tmp")
    [ "$got" = "$build" ] \
        || die "$label: the build did not take (found '$got', wanted '$build'). No file has been changed."
}

# stage_all leaves the tree untouched, so it is also exactly what -t needs to run.
stage_all() {
    local version=$1 build=$2

    # The Makefile and FyneApp expressions are anchored at the start of the line so that
    # only the assignments match. The manifest's are not, its attributes being indented,
    # but they carry the android: prefix, which the prose in that file's comment does not.
    stage "$MAKEFILE" "Makefile" "$version" "$build" \
        makefile_version makefile_build \
        "s/^\(APP_VERSION[[:space:]]*:=[[:space:]]*\).*/\1$version/" \
        "s/^\(APP_BUILD[[:space:]]*:=[[:space:]]*\).*/\1$build/"

    stage "$FYNEAPP" "FyneApp.toml" "$version" "$build" \
        fyne_version fyne_build \
        "s/^\(Version[[:space:]]*=[[:space:]]*\)\"[^\"]*\"/\1\"$version\"/" \
        "s/^\(Build[[:space:]]*=[[:space:]]*\)[0-9][0-9]*/\1$build/"

    stage "$MANIFEST" "AndroidManifest.xml" "$version" "$build" \
        manifest_version manifest_build \
        "s/\(android:versionName=\)\"[^\"]*\"/\1\"$version\"/" \
        "s/\(android:versionCode=\)\"[^\"]*\"/\1\"$build\"/"
}

commit_staged() {
    local i
    # Every destination is checked writable before any of them is written, so the one
    # failure that staging cannot catch -- a file that is read-only -- is still caught
    # while the tree is whole. Only an error part-way through a write can now split it,
    # and that is what the message below is for.
    for i in "${!STAGED_FOR[@]}"; do
        [ -w "${STAGED_FOR[$i]}" ] || die "${STAGED_FOR[$i]} is not writable. No file has been changed."
    done

    for i in "${!STAGED[@]}"; do
        # Written through rather than moved, so each file keeps its mode and its inode.
        cat "${STAGED[$i]}" > "${STAGED_FOR[$i]}" \
            || die "could not write ${STAGED_FOR[$i]} — the files may now disagree; check them before building."
    done
}

# Reads the real files back after the write, so what the build will see is what was asked
# for, not merely what was staged.
verify() {
    local version=$1 build=$2 bad=0
    local got
    for got in "$(makefile_version "$MAKEFILE")" "$(fyne_version "$FYNEAPP")" "$(manifest_version "$MANIFEST")"; do
        [ "$got" = "$version" ] || { echo "version is '$got', expected '$version'" >&2; bad=1; }
    done
    for got in "$(makefile_build "$MAKEFILE")" "$(fyne_build "$FYNEAPP")" "$(manifest_build "$MANIFEST")"; do
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
        -h|--help)             usage; exit 0 ;;
        -d|--dry-run)          DRY_RUN=1 ;;
        -i|--interactive)      INTERACTIVE=1 ;;
        -a|--auto)             AUTO=1 ;;
        -u|--update-from-tag)  FROM_TAG=1 ;;
        -v|--app-version)
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

# Nothing was asked for. -d on its own lands here too, being a dry run of nothing. The
# usage goes to standard error and the status is non-zero, so that a caller that meant
# to bump something cannot read the run as a bump that happened.
if [ "$AUTO" -eq 0 ] && [ "$FROM_TAG" -eq 0 ] && [ "$INTERACTIVE" -eq 0 ] \
   && [ -z "$OPT_VERSION" ] && [ -z "$OPT_BUILD" ]; then
    usage >&2
    exit 1
fi

# -a fixes both values by itself, so every option that would fix either of them
# contradicts it, and -u and -v contradict each other for the same reason. Refusing the
# combination is the only honest answer: silently letting one win would bump to a value
# the command line also asked not to use.
if [ "$AUTO" -eq 1 ]; then
    [ "$FROM_TAG" -eq 0 ]    || die "-a and -u each choose the new version (try --help)"
    [ -z "$OPT_VERSION" ]    || die "-a and -v each choose the new version (try --help)"
    [ -z "$OPT_BUILD" ]      || die "-a and -b each choose the new build number (try --help)"
    [ "$INTERACTIVE" -eq 0 ] || die "-a supplies both values, leaving -i nothing to ask (try --help)"
fi
if [ "$FROM_TAG" -eq 1 ] && [ -n "$OPT_VERSION" ]; then
    die "-u and -v each choose the new version (try --help)"
fi

read_current
note "Current: version $CUR_VERSION, build $CUR_BUILD"

# Precedence for each value: an explicit -v/-b wins; failing that -u derives the version
# from the tag; failing that -i asks; failing that a default applies. So -i tops up
# whichever value was not given on the command line rather than conflicting with it.
#
# The version's default is to hold, which is what -b alone leaves. Pinning the build
# alone says "this build of the version in force" -- a rebuild or a re-upload -- and
# moving the version underneath that would be the opposite of what was asked for.
if [ -n "$OPT_VERSION" ]; then
    NEW_VERSION=$OPT_VERSION
    reason=$(check_version "$NEW_VERSION") || die "--app-version $NEW_VERSION is not usable.
$reason"
elif [ "$FROM_TAG" -eq 1 ]; then
    TAG=$(latest_tag) || die "no tag to bump from: git describe found none reachable from HEAD.
    Tag the commit first, or pass the version instead: $PROG --app-version ..."
    validate_version "$TAG" || die "the most recent tag, $TAG, is not a semver version.
    Pass the version instead: $PROG --app-version ..."
    NEW_VERSION=$(bump_patch "$TAG")
    note "Most recent tag: $TAG"
    reason=$(check_version "$NEW_VERSION") || die "--update-from-tag gives $NEW_VERSION, from the tag $TAG, which is not usable.
$reason"
elif [ "$INTERACTIVE" -eq 1 ]; then
    note ""
    ask NEW_VERSION --app-version \
        "New version (semver, major.minor.patch, e.g. $(bump_patch "$CUR_VERSION"); current $CUR_VERSION): " \
        check_version
elif [ "$AUTO" -eq 1 ]; then
    NEW_VERSION=$(bump_patch "$CUR_VERSION")
else
    NEW_VERSION=$CUR_VERSION
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

note ""
note "  version  $CUR_VERSION -> $NEW_VERSION"
note "  build    $CUR_BUILD -> $NEW_BUILD"
note ""

stage_all "$NEW_VERSION" "$NEW_BUILD"

if [ "$DRY_RUN" -eq 1 ]; then
    note "Dry run: every file rewrote cleanly, but nothing was written."
    exit 0
fi

commit_staged
verify "$NEW_VERSION" "$NEW_BUILD"
note "Updated Makefile, FyneApp.toml and cmd/tilewords/AndroidManifest.xml."
