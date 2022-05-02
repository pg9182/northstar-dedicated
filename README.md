<h1 align="center">northstar-dedicated</h1>

**Docker image for the [Northstar](https://northstar.tf) dedicated server.** <a href="https://github.com/pg9182/northstar-dedicated/actions/workflows/ci.yml"><img src="https://github.com/pg9182/northstar-dedicated/actions/workflows/ci.yml/badge.svg?branch=master&event=push" align="right"/></a>

[`ghcr.io/pg9182/northstar-dedicated:1-tf2.0.11.0`](https://ghcr.io/pg9182/northstar-dedicated)

## Features

- **Much more efficient** than the running the server on Windows in general, especially around RAM usage.
- **Shares read-only game files** between multiple instances.
- **Environment variable configuration** support for convars and arguments.
- **Stable versioning scheme** with compatibility guarantees, smoothing over changes in Northstar itself and making automatic container updates safer.
- **Layered docker image** for faster builds, smaller downloads, and a much lower disk footprint.
- **Includes custom d3d11 and gfsdk stubs**, reducing the memory usage (by ~700 MB), and eliminating the need for a physical or emulated GPU (these are now part of the [R2Northstar](https://github.com/R2Northstar/NorthstarStubs) organization).
- **Includes a custom wine build** to reduce the size of the container (it will get even smaller soon).
- **Minimal Docker container** based on Alpine Linux.
- **Updates the process name to include the server status** (similar to the process title when running on Windows).
- **Automatically stops the container if Northstar crashes**, allowing it to be automatically restarted by a container orchestrator.
- **Graceful server shutdown** when stopping the container.

## Quick Start

1.  Copy the game files to your server, optionally removing [unnecessary files](#qs-reduce-size) to reduce the size.
2.  Start the container (replace `/path/to/titanfall2` with the patch from step 1):
    ```bash
    docker run --rm --interactive --pull always --publish 8081:8081/tcp --publish 37015:37015/udp --mount "type=bind,source=/path/to/titanfall2,target=/mnt/titanfall,readonly" --env NS_SERVER_NAME="[YOUR-REGION] Your server name" ghcr.io/pg9182/northstar-dedicated:1-tf2.0.11.0
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

- **Kernel:** Linux 5.3+ is recommended for full functionality, but Linux 4.9+ should work for all essential functionality. Older versions may or may not work.
- **CPU:** x86_64, at least 3 cores/threads (in the future, it will likely be able to run on 1).
- **RAM:** 2GB (physical or swap) per instance (it typically peaks to ~1.6GB at launch then settles to around 1GB).
- **Network:** A 16-player instance generally uses about 7-20 Mbps up (note that pilot game modes tend to use more bandwidth than titan-only ones) (this can be reduced significantly; see the FAQ at the bottom).
- **Disk:** With the instructions in the next section, each physical server (the game files are mounted read-only into the container and shared between instances) requires ~4GB for the game files (and ~2GB if you repack the VPKs). The container image is currently ~380MB. At startup, Titanfall reads ~1.75 GB before it reaches the lobby. Storing the files on tmpfs may improve performance.

### Reducing the size <a name="qs-reduce-size"></a>

To slim down a Titanfall 2 install, delete the files as specified below.

- delete `vpk/client_sp_* englishclient_sp_*`
- delete `r2/maps`
- delete `r2/cfg/*.ekv`
- delete `r2/paks/` except for `{patch_master,{common,common_mp}{,\(*}}.rpak ui*`
- delete `r2/sound/**`
- delete `r2/media`
- delete `r2/screenshots`
- delete `bin/x64_retail/{client,auditionwin64}.dll`
- delete `bin/dxsupport.cfg`
- delete `platform`
- delete `Titanfall2.exe`
- on wine without origin installed:
  - delete `Core`
  - delete `Support`
  - delete `__Installer`

<details><summary>Files for Titanfall 2 (v2.0.11.0) dedicated multiplayer</summary>

```
-                                                                      4096 2017-12-05 20:04:00 bin/
-                                                                      4096 2017-12-05 20:04:00 bin/x64_retail/
37bf20436643db1e483fc44a20ff5ec50abaf0c6a863a2eafaac03e9e5cdbd27     109056 2017-11-29 13:20:56 bin/x64_retail/amd_ags_x64.dll
2cd7762823e6e55039af31ba1c398867fc865dfbd68c4e87c19f5f7913e91281     423936 2017-11-29 13:20:56 bin/x64_retail/bink2w64.dll
3fd8fa90f3a0945f08f129536f1f6249d25fdf6cf15b52e8aaf88bb63735252a     197120 2017-11-29 13:21:00 bin/x64_retail/binkawin64.dll
b1a06f7aa52439a948a152bfd3301d9b595c78969bb77c9741bba935139f58a0    3873224 2017-11-29 13:20:56 bin/x64_retail/d3dcompiler_46.dll
0fe9657e21b72ed5e8936f86220474cb25ccaedfc452c8942af7e791881c8df5     980480 2017-12-05 17:39:22 bin/x64_retail/datacache.dll
58eb1a1b44b30275bdd21368de264d856bc310d37d93a0f76f111f0026913487    9843712 2017-12-05 17:40:02 bin/x64_retail/engine.dll
dc8cf61c84b27901d9bf2d192ba209ea311c5fdce0bf2c88d4a04fe8ff68beb4     994304 2017-12-05 17:39:20 bin/x64_retail/filesystem_stdio.dll
011f2d26c98375d33d693e839386f72ff59cfc201a507ef19faafa17c1558be6     967168 2017-11-29 13:20:58 bin/x64_retail/GFSDK_SSAO.win64.dll
0392891217da59eca0a1333e1e1e150ca9d99e8e830ef3f09b9f5bbe25de28de      94720 2017-11-29 13:20:58 bin/x64_retail/GFSDK_TXAA.win64.dll
93e70b95b90d7400dd88f048f5db40b47e16814f10b8c573c249b492b23b47b5     455680 2017-12-05 17:39:18 bin/x64_retail/inputsystem.dll
6791af0641e2346dff3dcbe0fb4baa63914590921ad46b6bfec2f0dc7ce0eff0     549888 2017-12-05 17:39:16 bin/x64_retail/launcher.dll
55cd092c10b06f792e3f72925b3b6687a6237bf615519efe88cafefaba714cb3     485888 2017-12-05 17:39:16 bin/x64_retail/localize.dll
1afffb8dfe2a97a00ca3c5a68ca5277249176c2fd51ee7c5ad70f6eb699a6050    2470400 2017-12-05 17:39:34 bin/x64_retail/materialsystem_dx11.dll
3821976711ec29275518bd0fd914ba7d2d3eb5aff15700450fee035bd026f924     524288 2017-11-29 13:21:06 bin/x64_retail/mileswin64.dll
f3c69398d4fcf944f1c9277a75395d179d1617a9d27314762a173f1c815d70d4    1110528 2017-11-29 13:20:58 bin/x64_retail/OriginSDK.dll
d7d7230dea3ed959422c3503f9e7c477dc356ff6cd9c3199d75a64f2bbc161df     259584 2017-12-05 17:39:20 bin/x64_retail/rtech_game.dll
1c96db33cbda1c52684ff26dcf798c71a3aa69cb16bc7c03ffc7a03fde1de4c5     549376 2017-12-05 17:39:20 bin/x64_retail/studiorender.dll
2ff3a9bf91d67b8d8e2c3c08eba4c8b38e8d39cb876d910d9f187ff1af4f2df0     477696 2017-12-05 17:39:06 bin/x64_retail/tier0.dll
7fe2e4914db6bc3c4dfb2620327c48bb964a6c2267d522b028ace9c63606110f    1256960 2017-12-05 17:39:22 bin/x64_retail/vgui2.dll
bc05f50e777261ed5f1672a33e7d3da7ef7a97d188380f628ad1fe5a0d4b05c6    2662400 2017-12-05 17:39:20 bin/x64_retail/vguimatsurface.dll
2452d36b0f4bc1fc091837db5dc6648aec2d785416f832a00c12217c316f25ba    2029568 2017-12-05 17:39:34 bin/x64_retail/vphysics.dll
b22952a07850d836babdd726b27bba87fd39228658ad44a170fd5e3b51b66caf     521216 2017-12-05 17:39:12 bin/x64_retail/vstdlib.dll
6c9f75a6fb5095719d7536b6c42cb8697dd3b41e16a142cf13694e8937daf92d         22 2017-12-05 17:38:46 build.txt
197a1da0a134f0bd565f92d5939a014882cc64cdaf195ce563d1e0998717dead         12 2017-12-05 17:38:46 gameversion.txt
-                                                                      4096 2017-12-05 20:04:00 r2/
8956fd053a30be003866cd3e7ebb3a6d94735906923201014026086599659daa        258 2017-11-29 13:21:16 r2/aidata.bin
-                                                                      4096 2017-12-05 20:04:00 r2/cfg/
-                                                                      4096 2017-12-05 20:04:00 r2/cfg/client/
07044476e7f6d2ac1e239bb64b951216eb9deb0712c0cff6e2783b5911d0cc0b      88019 2017-12-05 17:27:04 r2/cfg/client/st_data.bin
bd767c0c84bf5318728fd499527bbd8d8718f3d23b3696159772d199b7f42cc2       1320 2017-11-29 13:32:18 r2/cfg/config_default_pc.cfg
8577da2ea54085708b3b851bc50315a36bb740ba5135e747cfb12457b5d3060f          4 2017-11-29 13:32:18 r2/cfg/video_settings_changed_quit.cfg
5b27f13704d139c29bb2fa06ad6b043cb86da3be160704b0ddab8136df40ca4a       1511 2017-11-29 13:32:20 r2/GameInfo.txt
-                                                                      4096 2017-12-05 20:04:00 r2/paks/
-                                                                     16384 2017-12-05 20:04:00 r2/paks/Win64/
4af25130618f67ba87fd7d95b145007c4c45de75c1b6549440bbebc9d22b2e5f   20562504 2016-11-22 15:39:50 r2/paks/Win64/common(01).rpak
2cbb358a71af84700bf1f5194e29898be68cc2532db28af6f8f567aa2208ead1       4104 2016-10-28 17:57:40 r2/paks/Win64/common(01).starpak
168d6cec4e297c62318421d9818acc8fbb699e153f5eef9d980b32486554c50c   20289168 2017-02-11 03:02:56 r2/paks/Win64/common(02).rpak
0ef1a4f41fce6fda3c609a7bec5c28f7f62c2ac4ecce7be6cc017f1c279977b1   32440891 2017-03-14 15:04:46 r2/paks/Win64/common(03).rpak
5ebe1e7f07b5052920c048222b275fa06f266b54cd12e50baaebbc00da3aa2b0   13406799 2017-04-12 16:31:52 r2/paks/Win64/common(04).rpak
30a3da8374c113947f235c25e31a63c52a38800542dbb3a419d492b7012afe6a    5845608 2017-05-10 21:26:20 r2/paks/Win64/common(05).rpak
8db68082922fd7643523b5e93b994d25f7eb10b77ae93ea5f3c498d6db5e78ee    9029539 2017-06-23 18:24:18 r2/paks/Win64/common(06).rpak
8599a9d905cc531f0fd5f7cbbd9e12929cf180dc8e1701defbaeb8b01bafb6a1    8810305 2017-07-11 18:59:02 r2/paks/Win64/common(07).rpak
ea956ba5bd178d26220a27eeb3a39a7986e0ade126368101c92f03472e2b27a0    4644191 2017-08-11 14:33:00 r2/paks/Win64/common(08).rpak
5d33227c0d1944578c83e7405d5bbb44d06d67730fda51a90de83c5ca6f9e3f0    2424616 2017-10-17 14:22:22 r2/paks/Win64/common(09).rpak
3cce564ddff536b3909f23eac5ab08c34fa96c4b04696abe6f2e319ed509e70f    2761313 2017-11-16 16:50:54 r2/paks/Win64/common(10).rpak
8b83f849b113bfce8f28d6e6be3ae49d83fb14501f2c74b36035ed6e4c3039d9    2899655 2017-12-05 17:12:12 r2/paks/Win64/common(11).rpak
1b0f2f9d584bd5ce758664b0b9d852a7e8a1538c7e272816fd44d5aaf5626272  118286538 2016-09-28 14:31:34 r2/paks/Win64/common.rpak
4693f5f25d06c9998d24fc2a127a9f16cff95a6a6150cbf8506b4f18229ccc6c   10989012 2017-06-23 18:24:28 r2/paks/Win64/common_mp(06).rpak
c8c28bbb57a8d799257c63f8a3cabd37ff2a61b57ceedf0a51c53c683d810448     229026 2017-07-11 18:59:04 r2/paks/Win64/common_mp(07).rpak
83a870dfb3394f074eec50c1f27a23c8fa18c4a81301d55297ce91ed2bc4145e     127028 2017-08-11 14:33:04 r2/paks/Win64/common_mp(08).rpak
e362077c4e62d52ccb8f54dbd0b7581529dc63cfcf49e7b4009f8552440ce4c4      78282 2017-10-17 14:22:24 r2/paks/Win64/common_mp(09).rpak
6237a8999383887477b9f5165c58c673360f1a20cb2992ab3e46d8d3d36cc93b      78282 2017-11-16 16:50:58 r2/paks/Win64/common_mp(10).rpak
8ba444af8861ed442100de9549321647b459c38a620b46ea2350037fbf3507cd      94219 2017-12-05 17:12:16 r2/paks/Win64/common_mp(11).rpak
61f1c57ee628088ee62040347c56916b8f86e93e03c686f1f16f7d73156efeb1       2158 2017-12-05 17:24:22 r2/paks/Win64/patch_master.rpak
fd9b2c323abaa5a547f8fa5b4d8dd2fe2669ad0186354ec231ed38378f6ef5de     650752 2016-11-22 15:32:22 r2/paks/Win64/ui(01).dll
29d763d159ff8467b5fec3b7879dfe156bba51fd8a6b8ec699b640039b93a6ca   44394352 2016-11-22 15:32:20 r2/paks/Win64/ui(01).rpak
2cbb358a71af84700bf1f5194e29898be68cc2532db28af6f8f567aa2208ead1       4104 2016-10-28 17:57:24 r2/paks/Win64/ui(01).starpak
5a7b134027285132ae50baa3041a93d21a043dfb1223201aa0a245b7fa051bd3     703488 2017-02-11 02:52:52 r2/paks/Win64/ui(02).dll
29c961e8349cab30ad89de03f04d056ea6825387f2b9cd4e2b91cd744c82b585   42897822 2017-02-11 02:52:50 r2/paks/Win64/ui(02).rpak
230c63eff88448c93183ea06450c2735fb3cca9258e0a952adc38f0645e570fe     769024 2017-03-14 14:39:30 r2/paks/Win64/ui(03).dll
c432bac074bcb3ed06e6da884fc07c3039a1a3601b91df3e2b039db548af224e   49799244 2017-03-14 14:39:28 r2/paks/Win64/ui(03).rpak
8e146edf0dd4cdf5acbc99fd04ba3049fb281deab783f3e77b21d0da8eed5e06     817664 2017-04-12 16:22:22 r2/paks/Win64/ui(04).dll
f02b7e4a2c887d0aced17df1bea5c1dc850e9ffa6370913c73b3eb88af0527aa   34685941 2017-04-12 16:22:18 r2/paks/Win64/ui(04).rpak
0eb19e2870cdf22191d4730d18366ce715451e71bf65fc0cc13752ac10ad68e6     843264 2017-05-10 21:21:46 r2/paks/Win64/ui(05).dll
cfd74e1b1a3aedc8c4a5451d061e68c6f8c110a3a8a73557c0f00ae18c9b8469   58109323 2017-05-10 21:21:42 r2/paks/Win64/ui(05).rpak
f0ddc584963944036705f7c7c3aa4fb8fc7623bf7ebce96387db2e11ec69320c     899584 2017-06-23 18:23:20 r2/paks/Win64/ui(06).dll
549cad8da6dda3f5d7eac18faf376e803e6d5366fb3b1c2eb79cce0d5f8b238c   26588742 2017-06-23 18:23:16 r2/paks/Win64/ui(06).rpak
71b841f3846c0ab5cee5b67d79a70e5c110cdfa71ebc52a8d3c4faa466401e33     942592 2017-07-11 18:57:08 r2/paks/Win64/ui(07).dll
9c306f67b9003805d59f5a1da419444e7714ac124d100be421f877dfb1f0e360   21701185 2017-07-11 18:57:04 r2/paks/Win64/ui(07).rpak
9657cd5bb7755dcd8beb9a14c439c80c098a115b4a8f9d772cb08bab99d48115     944640 2017-08-11 14:31:46 r2/paks/Win64/ui(08).dll
dc8563ae518d03320477f780712070db28d6f539023f31def943c231526e4165   21819001 2017-08-11 14:31:42 r2/paks/Win64/ui(08).rpak
18f32450d59f0d6c87485cea2b7df382edc52a926b01f405dc374dfcef9eca35     945152 2017-10-17 14:21:16 r2/paks/Win64/ui(09).dll
dbc92b036dabf5b63b68c9dd3e6ea2cc13fb0af647fc4bfdaa2e70c325920cf9    9838621 2017-10-17 14:21:12 r2/paks/Win64/ui(09).rpak
4138549e3e399693a2c4bdbb38a7483feaa7f5255004be5d481feac6dc7bd6dd     945664 2017-11-16 16:49:10 r2/paks/Win64/ui(10).dll
b769d64742fc10578d8bc6decdd16977685f71691fa651b5b0f4dc16e7046d0a    1583833 2017-11-16 16:49:06 r2/paks/Win64/ui(10).rpak
ac211cf1b9452e998db88b6614feae7a1e13c6d37c84b247b00f5e4d1a74f135     945664 2017-12-05 17:10:10 r2/paks/Win64/ui(11).dll
3e26172658f94b9853d41f6f0790a635b19fc3c8b9160e54a996d752f1a61b60    1584537 2017-12-05 17:10:06 r2/paks/Win64/ui(11).rpak
49523abff88c059843696adb42bc80066cedb835ccbfe1ff5131c2fac323c391     590336 2016-09-28 14:31:24 r2/paks/Win64/ui.dll
f54e68193b783f19f014878ef1a34b08b1a913c1392b89af3342fc781d2f29e0   42486310 2016-09-28 14:31:22 r2/paks/Win64/ui.rpak
d8987197b67b50450a390e2b5cdf71c22d06b14ef09b063a4dc9d68d4d2b0cd3        258 2017-11-29 13:37:22 r2/pilotaidata.bin
-                                                                      4096 2017-12-05 20:04:00 r2/sound/
d7f48fb134fab750bac00faa25ec0e52dc54aba692debfe95f0011e17882f4d6     145880 2017-11-29 13:38:52 r2/sound/titanfall_2.mprj
d8987197b67b50450a390e2b5cdf71c22d06b14ef09b063a4dc9d68d4d2b0cd3        258 2017-11-29 13:38:56 r2/spectreaidata.bin
d8987197b67b50450a390e2b5cdf71c22d06b14ef09b063a4dc9d68d4d2b0cd3        258 2017-11-29 13:38:56 r2/titanaidata.bin
a5ca3a25c8ae56952a26141b0f6cdcb6c19086c8c39013f7cb345d5d723661af   13680184 2017-12-05 21:03:54 server.dll
-                                                                     16384 2017-12-05 20:04:00 vpk/
d8d4183376ef2eaecf23e7ec78587d357153af1fec35eb6fe62008d0fd273aea   64035225 2016-09-28 15:00:50 vpk/client_frontend.bsp.pak000_000.vpk
cffa27fb940dc6b2cda9e6ce0bf178783cdbbd35b90acc68ac94ebf37e422f31    4483897 2016-10-17 17:50:18 vpk/client_frontend.bsp.pak000_001.vpk
26c1ff0283c5f2434825baebb39fb286ae48cd67a4281cd420019583df4ae25e     383079 2016-10-28 18:08:24 vpk/client_frontend.bsp.pak000_002.vpk
26e9ea0a89efab4192fc372437f7eb1b9a7eb5480553085f5e4c6de0ae59d630    5469721 2016-11-22 17:19:52 vpk/client_frontend.bsp.pak000_003.vpk
f2616be2d904dc450e3e19d10db9e11a2fefd0cdb2afc4e11ced622062123b6f    5745264 2017-02-11 04:54:22 vpk/client_frontend.bsp.pak000_004.vpk
91b3a6f91d099b68170ffc14903e229929906824f2b61b05ded71f6fc2a75a31    5820518 2017-03-14 19:09:24 vpk/client_frontend.bsp.pak000_005.vpk
969d7a8191c1ad2094d7f983029f1462e798d6f3ad63d4cdbfc8f76ad8a54935    6534053 2017-04-12 22:21:32 vpk/client_frontend.bsp.pak000_006.vpk
9880c3794c19383cb3372593175a1eaf87a9f7edf35fbf0d211d87505a97c2f7    5791350 2017-05-10 21:56:18 vpk/client_frontend.bsp.pak000_007.vpk
41e50d7714de7b9aa2248812d9e8109c3268ca78dc042b057ba73e3cbbd6bee5    5969120 2017-06-23 18:43:12 vpk/client_frontend.bsp.pak000_008.vpk
dedbeca5f43500bb715ef5abb9c9e80176af9ab1e4343a1ff6808e00a95c7044    6547430 2017-07-11 19:17:22 vpk/client_frontend.bsp.pak000_009.vpk
27711caf07aa1f555be872885b89bb3ad2826c6da8aa44c7b626a3488ab0919a    4694784 2017-08-11 14:50:32 vpk/client_frontend.bsp.pak000_010.vpk
0ffe9592feccd659f044ade4c77cb6fd1255db07d3940a9ba601f406fefa057f    3299130 2017-10-17 14:39:30 vpk/client_frontend.bsp.pak000_011.vpk
774fd2507a8abbd1390d1740e6c123a3cc1dd043209d8b0bbf3bf23958de34e7    3481691 2017-11-16 17:08:54 vpk/client_frontend.bsp.pak000_012.vpk
20bed61a9d131898f4c50bd9f58970d9b7c29c753a7ef43e54f92ee4f7e2d7fb    3219225 2017-12-05 17:37:54 vpk/client_frontend.bsp.pak000_013.vpk
f01b78b82c6ee8357774a2703a2d76b924b0085dc766ab047f7ecb164b4e756c  139597551 2016-11-22 17:19:48 vpk/client_mp_angel_city.bsp.pak000_000.vpk
bc5486baf2aa288fcd4be7eb7ccca7b1d62fb80f55439f55f3f0ebab9a8de4d4  103111945 2017-02-11 04:53:50 vpk/client_mp_angel_city.bsp.pak000_001.vpk
741b8e55248c00a4a2da3c0f7042b5c12546583666f39af6d4c68291aded8fc0  102870801 2017-03-14 19:08:52 vpk/client_mp_angel_city.bsp.pak000_002.vpk
66fbaac41a83dfa7b58735e281aaf9aeda4595062eed9e08f11360711ecaac4b      60733 2017-04-12 22:20:44 vpk/client_mp_angel_city.bsp.pak000_003.vpk
9b40d1b4f964a2f6d2ebaff9b56d6361ac09d521e86081163a5c86dd14958949     828036 2017-05-10 21:55:32 vpk/client_mp_angel_city.bsp.pak000_004.vpk
afe739966f6d94d1d45d5c77d90aaeb8c927b34067a9a05d3f0f2f08872cd1df      25331 2017-06-23 18:42:24 vpk/client_mp_angel_city.bsp.pak000_005.vpk
8b53231156710952df600b85c20bb2ab66f949b2488d5620e12ba6e5a9bda1c7        201 2017-07-11 19:16:30 vpk/client_mp_angel_city.bsp.pak000_006.vpk
a983f8b68ce205956dfe1dd407d2b7e5aee250cdd79971994096282b53eba4e0      44842 2017-08-11 14:49:56 vpk/client_mp_angel_city.bsp.pak000_007.vpk
0bc3037739351ae6a9874dee106f7daf88ee68f9b0a2fc7991821579f14da670      36422 2017-10-17 14:39:04 vpk/client_mp_angel_city.bsp.pak000_008.vpk
a94661c87edc54a6cd3b3dc16e2ca2b0f506c689f7f65c1300b32df4220aad5f  170624557 2016-09-28 14:59:02 vpk/client_mp_black_water_canal.bsp.pak000_000.vpk
2f13bbc781089ad5d724921dfd4ef1ce7f3914cd45298dba17986572c52ede88      49413 2016-10-17 17:49:54 vpk/client_mp_black_water_canal.bsp.pak000_001.vpk
5257a8123308dceee7c7bf1a5ad44c13ecc8c954205b31e7b25030ed7444753d    1258422 2016-11-22 17:19:06 vpk/client_mp_black_water_canal.bsp.pak000_002.vpk
7c09232865f7fe498338d7fc6891ad42e665ec5b9400802603485b9fa527e488     576512 2017-02-11 04:53:26 vpk/client_mp_black_water_canal.bsp.pak000_003.vpk
3aac5b371ef2bde8124f559dd77fa43b5eaeb966fe3a55a166c85b92d30368c4      40433 2017-03-14 19:08:28 vpk/client_mp_black_water_canal.bsp.pak000_004.vpk
b5762934b2a60fa03ee6d3bf85877cbc721e01d5124fb0e3dd1601923470f7ba      35372 2017-04-12 22:20:30 vpk/client_mp_black_water_canal.bsp.pak000_005.vpk
ab35ee24bc14aca8a8e900a1cc021dfe49ebdb1af8e1923ada03b38f950c5fd5      39967 2017-05-10 21:55:16 vpk/client_mp_black_water_canal.bsp.pak000_006.vpk
f358f0a5184b1f17f43d0a6c2a90de1f8a8a6ed3b20e6dfa725b8c77a59bceb6      29579 2017-06-23 18:42:08 vpk/client_mp_black_water_canal.bsp.pak000_007.vpk
717e16d8d324aad0d3743beb3452b7bc226dc83449b604a47841ff9cad4c1db9      24813 2017-07-11 19:16:14 vpk/client_mp_black_water_canal.bsp.pak000_008.vpk
ef08126b492e018818f96b97968c6531f5cc30654ff981809c240d14246b88ff    7787187 2016-09-28 15:00:04 vpk/client_mp_coliseum.bsp.pak000_000.vpk
7c695756a61c0cd32029120af0316432784ae4863eea17e8d9ce4e52a1e39898     126862 2017-02-11 04:53:36 vpk/client_mp_coliseum.bsp.pak000_001.vpk
42b50fe12b678b3202bf827a8f85036fe9573bde2c98f3b1ca75ff3e3b8896df         58 2017-03-14 19:08:40 vpk/client_mp_coliseum.bsp.pak000_002.vpk
b9fa9b20561d40edf34aef41e7d68f87e048a449d9971958d5b3d9c4b12e6e7f    6411404 2017-02-11 04:54:08 vpk/client_mp_coliseum_column.bsp.pak000_000.vpk
afd839116ed225df6acba3d75e82b170afd1543d7cd4fd1aa51073f02109f1ac   73367140 2017-03-14 19:09:20 vpk/client_mp_colony02.bsp.pak000_000.vpk
5f09b6f9d864fd94c1770492b018ee90ae48f42235b65a04ab38d46397ae5299   24470590 2017-04-12 22:20:52 vpk/client_mp_colony02.bsp.pak000_001.vpk
3e06f86b59658c685135bd8eada1042df12bbe6de0248d30023bf3f727c8f987     850501 2017-05-10 21:55:36 vpk/client_mp_colony02.bsp.pak000_002.vpk
f237416dd7bd52a02989afead12236653859192704b37814348814a9f75e8df4       2484 2017-06-23 18:42:28 vpk/client_mp_colony02.bsp.pak000_003.vpk
1a61609a11b2c2bb6c2c6314837300182aba9a72459b4e40eb7b22f6c4afd240      56878 2017-08-11 14:50:00 vpk/client_mp_colony02.bsp.pak000_004.vpk
d7ce363c1183f76b5aadfccfa00facfae6c4a6f899c522ce4d7ac0feb53deb3c      49772 2017-10-17 14:39:08 vpk/client_mp_colony02.bsp.pak000_005.vpk
973c515ad1585d3a0e89208f4dab397ff3f5ff39afa3376d8ad85a8d225e7816  386878799 2016-09-28 14:37:26 vpk/client_mp_common.bsp.pak000_000.vpk
354aba4ba1a52b469e355045859220718b114b2038120c0e4db3ddafa4e44d63    5432180 2016-10-17 17:46:44 vpk/client_mp_common.bsp.pak000_001.vpk
32bf2e9722217d1022e1945fa666df33584b481a775aa154aef90d1f4f2ab72a     168439 2016-10-28 18:04:54 vpk/client_mp_common.bsp.pak000_002.vpk
e74f210d7ad8015342b1e630b14894547fc0b11e93f9fc350db896b2526c8266   91002902 2016-11-22 17:14:52 vpk/client_mp_common.bsp.pak000_003.vpk
5e70756db3c6b1646aa500eee9882052325696c28686a22dad981aa3a6bd9f89  102603888 2017-02-11 04:49:40 vpk/client_mp_common.bsp.pak000_004.vpk
fc421b1096976533f0916ce16882690b7fcb046da34876036a14ad421bc48ce9   73967916 2017-03-14 19:04:04 vpk/client_mp_common.bsp.pak000_005.vpk
6e8a9e9937d329c6b47366a83fce990ac76640cf1bc740e254c4348518e69487   77460091 2017-04-12 22:16:32 vpk/client_mp_common.bsp.pak000_006.vpk
e42a36817874ba8dbcac28dc5f5b2eb62a42b963b0b776774643c89997785971   88240801 2017-05-10 21:51:10 vpk/client_mp_common.bsp.pak000_007.vpk
7dd13056da8050cf99d8b8d884801ed6ac123afe5d8d2872a71e3b4abfc75496  102657133 2017-06-23 18:37:32 vpk/client_mp_common.bsp.pak000_008.vpk
adc43f28a85b6866d60ac1697dc1f5d246fdc5e4ce0d18a1e9216732634ecccb   82329531 2017-07-11 19:11:32 vpk/client_mp_common.bsp.pak000_009.vpk
1d4794a3266026919ff9502e48c1a5d50ed5e953aefb2a7e9a83dfb54ee3e054   22157072 2017-08-11 14:46:08 vpk/client_mp_common.bsp.pak000_010.vpk
40f59b65317af32653b6558e380248fb7ad61661752a4e61d95e969652978f93    4197760 2017-10-17 14:35:28 vpk/client_mp_common.bsp.pak000_011.vpk
0153af942fbdc7dd9da912900e90af7d84099fb84e5a327a085e5cef57e65f95    1178562 2017-11-16 17:05:02 vpk/client_mp_common.bsp.pak000_012.vpk
79ee263f2bb727c757dc58f0c42387fa9e0860cf5aedaa793f1f00a0b4c41623      72216 2017-12-05 17:33:52 vpk/client_mp_common.bsp.pak000_013.vpk
6b4a73b565e00ed410e07275f962880f79c48b96e97e3943fdfa5479eef950ec  174682818 2016-09-28 15:00:02 vpk/client_mp_complex3.bsp.pak000_000.vpk
002957233afaa239d81fef00dd4c08aef07c61d7cf1471db531e5d0b65c89396      53768 2016-10-17 17:50:02 vpk/client_mp_complex3.bsp.pak000_001.vpk
d4082c9fc13b0a4733c7e5dfe7bcfdcfbb2cdb5739b1972bbb5a604aca2eed22    1532960 2016-11-22 17:19:16 vpk/client_mp_complex3.bsp.pak000_002.vpk
48cff8738bd6637664f67b024f4649387f6eb06e7a92e77397af044b4f4277e1     272531 2017-02-11 04:53:36 vpk/client_mp_complex3.bsp.pak000_003.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:40 vpk/client_mp_complex3.bsp.pak000_004.vpk
7f64cc9d8895424f214f352196b08fe58d38d777e24d8976d6090bd79e3084ce    2482941 2017-05-10 21:55:26 vpk/client_mp_complex3.bsp.pak000_005.vpk
975f67b474c76d93df9597f18aa19a4f53912c557be3614c6986bff944bf6918      61021 2017-06-23 18:42:18 vpk/client_mp_complex3.bsp.pak000_006.vpk
7c2271ff98c9561649977f7c39d12b058f724b01caef2715263f1de451f5a43b      42265 2017-10-17 14:38:58 vpk/client_mp_complex3.bsp.pak000_007.vpk
fa11e056d1180bc8577defd2e849a8a37907ebbb8d2ab51a52c7c524f7d0a622  175100970 2016-09-28 14:58:26 vpk/client_mp_crashsite3.bsp.pak000_000.vpk
b81d5cdcdee956e3b7b90cc02e064375a53be66b136f058ffe2b6e956757c0bd      72030 2016-10-17 17:49:48 vpk/client_mp_crashsite3.bsp.pak000_001.vpk
931f87d1142dc63f3a401a986ad1eb680da9fa82c7fdeb515d58628a30ecea7d     662633 2016-11-22 17:19:00 vpk/client_mp_crashsite3.bsp.pak000_002.vpk
7c695756a61c0cd32029120af0316432784ae4863eea17e8d9ce4e52a1e39898     126862 2017-02-11 04:53:20 vpk/client_mp_crashsite3.bsp.pak000_003.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:24 vpk/client_mp_crashsite3.bsp.pak000_004.vpk
8afe8416a07f0983b231ecacacaa2e7e8b0f31307e2d611e6caf80cda6775db4  136645236 2017-05-10 21:55:12 vpk/client_mp_crashsite3.bsp.pak000_005.vpk
b1dfea83b64d9ec2f604adf78b8306c8bb13f2c91fba26b45aa7a52a4e3960d2      46044 2017-06-23 18:42:04 vpk/client_mp_crashsite3.bsp.pak000_006.vpk
6647935a4b2f441aab0fa2c8166b555633c121cff89a7580eb3d649c0733955b  137157091 2016-09-28 14:59:32 vpk/client_mp_drydock.bsp.pak000_000.vpk
1bf051f74eedf3721a5c7d5bd6032848f72ee90bb2af721083a8e7b19994ddae      48768 2016-10-17 17:49:58 vpk/client_mp_drydock.bsp.pak000_001.vpk
7c0db091f3d2c7ff446a10b78d9b721a375d3d8e812ab64181776e935663425c    1790309 2016-11-22 17:19:12 vpk/client_mp_drydock.bsp.pak000_002.vpk
8fe30445f0e23a56cce29e7054b9c667ab93da09b74591c0079fed7be64f399b    2190218 2017-02-11 04:53:30 vpk/client_mp_drydock.bsp.pak000_003.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:34 vpk/client_mp_drydock.bsp.pak000_004.vpk
79a48637c715c3de7816b1222799aeebf03000c114bd210c8bf8c7fc3fb0e1b6       5162 2017-06-23 18:42:12 vpk/client_mp_drydock.bsp.pak000_005.vpk
64a302bcd8877e6d86032e1298a60e2400dc35bd19dfe81b080b3275b83c2ec6      38295 2017-08-11 14:49:46 vpk/client_mp_drydock.bsp.pak000_006.vpk
7c3039f4789e646ca584eb795b304fece7bc42c5b4245a090a3390055f60d8a6      30115 2017-10-17 14:38:54 vpk/client_mp_drydock.bsp.pak000_007.vpk
a1f7e3484eb04728d82de640358ac3f7b3dcef92ad017c08876ff09b4a7ca361  181511086 2016-09-28 14:58:02 vpk/client_mp_eden.bsp.pak000_000.vpk
b2b18df3f72cab962d14359044461c99311d00e86df51d7797a79e519303df6e      34375 2016-10-17 17:49:42 vpk/client_mp_eden.bsp.pak000_001.vpk
d214fbf2f22eeb5b5871a4e1f1577c205f1a312cee1c99cf3084d9f26b5c44d1    3228114 2016-11-22 17:18:56 vpk/client_mp_eden.bsp.pak000_002.vpk
d3f2786ab2d5735deefe47dd7078bfc38b1fca135c918104a0ba98ce0cc24215     222435 2017-02-11 04:53:16 vpk/client_mp_eden.bsp.pak000_003.vpk
9aab1eb5dc8ca20b47c929d9c375b771efe2bcd71b3a8255009a6602c13e2b11      29225 2017-03-14 19:08:18 vpk/client_mp_eden.bsp.pak000_004.vpk
2175fcac57243c3bac248a58e9f43c1eb8c5ef13049dc968cb9057b6c47c8c83      30581 2017-04-12 22:20:20 vpk/client_mp_eden.bsp.pak000_005.vpk
5f2217bf171e74ab7f084b398328cea39d5663789159635f8c3c551cbc36b4ca      36092 2017-05-10 21:54:58 vpk/client_mp_eden.bsp.pak000_006.vpk
1b04ac8f505c19e8b2bca9682720a53acc34489a8b36f021108daa0c1c3e5cca      39757 2017-06-23 18:41:58 vpk/client_mp_eden.bsp.pak000_007.vpk
329244db08d3ad8d42b6ddc4070be73e88f08da91244c879f3b4f2e34b3a0d4b      37082 2017-08-11 14:49:30 vpk/client_mp_eden.bsp.pak000_008.vpk
59ae1db32d17a8493c40e776cafebdfc44f0f5f72d9059e91995907f0c0373d4  164954488 2016-09-28 14:57:34 vpk/client_mp_forwardbase_kodai.bsp.pak000_000.vpk
e21f592af657085b6375f138a78f795a98bd53f51d81c98c6ce15f44e8c24547      41347 2016-10-17 17:49:38 vpk/client_mp_forwardbase_kodai.bsp.pak000_001.vpk
f9cc5e76050e603ba045a1f5c8480e6c8c2b78dc6d3658c6921c8fb58965ab4a    1530658 2016-11-22 17:18:48 vpk/client_mp_forwardbase_kodai.bsp.pak000_002.vpk
0e56a7a86ae188ee6fef55bd11c69ff65bb1826f519d49a3b6514ab0474d0e2f    1916676 2017-02-11 04:53:10 vpk/client_mp_forwardbase_kodai.bsp.pak000_003.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:14 vpk/client_mp_forwardbase_kodai.bsp.pak000_004.vpk
2c028434c757c2a668c5c65e4a528b3fd5cf3a11996346ca880bdf28a7afa60d      38452 2017-06-23 18:41:54 vpk/client_mp_forwardbase_kodai.bsp.pak000_005.vpk
c2e246ec2e2fa31c6732dca941c4cbf6f9a06af9ad84940b40e0a0e87130f3fd      27941 2017-07-11 19:16:00 vpk/client_mp_forwardbase_kodai.bsp.pak000_006.vpk
8dba055e365b3d749680fc1ae5150eefcc4d7b653f2ec4667732088570b116ca  118965105 2017-04-12 22:21:04 vpk/client_mp_glitch.bsp.pak000_000.vpk
668f56c972bfd5740773b4aec41e768a922c4782b4b356459dff22183f0edd42     450400 2017-06-23 18:42:32 vpk/client_mp_glitch.bsp.pak000_001.vpk
5497f7ddb38a5a0a30ed1e0eff9fb3a0eea77a3b3e3b4c9fae62bde4cefb7232      15005 2017-10-17 14:39:12 vpk/client_mp_glitch.bsp.pak000_002.vpk
86a704964e2be46fb4d1237af2f750db48444b49173136ec47a002993bf1cac5      24795 2017-11-16 17:08:34 vpk/client_mp_glitch.bsp.pak000_003.vpk
c8414a63867e2e78eeb68b04e1daae2d23ba1e2e4bac8f8c93f46dc3ea2db8ed  148000943 2016-09-28 14:57:06 vpk/client_mp_grave.bsp.pak000_000.vpk
ed357efbf6104a2a1301d7b5fa8df6c5423a907f486d37a08b5b45a064b38bb3      53824 2016-10-17 17:49:32 vpk/client_mp_grave.bsp.pak000_001.vpk
5bcf28a90a9596e7c7de3f10c6bffbaa91ecb7a9b78d707d3983fb3bc4c35a3e    1657584 2016-11-22 17:18:44 vpk/client_mp_grave.bsp.pak000_002.vpk
0763added74c0cc6eef98efb155b5a6b719940558da8aff8a3ffc143e2b3ae05    3078616 2017-02-11 04:53:06 vpk/client_mp_grave.bsp.pak000_003.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:10 vpk/client_mp_grave.bsp.pak000_004.vpk
7f64cc9d8895424f214f352196b08fe58d38d777e24d8976d6090bd79e3084ce    2482941 2017-05-10 21:54:48 vpk/client_mp_grave.bsp.pak000_005.vpk
034256f188f0c8b1dbd9e027243b8302739bd3a8e941af9f6f03b262e7e5cd5b      57527 2017-06-23 18:41:48 vpk/client_mp_grave.bsp.pak000_006.vpk
45e15eb61c10a8e05b5ac6c2124bbbb6ce92c49b17cac24cb762b6308b7d2c1a         74 2017-07-11 19:15:54 vpk/client_mp_grave.bsp.pak000_007.vpk
9abc392c9d8f2c10c6a53bb456edfd71f12bc4ce5d7c63e63e209f7f8c4962fb  145541480 2016-09-28 14:56:38 vpk/client_mp_homestead.bsp.pak000_000.vpk
895d5f128902adc0c36ee4afd0073244fe18d01b663718b1912d7a813d0f82db      85175 2016-10-17 17:49:30 vpk/client_mp_homestead.bsp.pak000_001.vpk
060072f9ecc6f2d5262b0c55a674dc6db46d4f5896c57ffba0b2e37e2385acfc    1063451 2016-11-22 17:18:38 vpk/client_mp_homestead.bsp.pak000_002.vpk
7c695756a61c0cd32029120af0316432784ae4863eea17e8d9ce4e52a1e39898     126862 2017-02-11 04:53:00 vpk/client_mp_homestead.bsp.pak000_003.vpk
135f8e630b70c4833f65c6a96f956689d6e4419499308120ba4ff2b971af023a      76184 2017-03-14 19:08:06 vpk/client_mp_homestead.bsp.pak000_004.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:06 vpk/client_mp_homestead.bsp.pak000_005.vpk
586e1bdffa06b711c412dbe1fee85a10058e22477a58f4947c3a942c2305e59d     102887 2017-06-23 18:41:46 vpk/client_mp_homestead.bsp.pak000_006.vpk
bf71c25d6428a603d44a76be12ae1e8c5ab9e156dacc68025bd6e568d50a4c08      94231 2017-07-11 19:15:52 vpk/client_mp_homestead.bsp.pak000_007.vpk
9b15a06afe6d7e992ebc12ef12b81a205337887906a813d863c6fc2e23a4bbb0   53941745 2017-04-12 22:21:18 vpk/client_mp_lf_deck.bsp.pak000_000.vpk
783ec8e15c6256a29f7cda506de6ed81ac1753fad268e2c2dade1a42cfacbc0e   67055446 2017-02-11 04:54:06 vpk/client_mp_lf_meadow.bsp.pak000_000.vpk
4554388e25a96d76559c85139c7c40703893bf289631e2d5e0b8eda162ddb21d        157 2017-03-14 19:08:56 vpk/client_mp_lf_meadow.bsp.pak000_001.vpk
bbe6e24f31dd19771e0045fef242107d84dbd47b466d7b0bf190378460177a53   29431045 2017-04-12 22:20:48 vpk/client_mp_lf_meadow.bsp.pak000_002.vpk
7a24ffdbd26ce54c1a80af9af0156ca0a9c255278fb47006eca05c9662366b6f    1602109 2017-05-10 21:55:34 vpk/client_mp_lf_meadow.bsp.pak000_003.vpk
1d1ac9d4f72ac170221552ed0dce9999ca965498e3345110faa7e53ca47bd1ae   22191193 2017-02-11 04:53:54 vpk/client_mp_lf_stacks.bsp.pak000_000.vpk
d3ef3743c2ff2303eefc234044244d26eadbdfa1c12cadd5c07f3889ef5543f5        156 2017-03-14 19:08:54 vpk/client_mp_lf_stacks.bsp.pak000_001.vpk
ba205e748767b2ee3d426bb83aa8946746e48e65859ba375163bc0c3da07a081       6708 2017-04-12 22:20:46 vpk/client_mp_lf_stacks.bsp.pak000_002.vpk
be4fdcabf486ee08fb8fb80162683b68b01dcaff63ef4d79873a34d619bf8c8f   40333965 2017-07-11 19:17:18 vpk/client_mp_lf_township.bsp.pak000_000.vpk
956477904f189a73e892901f16e5d6031e606c93cda734d8198bc8fe35dd6100   26772577 2017-06-23 18:43:08 vpk/client_mp_lf_traffic.bsp.pak000_000.vpk
bd645a0be7b14bbaf8ae445829e011b814b518ca52b05129e65b9088cf89734b   38005949 2017-08-11 14:50:26 vpk/client_mp_lf_uma.bsp.pak000_000.vpk
7c0c6756bc6cba58e3fbcf0dcfe242db35867acf48809019ec0e4c1ebab251cc  152428864 2017-05-10 21:56:14 vpk/client_mp_relic02.bsp.pak000_000.vpk
2cf582bed35271f2d4729de09696d80ab63d18522687399fbeceebf85ca9f11e      86479 2017-06-23 18:42:36 vpk/client_mp_relic02.bsp.pak000_001.vpk
0f0cfed11713db3f1879d2648781650ad251b5791597f8ceaa51c767cf06af6d      93345 2017-11-16 17:08:40 vpk/client_mp_relic02.bsp.pak000_002.vpk
1b0f75b4593a585b8735203ed7feb26caf54c7872519c92730652bc62d776b03  124777504 2017-07-11 19:17:08 vpk/client_mp_rise.bsp.pak000_000.vpk
823a1cfdf067c5541f2f249c05304593b62d18544ebcb4c7aa9d2ffe75634b5a      53395 2017-08-11 14:50:16 vpk/client_mp_rise.bsp.pak000_001.vpk
40bd005a4e9dfdcf3e3cd9074711c4fc93d1b6e69ac0310a08770dbeae694869      61837 2017-10-17 14:39:24 vpk/client_mp_rise.bsp.pak000_002.vpk
ca714ead871e8cd3af5a82a040c1132b088bd9f0aac9cbcf42cb442ab24d4d27  179675106 2016-09-28 14:56:18 vpk/client_mp_thaw.bsp.pak000_000.vpk
1a8eabc07fc0f13f917daee00c29a7afd8af54b24bf42f97203c3bf18b302068      50809 2016-10-17 17:49:26 vpk/client_mp_thaw.bsp.pak000_001.vpk
3c931bf98af7549960e9ef530a58fd539e9ea9e359a2f0d99e5310ba582a2a16     465554 2016-11-22 17:18:34 vpk/client_mp_thaw.bsp.pak000_002.vpk
f5488cadf10047901ef17b9d4968aeb8f9b3936334ba63db11967a7b98fe4a23     151327 2017-02-11 04:52:56 vpk/client_mp_thaw.bsp.pak000_003.vpk
1a417147e61b60c0fc10771d33f53f1a6c6e49dd9882b8ef94821590d43618d7      13873 2017-03-14 19:08:02 vpk/client_mp_thaw.bsp.pak000_004.vpk
50089bba53f578c95954d490bafa9e094e10e557181d7a71173600c82c81454c       1209 2017-04-12 22:20:02 vpk/client_mp_thaw.bsp.pak000_005.vpk
b208cdaf7c450b3e6cb13888ed49f1a553ad06c2e48d5f47f277897c9f421b36      48374 2017-06-23 18:41:40 vpk/client_mp_thaw.bsp.pak000_006.vpk
aebe9ef79834d2036a326b362380538ea9b77817f24a2dc3bd86b1deb3b26832      51707 2017-07-11 19:15:48 vpk/client_mp_thaw.bsp.pak000_007.vpk
a1056519e6319457ba0074720578764a979a798989d1dffbbf0c64b75c041fbc      69053 2017-08-11 14:49:14 vpk/client_mp_thaw.bsp.pak000_008.vpk
ca0dbff03828a3fd48bc77fe36837a4450b416faf2be4a5ad2cf232e10b7798e   85046828 2017-06-23 18:43:02 vpk/client_mp_wargames.bsp.pak000_000.vpk
b6b853a4d0c8fe34693c0ba5277ab906f32a993bf5c6b35c317c3726bea824ef      43327 2017-07-11 19:16:46 vpk/client_mp_wargames.bsp.pak000_001.vpk
8f4dbffcf9ce06b60335a853c242bef4e0daaba0afeb890c251b69c9f795c3a8      42817 2017-08-11 14:50:12 vpk/client_mp_wargames.bsp.pak000_002.vpk
0c76d45d7dc9f8bb2be09f627f2f00127eff0de38e3dabc3032c69e058337393      43452 2017-10-17 14:39:20 vpk/client_mp_wargames.bsp.pak000_003.vpk
2f1c37754e38b4ac70a9559fbfbdbd28feda47f36be9cd9e3ad5bc844ee3feb6          4 2017-12-05 17:27:02 vpk/enable.txt
fd81fc08eee716d1fcbaf2f8ca47fd42610bc3ab19a0329d178cf916348f2996     130810 2017-12-05 17:37:54 vpk/englishclient_frontend.bsp.pak000_dir.vpk
5e4e9ef00c1e6f651689ab136df70e6ade56c8647db02c526280707801e7ab85     238911 2017-12-05 17:37:16 vpk/englishclient_mp_angel_city.bsp.pak000_dir.vpk
b95eb5da024fd98ba1a631498c528c9a3a3aeb1e79858f8271f56885e4c38333     249959 2017-12-05 17:37:02 vpk/englishclient_mp_black_water_canal.bsp.pak000_dir.vpk
e497598b65136aad67fc8cce708ca3b9f3b5b5ed3f1cb9b8a8a2e70561369079      58315 2017-12-05 17:37:12 vpk/englishclient_mp_coliseum.bsp.pak000_dir.vpk
ba3a2a26a147d108d8a403b6bd2b950803cf3771d03b573156632cf64c70b9ee      58520 2017-12-05 17:37:18 vpk/englishclient_mp_coliseum_column.bsp.pak000_dir.vpk
7528c9e5e090ded9508799afae1f416b3ba7534c072dbe391e568686419fa0b9     122369 2017-12-05 17:37:20 vpk/englishclient_mp_colony02.bsp.pak000_dir.vpk
fc5236a2861817426ad94a4d215dc6e76ed6e7565946960989025592b302cec5     205609 2017-12-05 17:33:52 vpk/englishclient_mp_common.bsp.pak000_dir.vpk
5a0e7953f6ee5c54ab8e5af54888951e9da172fbfb111103d67e290d12ee3a2c     225016 2017-12-05 17:37:12 vpk/englishclient_mp_complex3.bsp.pak000_dir.vpk
b599f5ed4a7a4064f64e0267bfeeacd17bef77b8b6bc76923ba24ffa38e3c7c4     219433 2017-12-05 17:36:56 vpk/englishclient_mp_crashsite3.bsp.pak000_dir.vpk
fd24ffe8c01908753f5f39f52bd9080b5e4092fffcdb109cb7a63900bde8570b     220066 2017-12-05 17:37:06 vpk/englishclient_mp_drydock.bsp.pak000_dir.vpk
b2a30138c7632387a4f4c2cf89d2a838c6c114594bb47e3ce8f0d78703418df7     230935 2017-12-05 17:36:52 vpk/englishclient_mp_eden.bsp.pak000_dir.vpk
20d5a0eeb8c13e66ddee107f4e176a55e2e80ccacf95f2a1fa775f7ab3f4d7cd     228594 2017-12-05 17:36:46 vpk/englishclient_mp_forwardbase_kodai.bsp.pak000_dir.vpk
acb99d71f899832f76284e38cd2ab05ea4e39e085905760d549bc4b44fcb5a93     119830 2017-12-05 17:37:24 vpk/englishclient_mp_glitch.bsp.pak000_dir.vpk
c3b1721cf66e530ffeb679f5c2ea3a3b8b768907dee85613e856f4420697405e     168681 2017-12-05 17:36:42 vpk/englishclient_mp_grave.bsp.pak000_dir.vpk
fdf3f717be35338eff143ebfe85a23b39d1460438717e71ac5e03698f376a913     192473 2017-12-05 17:36:38 vpk/englishclient_mp_homestead.bsp.pak000_dir.vpk
bc75c5e730ecfcc5564eac877cbf4b5e8663cf6e515cb8cc02f337fe86fb33d3      76395 2017-12-05 17:37:26 vpk/englishclient_mp_lf_deck.bsp.pak000_dir.vpk
a9501435e722abb2eab4f7eda08c2638aa6f82ec46400a08d7e96fe78cacead5      92332 2017-12-05 17:37:18 vpk/englishclient_mp_lf_meadow.bsp.pak000_dir.vpk
caeb7ad824e803abfd4bff2238571182169e033b7e9f44fd7058094c8ecd03c0      67937 2017-12-05 17:37:18 vpk/englishclient_mp_lf_stacks.bsp.pak000_dir.vpk
1c989ac874257fee1d0a1887da4059f9d9a8f8ccd84859ad18fa0989bf6d4ebd      50437 2017-12-05 17:37:40 vpk/englishclient_mp_lf_township.bsp.pak000_dir.vpk
27feaa2ecec018c89ecff30c8293026f73d345420864c0e04c914a92c89aa454      46759 2017-12-05 17:37:40 vpk/englishclient_mp_lf_traffic.bsp.pak000_dir.vpk
70dda276a3715025d017f32f2056688c5266f79cb5de5224e7ee492e281e6402      48889 2017-12-05 17:37:38 vpk/englishclient_mp_lf_uma.bsp.pak000_dir.vpk
e614e7c259ce77c6dfbe4a7199044b3bd26e7e9580601bcf0bd19c1711c3a35c     174101 2017-12-05 17:37:30 vpk/englishclient_mp_relic02.bsp.pak000_dir.vpk
ce7d7e926261d16f8deddc150d9e1e816a187e4c106616a3192bc0e2fd37baaf     119446 2017-12-05 17:37:38 vpk/englishclient_mp_rise.bsp.pak000_dir.vpk
8f5343bd17047bbea1168153ba6e3b347e96cd1db42b9416f8bc64b8ed2e28be     215329 2017-12-05 17:36:34 vpk/englishclient_mp_thaw.bsp.pak000_dir.vpk
0f18ab8c3f0dcf005af2c0f2a6fce5418120b4e43459843b10ea5d193f067918     109254 2017-12-05 17:37:32 vpk/englishclient_mp_wargames.bsp.pak000_dir.vpk
```

</details>

The size can be reduced even more (~2 GiB) by using my [`tf2-vpkoptim`](https://github.com/pg9182/tf2vpk) tool to repack the VPKs.

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
<li>X11 (so DirectX context creation doesn't fail) (Xvfb won't usually work for a few reasons unless you compile and use my d3d11 + gfsdk stubs or are on Northstar 1.6.0+ (which includes them) instead of the options below)</li>
<li>Titanfall 2 (just copied game files, no need to install)</li>
<li>Northstar 1.1.2+</li>
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

### Windows

The d3d11 and gfsdk stubs I developed for northstar-dedicated can be used when running the dedicated server on Windows to drastically reduce the required resources. Unlike with Wine, you will need to put them in `bin/x64_retail` instead of the directory `NorthstarLauncher.exe` is contained in. See [here](https://github.com/R2Northstar/NorthstarStubs) for more details.

## Container

TODO: More documentation, FAQ, etc.

### Configuration

#### Game files

Titanfall 2 is not included in the container image for size and legal reasons. The game files must be obtained separately and mounted read-only to `/mnt/titanfall`.

#### Mods

To include additional mods, mount them under `/mnt/mods/`, which is equivalent to the `R2Northstar/mods` folder (preferably read-only unless you know what you are doing).

#### Navs

To enable AI and auto-titans, navmeshes and graphs should be mounted to `/mnt/navs/` or a subdirectory. Prebuilt nav files can be found [here](https://github.com/taskinoz/Northstar-Navs). If a file conflicts with one already added, it will be overridden (in lexical order). On Northstar v1.7.0 and later, navmeshes are included for all maps by default.

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

- **The server status in htop isn't updating** <br/>
  Press F2 to enter setup, then enable `Update process names on every refresh`. I also recommend enabling `Hide userland process threads` to reduce the clutter.
- **How do I override built-in mods?** <br/>
  You can extend the image with your changes as additional steps modifying `/usr/lib/northstar`. Alternatively, you can mount the mods read-only into `/usr/lib/northstar/R2Northstar/mods`, but this is not officially supported and may break at any time.
- **How do I get old logs after a crash?** <br/>
  With the default Docker configuration, if you add a name to the container and remove `--rm`, you will be able to used `docker logs` to view them. You can also use a log management solution like Loki (via promtail or the Docker driver). Consider adding `+spewlog_enable 0` to `NS_EXTRA_ARGUMENTS` to reduce the logspam.
- **How can I optimize the server and reduce the bandwidth required for running it?** <br/>
  Add `+net_compresspackets 1 +net_compresspackets_minsize 64 +sv_maxrate 127000` to `NS_EXTRA_ARGUMENTS`. The CPU overhead is neglegible.

### Deployment

The following sections provide example configuration for deploying the container using container orchestration tools.

#### docker-compose

Example configuration (simple):

```yml
version: "3.9"

services:
  northstar1:
    image: ghcr.io/pg9182/northstar-dedicated:1-tf2.0.11.0
    pull_policy: always
    environment:
      - NS_PORT=37015
      - NS_PORT_AUTH=8081
      - 'NS_SERVER_NAME=your server name'
      - 'NS_SERVER_DESC=your server description'
      - |
        NS_EXTRA_ARGUMENTS=
        +setplaylist private_match
        +net_compresspackets_minsize 64
        +net_compresspackets 1
        +spewlog_enable 0
        +sv_maxrate 127000
    volumes:
      - /path/to/titanfall:/mnt/titanfall:ro
    ports:
      - '37015:37015/udp'
      - '8081:8081/tcp'
    restart: always
```

Example configuration (complex):

```yml
version: "3.9"

x-logging:
  &logging
  logging:
    driver: "json-file"
    options:
      max-file: "5"
      max-size: "400m"

services:
  northstar1:
    << : *logging
    image: ghcr.io/pg9182/northstar-dedicated:1-tf2.0.11.0-ns1.6.3
    pull_policy: always
    environment:
      - NS_PORT=37015
      - NS_PORT_AUTH=37115
      - 'NS_SERVER_NAME=your server name, possibly with {{hostname}}'
      - 'NS_SERVER_DESC=your server description, which can also include {{hostname}}'
      - NS_RETURN_TO_LOBBY=0
      - NS_INSECURE=0
      - |
        NS_EXTRA_ARGUMENTS=
        +setplaylist private_match
        +ns_private_match_only_host_can_change_settings 1
        +ns_private_match_last_mode turbo_ttdm
        +ns_private_match_countdown_length 0
        +ns_should_return_to_lobby 0
        +setplaylistvaroverrides "max_players 24 respawn_delay 0 run_epilogue 0 earn_meter_titan_multiplier 2 aegis_upgrades 1 titan_shield_regen 1"
        +net_compresspackets_minsize 64
        +net_compresspackets 1
        +spewlog_enable 0
        +sv_maxrate 127000
        +rcon_admin 1009497984978
        +grant_admin 1009497984978
        +autoannounce "map console (`) commands: !skip to vote skip, !extend to vote extend"
    volumes:
      - ./titanfall/2.0.11.0-dedicated-mp:/mnt/titanfall:ro
      - ./navs:/mnt/navs:ro
      - ./mods/RCON:/mnt/mods/RCON:ro
      - ./mods/Karma.Abuse:/mnt/mods/Karma.Abuse:ro
      - ./mods/Takyon.PlayerVote:/mnt/mods/Takyon.PlayerVote:ro
    ports:
      - '37015:37015/udp'
      - '37115:37115/tcp'
    restart: always
```

#### kubernetes

TODO

#### Nomad

TODO
