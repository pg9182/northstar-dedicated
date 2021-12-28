# northstar-dedicated
Docker image for the Northstar dedicated server.

## TODO

Most of my progress and notes can be found on my repos and gists if anyone's interested. Some of it is also discussed on the NS discord.

Note that I'm not currently accepting contributions (at least until I finish the directx stubbing and the initial version).

### Wine

- [X] Figure out how to run it natively.
- [X] Figure out how to run it with wined3d + llvmpipe.
- [X] Figure out how to run it with dxvk + lavapipe.
- [X] Build minimal version.
- [ ] Figure out how to run it without an X server.
- [X] Build minimal version.

### Northstar

- [ ] Create additional patches for optimization where necessary.
- [ ] Implement a CLI flag to specify the TF2 install folder (so TF2 can be mounted into the container).
- [ ] Fix console creation so it re-uses the original one.
- [ ] Expose prometheus metrics?

### Titanfall 2

- [X] Figure out which files are unnecessary.
- [ ] Maybe: remove textures from vpks.

### dxvk

- [X] Initial research and API traces.
- [X] Remove the win32 surface so it can run on xvfb.
- [X] Neutralize the render loop / presenter.
- [ ] Finish removing vulkan from dxvk fork by incrementally stubbing / re-implementing APIs.
  - [ ] Neutralize texture loading and shader compilation, but still return valid buffers where needed.
- [ ] Remove leftover cruft.
- [ ] Slim down the build system.

### docker

- [ ] Create initial docker image based on Fedora.
- [X] Design versioning scheme.
- [ ] Create a slimmer docker image from scratch or based on Alpine with custom packages.
- [ ] Provide config via a custom entrypoint which supports env vars.
- [ ] Fix Northstar logging.
- [ ] GitHub actions.

### misc

- [ ] Webpage.
- [ ] Docs.

## Versioning

Tentative.

Tags (stable - master branch):

- `{number-for-breaking-changes}.{YYMMDD}-tf{titanfall-version}-ns{northstar-version}`
- `{number-for-breaking-changes}.{YYMMDD}` || `dev` for the dev branch
- `{number-for-breaking-changes}-tf{titanfall-version}`

Note: no latest tag, since people will need to update tf2 manually

Tags (dev - CI builds):
- `dev` for main development branch
- `dev.{YYMMDD}.git{short-hash}`
