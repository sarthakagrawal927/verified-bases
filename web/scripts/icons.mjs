#!/usr/bin/env node
/**
 * Generate raster favicon assets from the SVG source.
 *
 * Run automatically via `npm run prebuild`. Idempotent — re-run safely
 * after editing public/favicon.svg.
 *
 * Outputs:
 *   public/favicon-32.png       — 32×32 transparent
 *   public/apple-touch-icon.png — 180×180 with dark background
 *   public/og.png               — 1200×630 fallback for OG image
 *
 * sharp is bundled with Astro (image optimization), so no extra dep.
 */
import { readFile, writeFile, stat } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pub = resolve(here, '..', 'public');

let sharp;
try {
  sharp = (await import('sharp')).default;
} catch (err) {
  console.warn('[icons] sharp not available — skipping favicon rasterization.');
  console.warn('         Run `npm install` first, or commit pre-built PNGs.');
  process.exit(0);
}

async function fresh(out, src) {
  try {
    const [outStat, srcStat] = await Promise.all([stat(out), stat(src)]);
    return outStat.mtimeMs >= srcStat.mtimeMs;
  } catch {
    return false;
  }
}

async function rasterize({ src, out, size, bg }) {
  if (await fresh(out, src)) {
    console.log(`[icons] ${out.replace(pub + '/', '')} up-to-date`);
    return;
  }
  const svg = await readFile(src);
  let pipe = sharp(svg, { density: 300 }).resize(...size, { fit: 'contain', background: bg ?? { r: 0, g: 0, b: 0, alpha: 0 } });
  if (bg) pipe = pipe.flatten({ background: bg });
  const buf = await pipe.png().toBuffer();
  await writeFile(out, buf);
  console.log(`[icons] wrote ${out.replace(pub + '/', '')} (${buf.length} bytes)`);
}

await rasterize({
  src:  resolve(pub, 'favicon.svg'),
  out:  resolve(pub, 'favicon-32.png'),
  size: [32, 32],
});

await rasterize({
  src:  resolve(pub, 'favicon.svg'),
  out:  resolve(pub, 'apple-touch-icon.png'),
  size: [180, 180],
  bg:   { r: 10, g: 10, b: 11 },
});

await rasterize({
  src:  resolve(pub, 'og.svg'),
  out:  resolve(pub, 'og.png'),
  size: [1200, 630],
  bg:   { r: 10, g: 10, b: 11 },
});
