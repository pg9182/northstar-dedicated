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

FROM alpine:3.15 AS staging
RUN apk add --no-cache gnutls tzdata ca-certificates sudo
RUN adduser -D northstar && \
    echo 'northstar ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/northstar
RUN mkdir /mnt/titanfall /mnt/mods

COPY --from=build /etc/apk/keys/ /etc/apk/keys/
COPY --from=build-wine /nsbuild/packages/main/x86_64/ /tmp/northstar/
# TODO: remove xvfb once we fix the patches
RUN apk add --no-cache /tmp/northstar/*.apk xvfb && rm -rf /tmp/northstar
COPY --from=build-gfsdk /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar
COPY --from=build-d3d11 /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar
COPY --from=build-northstar /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar
COPY --from=build-entrypoint /nsbuild/packages/main/x86_64/ /tmp/northstar/
RUN apk add --no-cache /tmp/northstar/*.apk && rm -rf /tmp/northstar

FROM scratch
COPY --from=staging / /
USER northstar
RUN /usr/libexec/nsdedi __wineprefix__
# Note: TF2 doesn't seem to like Docker's userspace port forwarding... I suggest
# you use host networking for now.
#EXPOSE 8081/tcp
#EXPOSE 37015/udp
ENTRYPOINT ["/usr/libexec/nsdedi", "                                                          "]
