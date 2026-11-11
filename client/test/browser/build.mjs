import * as esbuild from 'esbuild';
import { mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const fixtures = resolve(here, '..', 'fixtures', 'contract');
const out = resolve(here, 'dist');

const files = readdirSync(fixtures)
    .filter((name) => name.endsWith('.json'))
    .sort()
    .map((name) => JSON.parse(readFileSync(join(fixtures, name), 'utf-8')));

mkdirSync(out, { recursive: true });

await esbuild.build({
    entryPoints: [resolve(here, 'runner.ts')],
    bundle: true,
    format: 'iife',
    platform: 'browser',
    target: ['es2020'],
    outfile: join(out, 'contract.js'),
    define: { __FIXTURES__: JSON.stringify(files) },
    logLevel: 'info',
});

writeFileSync(join(out, 'contract.html'), '<!doctype html><html><head><meta charset="utf-8"><title>running</title></head><body><script src="contract.js"></script></body></html>\n');
console.log(`open file://${join(out, 'contract.html')}`);
