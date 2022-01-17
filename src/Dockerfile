# syntax=docker/dockerfile:1.2

FROM alpine:3.15.0 AS base

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
RUN cd /nsbuild/src/main/wine && abuild -r

FROM build AS build-gfsdk
COPY --chown=nsbuild:nsbuild ./gfsdk/ /nsbuild/src/main/gfsdk/
RUN cd /nsbuild/src/main/gfsdk && abuild -r

FROM build AS build-d3d11
COPY --chown=nsbuild:nsbuild ./d3d11/ /nsbuild/src/main/d3d11/
RUN cd /nsbuild/src/main/d3d11 && abuild -r

FROM build AS build-northstar
COPY --chown=nsbuild:nsbuild ./northstar/ /nsbuild/src/main/northstar/
RUN cd /nsbuild/src/main/northstar && abuild -r

FROM build AS build-entrypoint
COPY --chown=nsbuild:nsbuild ./entrypoint/ /nsbuild/src/main/entrypoint/
RUN cd /nsbuild/src/main/entrypoint && abuild -r

FROM base
RUN apk add --no-cache gnutls tzdata ca-certificates sudo
RUN --mount=from=build,source=/etc/apk/keys,target=/etc/apk/keys \
    --mount=from=build-wine,source=/nsbuild/packages/main/x86_64,target=/nsbuild/wine \
    apk add --no-cache /nsbuild/wine/northstar-dedicated-wine-[0-9]*-r*.apk xvfb
RUN --mount=from=build,source=/etc/apk/keys,target=/etc/apk/keys \
    --mount=from=build-northstar,source=/nsbuild/packages/main/x86_64,target=/nsbuild/northstar \
    apk add --no-cache /nsbuild/northstar/*.apk
RUN --mount=from=build,source=/etc/apk/keys,target=/etc/apk/keys \
    --mount=from=build-gfsdk,source=/nsbuild/packages/main/x86_64,target=/nsbuild/gfsdk \
    --mount=from=build-d3d11,source=/nsbuild/packages/main/x86_64,target=/nsbuild/d3d11 \
    --mount=from=build-entrypoint,source=/nsbuild/packages/main/x86_64,target=/nsbuild/entrypoint \
    apk add --no-cache /nsbuild/gfsdk/*.apk /nsbuild/d3d11/*.apk /nsbuild/entrypoint/*.apk

RUN adduser -D northstar && \
    echo 'northstar ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/northstar && \
    mkdir /mnt/titanfall /mnt/mods
USER northstar

RUN /usr/libexec/nsdedi __wineprefix__ && \
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
ENTRYPOINT ["/usr/libexec/nsdedi", "                                                                                                                        "]
