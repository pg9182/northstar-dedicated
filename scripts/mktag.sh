#!/bin/bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
    echo "usage: $0 image_name (stable|stable-git|dev|dev-git)..."
    exit 2
fi

cd "$(dirname "$0")/.."
seg_ever="$(cd src/entrypoint && . APKBUILD && echo "$pkgver")"
seg_nsver="$(cd src/northstar && . APKBUILD && echo "$pkgver")"
seg_tfver="$(cd src/northstar && . APKBUILD && echo "$pkgver_tf")"
seg_git="$(git rev-parse --short HEAD)"
seg_yymmdd="$(date --utc +%Y%m%d)"

img="$1"; shift

for x in $*; do
    case $x in
    "stable")
        echo "${img}:${seg_ever}-tf${seg_tfver}"
        echo "${img}:${seg_ever}-tf${seg_tfver}-ns${seg_nsver}"
        ;;
    "stable-git")
        echo "${img}:${seg_ever}.${seg_yymmdd}.git${seg_git}-tf${seg_tfver}-ns${seg_nsver}"
        ;;
    "dev")
        echo "${img}:dev"
        echo "${img}:dev.${seg_yymmdd}"
        ;;
    "dev-git")
        echo "${img}:dev.${seg_yymmdd}.git${seg_git}"
        ;;
    *)
        echo "invalid argument: $x" >&2
        exit 1
        ;;
    esac
done
