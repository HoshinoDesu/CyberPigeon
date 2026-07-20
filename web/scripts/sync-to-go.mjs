import {
  cpSync,
  existsSync,
  mkdirSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { join, resolve } from "node:path";

const webRoot = resolve(import.meta.dirname, "..");
const outDir = join(webRoot, "out");
const targetDir = resolve(webRoot, "../internal/server/web");

if (!existsSync(outDir)) {
  console.error("Missing web/out. Run `npm run build` first.");
  process.exit(1);
}

mkdirSync(targetDir, { recursive: true });

for (const entry of readdirSync(targetDir)) {
  rmSync(join(targetDir, entry), { recursive: true, force: true });
}

/** Skip verbose RSC flight text dumps; keep real static assets. */
function shouldCopy(name, isDir) {
  if (name.startsWith("__next.") && !isDir) return false;
  if (name.endsWith(".txt") && !isDir && !name.startsWith("_")) return false;
  // Keep _buildManifest.js etc.; drop only plain *.txt debug dumps.
  if (name.endsWith(".txt") && !isDir) return false;
  return true;
}

function copyRecursive(src, dest) {
  const st = statSync(src);
  if (st.isDirectory()) {
    const children = readdirSync(src).filter((name) => {
      const childPath = join(src, name);
      const childStat = statSync(childPath);
      return shouldCopy(name, childStat.isDirectory());
    });
    if (children.length === 0) return false;

    mkdirSync(dest, { recursive: true });
    let wrote = false;
    for (const name of children) {
      if (copyRecursive(join(src, name), join(dest, name))) wrote = true;
    }
    if (!wrote) {
      rmSync(dest, { recursive: true, force: true });
      return false;
    }
    return true;
  }

  const base = src.split(/[\\/]/).pop();
  if (!shouldCopy(base, false)) return false;
  cpSync(src, dest);
  return true;
}

if (!copyRecursive(outDir, targetDir)) {
  console.error("Nothing was copied from web/out");
  process.exit(1);
}

function pruneEmptyDirs(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (!statSync(p).isDirectory()) continue;
    pruneEmptyDirs(p);
    if (readdirSync(p).length === 0) {
      rmSync(p, { recursive: true, force: true });
    }
  }
}
pruneEmptyDirs(targetDir);

if (!existsSync(join(targetDir, "index.html"))) {
  console.error("index.html missing after sync");
  process.exit(1);
}

writeFileSync(
  join(targetDir, ".build-info"),
  `synced_from=web/out\nsynced_at=${new Date().toISOString()}\n`,
);

console.log(`Synced static export -> ${targetDir}`);
