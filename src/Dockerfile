FROM alpine:3.15 AS base

FROM base AS build
RUN apk update && apk add alpine-sdk sudo
RUN adduser -Dh /nsbuild nsbuild && \
    adduser nsbuild abuild && \
    echo 'nsbuild ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/nsbuild

USER nsbuild
WORKDIR /nsbuild/
RUN mkdir -p /nsbuild/src/main /nsbuild/packages/main && \
    abuild-keygen -ain

FROM build AS build-wine
COPY ./wine/ /nsbuild/src/main/wine
RUN cd /nsbuild/src/main/wine && sudo chown -R nsbuild . && abuild -r

FROM build AS build-gfsdk
COPY ./gfsdk/ /nsbuild/src/main/gfsdk
RUN cd /nsbuild/src/main/gfsdk && sudo chown -R nsbuild . && abuild -r

FROM build AS build-d3d11
COPY ./d3d11/ /nsbuild/src/main/d3d11
RUN cd /nsbuild/src/main/d3d11 && sudo chown -R nsbuild . && abuild -r

FROM build AS build-northstar
COPY ./northstar/ /nsbuild/src/main/northstar
RUN cd /nsbuild/src/main/northstar && sudo chown -R nsbuild . && abuild -r

FROM build AS build-entrypoint
COPY ./entrypoint/ /nsbuild/src/main/entrypoint
RUN cd /nsbuild/src/main/entrypoint && sudo chown -R nsbuild . && abuild -r

# these convoluted steps allow us to have alpine+wine, northstar, our custom
# stuff, and the user+wineprefix in their own layers while keeping the rest flat
# (also see https://github.com/moby/moby/issues/21950)

FROM base AS pkgstaging-base
RUN apk add --no-cache gnutls tzdata ca-certificates sudo
COPY --from=build /etc/apk/keys/ /etc/apk/keys/
COPY --from=build-wine /nsbuild/packages/main/x86_64/ /tmp/northstar/
# TODO: remove xvfb once we fix the patches
RUN apk add --no-cache /tmp/northstar/*.apk xvfb && rm -rf /tmp/northstar

FROM pkgstaging-base AS pkgstaging-1
COPY --from=build-northstar /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar

FROM scratch AS pkgstaging-1-diff
COPY --from=pkgstaging-1 /lib/apk /lib/apk
COPY --from=pkgstaging-1 /usr/lib/northstar /usr/lib/northstar

FROM pkgstaging-1 AS pkgstaging-2
COPY --from=build-gfsdk /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar
COPY --from=build-d3d11 /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar
COPY --from=build-entrypoint /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar

FROM scratch AS pkgstaging-2-diff
COPY --from=pkgstaging-2 /lib/apk /lib/apk
COPY --from=pkgstaging-2 /usr/lib/northstar-dedicated-gfsdk /usr/lib/northstar-dedicated-gfsdk
COPY --from=pkgstaging-2 /usr/lib/northstar-dedicated-d3d11 /usr/lib/northstar-dedicated-d3d11
COPY --from=pkgstaging-2 /usr/libexec/nsdedi /usr/libexec/nsdedi

FROM scratch AS pkgstaging-merged
COPY --from=pkgstaging-base / /
COPY --from=pkgstaging-1-diff / /
COPY --from=pkgstaging-2-diff / /

FROM pkgstaging-merged AS pkgstaging-merged-audit
RUN { ! apk audit --system | grep -v 'X usr/share/fonts/encodings/encodings.dir' | grep -E '^[MX] ' >&2; } || \
    { echo "error: missing files, ensure the Dockerfile is up-to-date with the packages" >&2; exit 2; }

FROM pkgstaging-merged AS staging
RUN adduser -D northstar && \
    echo 'northstar ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/northstar && \
    mkdir /mnt/titanfall /mnt/mods
USER northstar

FROM staging
RUN /usr/libexec/nsdedi __wineprefix__
EXPOSE 8081/tcp
EXPOSE 37015/udp
ENTRYPOINT ["/usr/libexec/nsdedi", "                                                          "]
