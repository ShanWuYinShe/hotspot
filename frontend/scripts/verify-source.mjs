import { build, transformWithEsbuild } from "vite";
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";

const sourceRoot = "src/app";
const sourceExtensions = new Set([".ts", ".tsx"]);
const lucideImportPattern = /import\s*\{([^}]*)\}\s*from\s*["']lucide-react["']/g;
const coreImports = [
  "ActionIconButton",
  "AppearanceMenu",
  "ContextRail",
  "useDismissibleLayer",
  "useSocial",
  "useComposer",
  "useTheme",
  "usePlatform",
  "ROUTES",
];

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) files.push(...await walk(path));
    else if (sourceExtensions.has(`.${entry.name.split(".").pop()}`)) files.push(path);
  }
  return files;
}

function namedImports(source, modulePattern) {
  const imported = new Set();
  for (const match of source.matchAll(modulePattern)) {
    for (const item of match[1].replace(/\n/g, " ").split(",")) {
      const name = item.trim().split(/\s+as\s+/).at(-1);
      if (name) imported.add(name);
    }
  }
  return imported;
}

function declares(source, name) {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`(?:function|class|const|let|var)\\s+${escaped}\\b|(?:type|interface)\\s+${escaped}\\b`).test(source);
}

const files = await walk(sourceRoot);
const sourceByFile = new Map(await Promise.all(files.map(async (file) => [file, await readFile(file, "utf8")])))
const knownLucide = new Set();
for (const source of sourceByFile.values()) {
  for (const name of namedImports(source, lucideImportPattern)) knownLucide.add(name);
}

const errors = [];
for (const [file, source] of sourceByFile) {
  const display = relative(process.cwd(), file);
  try {
    await transformWithEsbuild(source, file, {
      loader: file.endsWith(".tsx") ? "tsx" : "ts",
      jsx: "automatic",
      target: "es2022",
    });
  } catch (error) {
    errors.push(`${display}: syntax transform failed\n${error.message}`);
  }

  const lucideImports = namedImports(source, lucideImportPattern);
  const jsxNames = new Set([...source.matchAll(/<([A-Z][A-Za-z0-9]*)\b/g)].map((match) => match[1]));
  for (const name of jsxNames) {
    if (knownLucide.has(name) && !lucideImports.has(name)) {
      errors.push(`${display}: <${name} /> is used but not imported from lucide-react.`);
    }
  }

  for (const name of coreImports) {
    const isUsed = new RegExp(`\\b${name}\\b`).test(source);
    const isImported = new RegExp(`import[\\s\\S]*?\\b${name}\\b[\\s\\S]*?from[\\s\\S]*?["'][^"']+["']`).test(source);
    if (isUsed && !isImported && !declares(source, name) && !file.endsWith(`/${name}.tsx`)) {
      errors.push(`${display}: shared primitive ${name} is used without a local declaration or import.`);
    }
  }
}

try {
  // The standalone bundle resolves the dependency graph from the real
  // entry (src/main.tsx via index.html). Figma Make's hosted environment
  // generates its own __figma__entrypoint__.ts; that file is not present
  // in this extracted project, so resolve from the local root instead.
  await build({
    configFile: "vite.config.ts",
    logLevel: "error",
    build: {
      outDir: "/tmp/hotspot-source-audit-dist",
      emptyOutDir: true,
      rollupOptions: {
        input: "src/main.tsx",
        external: ["figma:foundry-client-api"],
      },
    },
  });
} catch (error) {
  errors.push(`Entry dependency bundle failed\n${error.message}`);
}

if (errors.length) {
  console.error("\nSource verification failed:\n");
  for (const error of errors) console.error(`- ${error}`);
  process.exit(1);
}

console.log(`Source verification passed: ${files.length} app source files parsed and entry dependency bundle resolved.`);
