# northstar-dedicated

Docker image for the Northstar dedicated server. **Not ready yet (it'll probably be another day or two).**

## Versioning

Tentative.

### Stable

These tags will be used for stable versions from the master branch. The major version will be incremented when a breaking change is made.

- `{major-version}.{YYMMDD}.git{short-hash}-tf{titanfall-ver}-ns{northstar-ver}` (unique tag)
- `{major-version}-tf{titanfall-ver}-ns{northstar-ver}` (recommended if you're using custom config)
- `{major-version}-tf{titanfall-ver}` (recommended if you're only using the documented env vars)

**Note:** There isn't a `latest` tag since the Titanfall 2 version may need to be updated manually.

### Development

These tags will be used for the dev branch and for all CI builds.

- `dev.{YYMMDD}.git{short-hash}` (unique tag)
- `dev` (only for the main dev branch)

## Titanfall 2

To use this docker container, you will need a copy of the Titanfall 2 game files to mount as a volume. You can reduce the size to around 5 gigabytes with the instructions in the next section.

### Reducing the size

To slim down a Titanfall 2 install, delete the files as specified below.

- delete `vpk/client_sp_* englishclient_sp_*`
- delete `r2/maps/sp_*`
- delete `r2/paks/` except for `{patch_master,{common,common_mp,highlight,particle_scripts}{,\(*}}.rpak ui{,\(*}.dll`
- delete `r2/sound`
- delete `r2/media`
- delete `r2/screenshots`
- on wine without origin installed:
  - delete `Core`
  - delete `Support`
  - delete `__Installer`

### System requirements

TODO

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
<li>X11 (so DirectX context creation doesn't fail) (Xvfb won't usually work for a few reasons)/li>
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

## TODO

Most of my progress and notes can be found on my repos and gists if anyone's interested. Some of it is also discussed on the NS discord. I've also been adding the most relevant bits to this README as I finalize them.

Note that I'm not currently accepting contributions (at least until I finish the directx stubbing and the initial version).

### Wine

- [X] Figure out how to run it natively.
- [X] Figure out how to run it with wined3d + llvmpipe.
- [X] Figure out how to run it with dxvk + lavapipe.
- [X] Build minimal version.
- [X] Figure out how to run it without an X server.
- [X] Build minimal version.
- [ ] Get it working on ARM64 with Box64 / Hangover / qemu-user.

### Northstar

- [ ] Create additional patches for optimization where necessary.
- [ ] Fix console creation so it re-uses the original one.
- [ ] Expose prometheus metrics?

### Titanfall 2

- [X] Figure out which files are unnecessary.
- [X] Create stubs for `GFSDK_{SSAO,TXAA}.win64.dll` so d3d9 isn't required (thanks p0358).
- [ ] Maybe: remove textures from vpks.

### dxvk

- [X] Initial research and API traces.
- [X] Remove the win32 surface so it can run on xvfb.
- [X] Neutralize the render loop / presenter.
- [X] Finish removing vulkan from dxvk fork by incrementally stubbing / re-implementing APIs.
  - [X] Neutralize texture loading and shader compilation, but still return valid buffers where needed.
- [X] Remove leftover cruft.
- [X] Slim down the build system.

### docker

- [X] Create initial docker image based on Fedora.
- [X] Design versioning scheme.
- [X] Create a slimmer docker image from scratch or based on Alpine with custom packages.
- [X] Figure out a way to be able to mount TF2 read-only without copying it.
- [X] Provide config via a custom entrypoint which supports env vars.
- [X] Fix Northstar logging.
- [ ] GitHub actions.
- [ ] Windows support? (this depends on whether my hook will work on it)
- [ ] Publish image.

### misc

- [ ] Webpage.
- [ ] Docs.
