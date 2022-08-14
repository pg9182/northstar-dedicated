#!/usr/bin/env python3
"""
- Checks Github API for newest version of Northstar
- Downloads it for checksum calculation 
- Updates APKBUILD accordingly
"""
# %%

# Imports
# Only modules in the standard library were chosen to avoid needing to install Python dependencies
import hashlib
import re
import json
import urllib.request

# Pre-defined values
REPO_API_RELEASE_URL = "https://api.github.com/repos/R2Northstar/Northstar/releases/latest"

NORTHSTAR_APKBUILD_NAME = "src/northstar/APKBUILD"

# Get JSON object containing info about all releases
api_response = urllib.request.urlopen(REPO_API_RELEASE_URL).read()
api_response

# %%

# Get version number of newest release
release_version = json.loads(api_response)["tag_name"].replace("v", "")
print(f"Newest version is: {release_version}")

# The part of the JSON object that refers to the zip archive
tar_object = [x for x in json.loads(
    api_response)["assets"] if "zip" in x["name"]][0]


# Simple check to ensure that versions match up
assert(release_version in tar_object["name"])

# Download zip archive file for checksum calculation
g = urllib.request.urlopen(tar_object["browser_download_url"])
with open(tar_object["name"], "bw") as f:
    f.write(g.read())

# Calculate SHA256 checksum to update in manifest
sha512_hash = hashlib.sha512()
with open(tar_object["name"], "rb") as f:
    # Read and update hash string value in blocks of 4K
    for byte_block in iter(lambda: f.read(4096), b""):
        sha512_hash.update(byte_block)
    release_checksum = sha512_hash.hexdigest()


# Read Flatpak manifest
with open(NORTHSTAR_APKBUILD_NAME, "rt") as f:
    file_content = f.read()

# Set version number, size, and checksum
file_content = re.sub(r"pkgver=(\d+\.\d+\.\d+)", f"pkgver={release_version}", file_content)
file_content = re.sub(r"[0-9a-fA-F]+  Northstar\.release\.v\$pkgver\.zip", f"{release_checksum}  Northstar.release.v$pkgver.zip", file_content)

# Write back updated content
with open(NORTHSTAR_APKBUILD_NAME, "wt") as f:
    f.write(file_content)

print("Done")

# %%
