#!/bin/bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
    echo "usage: $0 image_name (stable|stable-git|dev-git)..."
    exit 2
fi

cd "$(dirname "$0")/.."
seg_ever="$(cd src/entrypoint && . APKBUILD && echo "$pkgver")"
seg_nsver="$(cd src/northstar && . APKBUILD && echo "$pkgver")"
seg_tfver="$(cd src/northstar && . APKBUILD && echo "$pkgver_tf")"
seg_git="$(git rev-parse --short HEAD)"
seg_yymmdd="$(date --utc +%Y%m%d)"

img="$1"; shift

declare -a tags
for x in $*; do
    case $x in
    "stable")
        tags+=(
            "${img}:${seg_ever}-tf${seg_tfver}-ns${seg_nsver}"
            "${img}:${seg_ever}-tf${seg_tfver}"
        ) ;;
    "stable-git")
        tags+=(
            "${img}:${seg_ever}.${seg_yymmdd}.git${seg_git}-tf${seg_tfver}-ns${seg_nsver}"
        ) ;;
    "dev-git")
        tags+=(
            "${img}:dev.${seg_yymmdd}.git${seg_git}"
        ) ;;
    *)
        echo "invalid argument: $x" >&2
        exit 1
        ;;
    esac
done

printf "%s\n" "${tags[@]}"

if [[ ${GITHUB_ACTIONS-} == "true" && -n ${TAGS_GHA_OUTPUT-} ]]; then
    echo "::set-output name=${TAGS_GHA_OUTPUT}::$(IFS=","; echo "${tags[*]}")"
fi
