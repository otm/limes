#!/bin/bash

GIT_TAG=$1

if [[ -e build ]]; then
  rm -rf build
fi

mkdir -p build/darwin/root/usr/local/bin/
mkdir -p build/darwin/root/Library/LaunchDaemons
mkdir -p build/darwin/scripts
mkdir -p build/darwin/flat/base.pkg/

install -m 0755 limes_darwin_amd64 build/darwin/root/usr/local/bin/

install -m 0644 assets/io.github.otm.limes-ip.plist build/darwin/root/Library/LaunchDaemons/
install -m 0755 assets/postinstall.sh build/darwin/scripts/postinstall

NUM_FILES=$(find build/darwin/root | wc -l)
INSTALL_KB_SIZE=$(du -k -s build/darwin/root | awk '{print $1}')

cat <<EOF > build/darwin/flat/base.pkg/PackageInfo
<?xml version="1.0" encoding="utf-8" standalone="no"?>
<pkg-info overwrite-permissions="true" relocatable="false" identifier="io.github.otm.limes" postinstall-action="none" version="${GIT_TAG}" format-version="2" generator-version="InstallCmds-502 (14B25)" auth="root">
    <payload numberOfFiles="${NUM_FILES}" installKBytes="${INSTALL_KB_SIZE}"/>
    <bundle-version/>
    <upgrade-bundle/>
    <update-bundle/>
    <atomic-update-bundle/>
    <strict-identifier/>
    <relocate/>
    <scripts>
        <preinstall/>
        <postinstall file="./postinstall"/>
    </scripts>
</pkg-info>
EOF

BASE=$(pwd)
PKG_LOCATION="Limes-${GIT_TAG}.pkg"

( cd build/darwin/root && find . | cpio -o --format odc --owner 0:80 | gzip -c ) > build/darwin/flat/base.pkg/Payload
( cd build/darwin/scripts && find . | cpio -o --format odc --owner 0:80 | gzip -c ) > build/darwin/flat/base.pkg/Scripts
mkbom -u 0 -g 80 build/darwin/root build/darwin/flat/base.pkg/Bom || exit 1
( cd build/darwin/flat/base.pkg && /usr/local/bin/xar --compression none -cf "${BASE}/${PKG_LOCATION}" * ) || exit 2
echo "osx package has been built: ${PKG_LOCATION}"
