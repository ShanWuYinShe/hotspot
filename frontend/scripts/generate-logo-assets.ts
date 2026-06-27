import sharp from 'sharp';
import pngToIco from 'png-to-ico';
import { mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';

const projectRoot = process.cwd();
const assetsDir = join(projectRoot, 'assets', 'logo');
const staticDir = join(projectRoot, 'static');
const masterPath = join(assetsDir, 'logo-master.png');

const pwaIcons = [192, 512];
const namedOutputs: Array<{ size: number; file: string }> = [
  { size: 16, file: 'favicon-16x16.png' },
  { size: 32, file: 'favicon-32x32.png' },
  { size: 180, file: 'apple-touch-icon.png' }
];

async function ensureDirectories() {
  mkdirSync(assetsDir, { recursive: true });
}

async function generatePng(size: number, name: string) {
  const outPath = join(staticDir, name);
  await sharp(masterPath).resize(size, size).png({ compressionLevel: 9 }).toFile(outPath);
}

async function generateSvg() {
  const pngBuffer = await sharp(masterPath)
    .resize(512, 512)
    .png({ compressionLevel: 9 })
    .toBuffer();
  const base64 = pngBuffer.toString('base64');
  const svgContent = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><image href="data:image/png;base64,${base64}" width="512" height="512"/></svg>`;
  writeFileSync(join(staticDir, 'favicon.svg'), svgContent);
}

async function generateFaviconIco() {
  const icoSizes = [16, 32, 48];
  const buffers = await Promise.all(
    icoSizes.map((size) =>
      sharp(masterPath).resize(size, size).png({ compressionLevel: 9 }).toBuffer()
    )
  );
  const icoBuffer = await pngToIco(buffers);
  writeFileSync(join(staticDir, 'favicon.ico'), icoBuffer);
}

async function main() {
  await ensureDirectories();
  for (const size of pwaIcons) {
    await generatePng(size, `android-chrome-${size}x${size}.png`);
  }
  for (const output of namedOutputs) {
    await generatePng(output.size, output.file);
  }
  await generateSvg();
  await generateFaviconIco();
}

main().catch((error) => {
  console.error('Failed to generate logo assets', error);
  process.exit(1);
});
