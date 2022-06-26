# syntax=docker/dockerfile:1.2

FROM alpine:3.15.4 AS base

FROM base AS build
RUN apk update && apk add alpine-sdk sudo
RUN adduser -Dh /nsbuild nsbuild && \
    adduser nsbuild abuild && \
    echo 'nsbuild ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/nsbuild

USER nsbuild
RUN mkdir -p /nsbuild/src/main /nsbuild/packages/main && \
    abuild-keygen -ain

FROM build AS build-wine
COPY --chown=nsbuild:nsbuild ./wine/ /nsbuild/src/main/wine/
RUN ulimit -n 1024; cd /nsbuild/src/main/wine && abuild -r

FROM build AS build-northstar
COPY --chown=nsbuild:nsbuild ./northstar/ /nsbuild/src/main/northstar/
RUN ulimit -n 1024; cd /nsbuild/src/main/northstar && abuild -r

FROM build AS build-nswrap
COPY --chown=nsbuild:nsbuild ./nswrap/ /nsbuild/src/main/nswrap/
RUN ulimit -n 1024; cd /nsbuild/src/main/nswrap && abuild -r

FROM build AS build-entrypoint
COPY --chown=nsbuild:nsbuild ./entrypoint/ /nsbuild/src/main/entrypoint/
RUN ulimit -n 1024; cd /nsbuild/src/main/entrypoint && abuild -r

FROM base
RUN apk add --no-cache gnutls tzdata ca-certificates sudo
RUN --mount=from=build-wine,source=/nsbuild/packages/main/x86_64,target=/nsbuild/wine \
    apk add --no-cache --allow-untrusted /nsbuild/wine/northstar-dedicated-wine-[0-9]*-r*.apk xvfb
RUN --mount=from=build-northstar,source=/nsbuild/packages/main/x86_64,target=/nsbuild/northstar \
    apk add --no-cache --allow-untrusted /nsbuild/northstar/*.apk
RUN --mount=from=build-nswrap,source=/nsbuild/packages/main/x86_64,target=/nsbuild/nswrap \
    --mount=from=build-entrypoint,source=/nsbuild/packages/main/x86_64,target=/nsbuild/entrypoint \
    apk add --no-cache --allow-untrusted /nsbuild/nswrap/*.apk /nsbuild/entrypoint/*.apk

# silence Xvfb xkbcomp warnings by working around the bug (present in libX11 1.7.2) fixed in libX11 1.8 by https://gitlab.freedesktop.org/xorg/lib/libx11/-/merge_requests/79
RUN echo 'partial xkb_symbols "evdev" {};' > /usr/share/X11/xkb/symbols/inet

RUN adduser -D northstar && \
    echo 'northstar ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/northstar && \
    mkdir /mnt/titanfall /mnt/mods /mnt/navs
USER northstar

ENV WINEPREFIX="/home/northstar/.wine"
RUN /usr/bin/nswrap-wineprefix && \
    for x in \
        /home/northstar/.wine/drive_c/"Program Files"/"Common Files"/System/*/* \
        /home/northstar/.wine/drive_c/windows/* \
        /home/northstar/.wine/drive_c/windows/system32/* \
        /home/northstar/.wine/drive_c/windows/system32/drivers/* \
        /home/northstar/.wine/drive_c/windows/system32/wbem/* \
        /home/northstar/.wine/drive_c/windows/system32/spool/drivers/x64/*/* \
        /home/northstar/.wine/drive_c/windows/system32/Speech/common/* \
        /home/northstar/.wine/drive_c/windows/winsxs/*/* \
    ; do \
        orig="/usr/lib/wine/x86_64-windows/$(basename "$x")"; \
        if cmp -s "$orig" "$x"; then ln -sf "$orig" "$x"; fi; \
    done && \
    for x in \
        /home/northstar/.wine/drive_c/windows/globalization/sorting/*.nls \
        /home/northstar/.wine/drive_c/windows/system32/*.nls \
    ; do \
        orig="/usr/share/wine/nls/$(basename "$x")"; \
        if cmp -s "$orig" "$x"; then ln -sf "$orig" "$x"; fi; \
    done
EXPOSE 8081/tcp
EXPOSE 37015/udp
ENTRYPOINT ["/usr/libexec/nsdedi"]
