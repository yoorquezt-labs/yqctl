#!/usr/bin/env node

// Downloads the platform-specific yqctl binary from GitHub Releases.
// This is the postinstall script for `npm install -g yqctl`.

const os = require("os");
const fs = require("fs");
const path = require("path");
const https = require("https");
const { execSync } = require("child_process");

const REPO = "yoorquezt-labs/yqctl";
const BIN_NAME = os.platform() === "win32" ? "yqctl.exe" : "yqctl";

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const platforms = {
    "darwin-x64": "yqctl_darwin_amd64",
    "darwin-arm64": "yqctl_darwin_arm64",
    "linux-x64": "yqctl_linux_amd64",
    "linux-arm64": "yqctl_linux_arm64",
    "win32-x64": "yqctl_windows_amd64",
  };

  const key = `${platform}-${arch}`;
  const name = platforms[key];

  if (!name) {
    console.error(`Unsupported platform: ${key}`);
    console.error("Please install from source: go install github.com/yoorquezt-labs/yqctl@latest");
    process.exit(1);
  }

  return name;
}

function getVersion() {
  const pkg = require("./package.json");
  return pkg.version;
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (url) => {
      https.get(url, { headers: { "User-Agent": "yqctl-npm" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          follow(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: HTTP ${res.statusCode}`));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => {
          file.close();
          resolve();
        });
      }).on("error", reject);
    };
    follow(url);
  });
}

async function main() {
  const version = getVersion();
  const platform = getPlatform();
  const ext = os.platform() === "win32" ? ".zip" : ".tar.gz";
  const archive = `${platform}${ext}`;

  const url = `https://github.com/${REPO}/releases/download/v${version}/${archive}`;
  const binDir = path.join(__dirname, "bin");
  const tmpFile = path.join(__dirname, `tmp_${archive}`);

  console.log(`Downloading yqctl v${version} for ${platform}...`);

  fs.mkdirSync(binDir, { recursive: true });

  try {
    await download(url, tmpFile);

    if (ext === ".tar.gz") {
      execSync(`tar -xzf "${tmpFile}" -C "${binDir}" ${BIN_NAME}`, { stdio: "pipe" });
    } else {
      execSync(`unzip -o "${tmpFile}" ${BIN_NAME} -d "${binDir}"`, { stdio: "pipe" });
    }

    fs.chmodSync(path.join(binDir, BIN_NAME), 0o755);
    fs.unlinkSync(tmpFile);

    console.log(`yqctl v${version} installed successfully!`);
  } catch (err) {
    console.error(`Failed to install yqctl: ${err.message}`);
    console.error("Fallback: go install github.com/yoorquezt-labs/yqctl@latest");

    // Clean up
    try { fs.unlinkSync(tmpFile); } catch {}
    process.exit(1);
  }
}

main();
