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
seg_rfc3339="$(date --utc --rfc-3339=seconds)"

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

if [[ ${GITHUB_ACTIONS-} == "true" ]]; then
    echo "tags=$(IFS=","; echo "${tags[*]}")" >> $GITHUB_OUTPUT
    echo "ever=${seg_ever}" >> $GITHUB_OUTPUT
    echo "nsver=${seg_nsver}" >> $GITHUB_OUTPUT
    echo "tfver=${seg_tfver}" >> $GITHUB_OUTPUT
    echo "git=${seg_git}" >> $GITHUB_OUTPUT
    echo "yymmdd=${seg_yymmdd}" >> $GITHUB_OUTPUT
    echo "rfc3339=${seg_rfc3339}" >> $GITHUB_OUTPUT
fi
