<h1 align="center">northstar-dedicated</h1>

**Docker image for the [Northstar](https://northstar.tf) dedicated server.** <a href="https://github.com/pg9182/northstar-dedicated/actions/workflows/ci.yml"><img src="https://github.com/pg9182/northstar-dedicated/actions/workflows/ci.yml/badge.svg?branch=master&event=push" align="right"/></a>

[`ghcr.io/pg9182/northstar-dedicated:1-tf2.0.11.0`](https://ghcr.io/pg9182/northstar-dedicated)

## Features

- **Shares read-only game files** between multiple instances.
- **Environment variable configuration** support for convars and arguments.
- **Stable versioning scheme** with compatibility guarantees, smoothing over changes in Northstar itself and making automatic container updates safer.
- **Layered docker image** for faster builds, smaller downloads, and a much lower disk footprint.
- **Includes custom d3d11 and gfsdk stubs**, reducing the memory usage (by ~700 MB), and eliminating the need for a physical or emulated GPU.
- **Includes a custom wine build** to reduce the size of the container (it will get even smaller soon).
- **Minimal Docker container** based on Alpine Linux.
- **Updates the process name to include the server status** (similar to the process title when running on Windows).
- **Automatically stops the container if Northstar crashes**, allowing it to be automatically restarted by a container orchestrator.
- **Graceful server shutdown** when stopping the container.

## Quick Start

1.  Copy the game files to your server, optionally removing [unnecessary files](#qs-reduce-size) to reduce the size.
2.  Start the container (replace `/path/to/titanfall2` with the patch from step 1):
    ```bash
    docker run --rm --interactive --pull always --publish 8081:8081/tcp --publish 37015:37015/udp --mount "type=bind,source=/path/to/titanfall2,target=/mnt/titanfall" --env NS_SERVER_NAME="[YOUR-REGION] Your server name" ghcr.io/pg9182/northstar-dedicated:1-tf2.0.11.0
    ```

## Versioning

### Stable

These tags will be used for stable versions from the master branch. The major version will be incremented when a breaking change is made.

- `{major-version}.{YYMMDD}.git{short-hash}-tf{titanfall-ver}-ns{northstar-ver}` (unique tag)
- `{major-version}-tf{titanfall-ver}-ns{northstar-ver}` (recommended if you're using custom config)
- `{major-version}-tf{titanfall-ver}` (recommended if you're only using the documented env vars)

**Note:** There isn't a `latest` tag since the Titanfall 2 version may need to be updated manually.

### Development

These tags will be used for the dev branch and for all CI builds.

- `dev.{YYMMDD}.git{short-hash}` (unique tag)

## Titanfall 2

To use this docker container, you will need a copy of the Titanfall 2 game files to mount as a volume. You can reduce the size to around 5 gigabytes with the instructions in the next section.

### System requirements

- **CPU:** x86_64, at least 4 cores (in the future, it will likely be able to run on 1).
- **RAM:** 2GB (physical or swap) per instance (it typically peaks to ~1.4GB at launch then settles to around 1GB).
- **Network:** A 16-player instance generally uses about 7-20 Mbps up (note that pilot game modes tend to use more bandwidth than titan-only ones).
- **Disk:** With the instructions in the next section, each physical server (the game files are mounted read-only into the container and shared between instances) requires ~5GB for the game files. The container image is currently ~1.2GB, but this will be reduced in a future version. At startup, Titanfall reads ~1.75 GB before it reaches the lobby. Storing the files on tmpfs may improve performance.

### Reducing the size <a name="qs-reduce-size"></a>

To slim down a Titanfall 2 install, delete the files as specified below.

- delete `vpk/client_sp_* englishclient_sp_*`
- delete `r2/maps/sp_*`
- delete `r2/paks/` except for `{patch_master,{common,common_mp,highlight,particle_scripts}{,\(*}}.rpak ui{,\(*}.dll`
- delete `r2/sound/**`
- delete `r2/media`
- delete `r2/screenshots`
- on wine without origin installed:
  - delete `Core`
  - delete `Support`
  - delete `__Installer`

TODO: It's possible to reduce it by at least another gigabyte by stripping textures and models from the VPKs.

### Running with wine

**This is not relevant if you are here for the Docker container.** If you want to try running this without Docker, my stubs, or my custom Wine build, I've successfully used the following configurations. Note that you will need a full X server running (Xvfb won't do).

<details><table><thead><tr><th colspan="2">Common</th></tr><tbody><tr><td colspan="2"><ul>
<li>Fedora 35</li>
<li>ext4 filesystem</li>
<li>Wine, either:<ul>
<li>Wine 7.0.0-rc2 (distro packages)</li>
<li><details><summary>Wine 7.0.0-rc2 (source)</summary>
<pre><code>dnf install kernel-devel mingw64-gcc gnutls-devel libjpeg-turbo-devel libpng-devel mesa-libGL-devel libunwind-devel xorg-x11-server-devel mesa-libOSMesa-devel
mkdir wine-build wine-pkg
pushd wine-build
../wine/configure --enable-win64 --without-alsa --without-capi --without-coreaudio --without-cups --without-dbus --without-fontconfig --without-freetype --without-gettext --without-gphoto --without-gssapi --without-gstreamer --without-netapi --without-krb5 --without-ldap --without-openal --without-opencl --without-oss --without-pcap --without-pulse --without-sane --without-sdl --without-udev --without-usb --without-v4l2 --without-vkd3d --without-vulkan --without-xcomposite --without-xcursor --without-xfixes --without-xinput --without-xinput2 --without-xrandr --without-xinerama --without-xshape --without-xshm --without-xxf86vm --with-x
make -j4
popd
make -C wine-build install DESTDIR=$PWD/wine-pkg</pre></code>
</details></li>
</ul></li>
<li>X11 (so DirectX context creation doesn't fail) (Xvfb won't usually work for a few reasons unless you compile and use my d3d11 + gfsdk stubs instead of the options below)</li>
<li>Titanfall 2 (just copied game files, no need to install)</li>
<li>Northstar 1.1.2</li>
<li>Mesa 21.3.2 (distro packages)</li>
<li><code>WINEARCH=win64</code></li>
<li><code>WINEDEBUG=-all</code></li>
</ul></td></tr><tr><th colspan="2">Either:</th></tr><tr><td width="50%" valign="top"><ul>
<li>libGL (distro packages)</li>
<li>WineD3D (built-in)</li>
<li><code>LIBGL_ALWAYS_SOFTWARE=1</code></li>
<li><code>GALLIUM_DRIVER=llvmpipe</code></li>
</ul></td><td width="50%" valign="top"><ul>
<li>Vulkan (distro packages)</li>
<li>DXVK 1.9.2</li>
<li><code>VULKAN_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.x86_64.json</code></li>
</ul><p>
<b>Note:</b> DXVK will always use your physical adapter instead of lavapipe if you have one present unless you patch <a href="https://github.com/doitsujin/dxvk/blob/94674ac45e8a4618c00519e04b254de76aad35a2/src/dxvk/dxvk_device_filter.cpp#L33">this check</a>.
</p></td></tr></tbody></table></details>

### Tracing

While working on the stubs, I needed to trace which DirectX calls Titanfall uses. You can do this by running it under WineD3D (with llvmpipe so it isn't polluted or made inconsistent by the hardware) and `WINEDEBUG=trace+d3d11,trace+dxgi`. Beware that this may have more than what is directly called by Titanfall due to internal implementation details of WineD3D.

### ARM64

ARM64 support (mostly for running this on the Oracle Free Tier) is currently blocked by [ptitSeb/box64#210](https://github.com/ptitSeb/box64/issues/210). FEX and qemu-user are too slow to run Northstar.

## Container

TODO: More documentation, FAQ, etc.

### Configuration

#### Game files

Titanfall 2 is not included in the container image for size and legal reasons. The game files must be obtained separately and mounted read-only to `/mnt/titanfall`.

#### Mods

To include additional mods, mount them under `/mnt/mods/`, which is equivalent to the `R2Northstar/mods` folder (preferably read-only unless you know what you are doing).

#### Environment variables

The following environment variables are mapped to convars or command-line arguments as necessary and will continue to be supported in releases of the image with the same major version. The default values are based on official Northstar releases and can be found in [nsinit.go](./src/entrypoint/nsinit.go).

| Environment variable      | Description |
| ---                       | --- |
| NS_SERVER_NAME            | **Required.** The server name to show in the server browser. |
| NS_SERVER_DESC            | The server description to show in the server browser. |
| NS_SERVER_PASSWORD        | The password for the server. If empty, the server is public. |
| NS_PORT                   | The UDP game port. Must match with the forwarded port and be accessible from the default external IP. |
| NS_PORT_AUTH              | The TCP player authentication port. Must match with the forwarded port and be accessible from the default external IP. |
| NS_MASTERSERVER_URL       | The base URL of the master server. |
| NS_MASTERSERVER_REGISTER  | True/false for whether the server should register with the master server. If false, you will probably want to set NS_INSECURE to true. |
| NS_INSECURE               | Whether to allow unauthenticated direct connections to the server. |

In `NS_SERVER_NAME` and `NS_SERVER_DESC`, `{{hostname}}` will be replaced with the container's hostname (usually a short random string). This is useful to easily start and identify multiple instances dynamically, or to match instances with container logs.

Additional command-line arguments (including convars starting with `+`) can be provided via the `NS_EXTRA_ARGUMENTS` environment variable. Arguments including spaces must be quoted using shell quoting rules.

### FAQ

- **How do I override built-in mods?** <br/>
  You can extend the image with your changes as additional steps modifying `/usr/lib/northstar`. Alternatively, you can mount the mods read-only into `/usr/lib/northstar/R2Northstar/mods`, but this is not officially supported and may break at any time.

### Deployment

The following sections provide example configuration for deploying the container using container orchestration tools.

#### docker-compose

TODO

#### kubernetes

TODO

#### Nomad

TODO
