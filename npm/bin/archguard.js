#!/usr/bin/env node

const fs = require("fs");
const os = require("os");
const path = require("path");
const https = require("https");
const { spawnSync, execFileSync } = require("child_process");

const repoOwner = "honzikec";
const repoName = "archguard";
const pkg = require("../package.json");

async function main() {
  const explicitBinary = process.env.ARCHGUARD_BINARY;
  const binary = explicitBinary || (await ensureBinary());
  const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
  if (result.error) {
    throw result.error;
  }
  if (result.signal) {
    process.kill(process.pid, result.signal);
    return;
  }
  process.exit(result.status === null ? 1 : result.status);
}

async function ensureBinary() {
  const version = await resolveVersion();
  const cacheDir = path.join(os.homedir(), ".cache", "archguard", version);
  const extension = os.platform() === "win32" ? ".exe" : "";
  const binaryPath = path.join(cacheDir, `archguard${extension}`);
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  fs.mkdirSync(cacheDir, { recursive: true });
  const archiveName = assetName(version);
  const archivePath = path.join(cacheDir, archiveName);
  const downloadUrl = `https://github.com/${repoOwner}/${repoName}/releases/download/${version}/${archiveName}`;
  await downloadFile(downloadUrl, archivePath);
  execFileSync("tar", ["-xf", archivePath, "-C", cacheDir], { stdio: "inherit" });
  fs.chmodSync(binaryPath, 0o755);
  return binaryPath;
}

async function resolveVersion() {
  if (process.env.ARCHGUARD_VERSION) {
    return normalizeVersion(process.env.ARCHGUARD_VERSION);
  }
  if (!pkg.version || pkg.version === "0.0.0-dev") {
    const latest = await fetchJson(`https://api.github.com/repos/${repoOwner}/${repoName}/releases/latest`);
    return normalizeVersion(latest.tag_name);
  }
  return normalizeVersion(`v${pkg.version}`);
}

function normalizeVersion(version) {
  if (!version) {
    throw new Error("ArchGuard version is required.");
  }
  return version.startsWith("v") ? version : `v${version}`;
}

function assetName(version) {
  const normalizedVersion = version.replace(/^v/, "");
  const platform = mapPlatform(os.platform());
  const arch = mapArch(os.arch());
  const extension = platform === "windows" ? "zip" : "tar.gz";
  return `archguard_${normalizedVersion}_${platform}_${arch}.${extension}`;
}

function mapPlatform(platform) {
  switch (platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }
}

function mapArch(arch) {
  switch (arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }
}

function downloadFile(url, destination) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: {
        "User-Agent": "@archguard/cli"
      }
    }, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        downloadFile(response.headers.location, destination).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download ${url}: status ${response.statusCode}`));
        response.resume();
        return;
      }
      const file = fs.createWriteStream(destination);
      response.pipe(file);
      file.on("finish", () => {
        file.close(resolve);
      });
      file.on("error", (error) => {
        fs.unlink(destination, () => reject(error));
      });
    });
    request.on("error", reject);
  });
}

function fetchJson(url) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: {
        "Accept": "application/vnd.github+json",
        "User-Agent": "@archguard/cli"
      }
    }, (response) => {
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to fetch ${url}: status ${response.statusCode}`));
        response.resume();
        return;
      }
      let body = "";
      response.setEncoding("utf8");
      response.on("data", (chunk) => {
        body += chunk;
      });
      response.on("end", () => {
        try {
          resolve(JSON.parse(body));
        } catch (error) {
          reject(error);
        }
      });
    });
    request.on("error", reject);
  });
}

main().catch((error) => {
  console.error(error.message || String(error));
  process.exit(1);
});
