import * as esbuild from 'esbuild';
import {fileURLToPath} from 'url';
import {basename, dirname, relative, resolve} from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const isWatch = process.argv.includes('--watch');
const writeLedger = process.argv.includes('--write-ledger');
const ledgerPath = resolve(__dirname, 'size-ledger.json');
const LEDGER_SLACK = 1.02;
const LEDGER_WRITE_MARGIN = 1.05;

const outDir = resolve(__dirname, '../internal/server/static');

const baseConfig = {
    entryPoints: [resolve(__dirname, 'src/index.ts')],
    bundle: true,
    format: 'iife',
    platform: 'browser',
    target: ['es2020'],
    globalName: 'LiveUIModule',
    treeShaking: true,
    metafile: true,
    legalComments: 'none',
    mainFields: ['browser', 'module', 'main'],
    conditions: ['browser'],
    banner: {
        js: `// LiveUI Client v1.0.0 - Built with esbuild`,
    },
};

const targets = [
    {
        label: 'production',
        outfile: resolve(outDir, 'pondlive.js'),
        config: {
            minify: true,
            sourcemap: false,
            drop:  ['console', 'debugger'],
        },
        postprocess: async (result) => {
            await reportBundleSize(result.outputFiles?.[0]?.path ?? resolve(outDir, 'pondlive.js'), 'esbuild');
            await checkLedger(result.metafile);
            await minifyWithTerser(resolve(outDir, 'pondlive.js'));
        },
    },
    {
        label: 'development',
        outfile: resolve(outDir, 'pondlive-dev.js'),
        config: {
            sourcemap: true,
            minify: false,
            drop: [],
        },
        postprocess: async (result) => {
            await reportBundleSize(result.outputFiles?.[0]?.path ?? resolve(outDir, 'pondlive-dev.js'), 'esbuild');
        },
    },
];

async function checkLedger(metafile) {
    const fs = await import('fs');
    const measured = {};
    for (const output of Object.values(metafile.outputs)) {
        for (const [input, info] of Object.entries(output.inputs)) {
            const key = relative(__dirname, resolve(process.cwd(), input)).split('\\').join('/');
            measured[key] = (measured[key] ?? 0) + info.bytesInOutput;
        }
    }
    if (writeLedger) {
        const lines = {};
        for (const key of Object.keys(measured).sort()) {
            lines[key] = Math.ceil(measured[key] * LEDGER_WRITE_MARGIN);
        }
        fs.writeFileSync(ledgerPath, JSON.stringify(lines, null, 2) + '\n');
        console.log(`ledger written: ${Object.keys(lines).length} modules`);
        return;
    }
    const ledger = JSON.parse(fs.readFileSync(ledgerPath, 'utf-8'));
    const failures = [];
    for (const key of Object.keys(measured).sort()) {
        if (!(key in ledger)) {
            failures.push(`${key}: ${measured[key]} B has no line in size-ledger.json`);
        } else if (measured[key] > Math.floor(ledger[key] * LEDGER_SLACK)) {
            failures.push(`${key}: ${measured[key]} B is over its ledger line ${ledger[key]} B (+2% allowed)`);
        }
    }
    if (failures.length > 0) {
        console.error('size ledger exceeded:\n  ' + failures.join('\n  '));
        process.exit(1);
    }
}

async function reportBundleSize(filePath, label) {
    const {statSync} = await import('fs');
    try {
        const stats = statSync(filePath);
        const sizeKB = (stats.size / 1024).toFixed(2);
        console.log(`📦 ${label} size (${basename(filePath)}): ${sizeKB} KB`);
    } catch (err) {
        console.warn(`⚠️ unable to read bundle size for ${filePath}:`, err);
    }
}

async function minifyWithTerser(filePath) {
    const fs = await import('fs');
    const {minify} = await import('terser');

    const code = fs.readFileSync(filePath, 'utf-8');
    const sourceMapPath = `${filePath}.map`;

    const minified = await minify(code, {
        sourceMap: false,
        compress: {
            passes: 3,
            unsafe: true,
            unsafe_math: true,
            unsafe_methods: true,
            unsafe_proto: true,
            unsafe_regexp: true,
            unsafe_comps: true,
            drop_console: true,
            drop_debugger: true,
            pure_funcs: ['console.log', 'console.debug', 'console.info', 'console.warn'],
            pure_getters: true,
            keep_fargs: false,
            booleans_as_integers: true,
            toplevel: true,
            hoist_funs: true,
            hoist_vars: true,
            if_return: true,
            join_vars: true,
            sequences: true,
            properties: true,
            dead_code: true,
            evaluate: true,
            reduce_vars: true,
            collapse_vars: true,
        },
        mangle: {
            toplevel: true,
            properties: {
                regex: /^_/,
            },
        },
        format: {
            comments: false,
            preamble: '// LiveUI Client v1.0.0',
        },
    });

    if (minified.code) {
        fs.writeFileSync(filePath, minified.code);
        await reportBundleSize(filePath, 'terser');
    }
    if (minified.map) {
        fs.writeFileSync(sourceMapPath, minified.map);
    }
}

function resolveConfig(target) {
    return {
        ...baseConfig,
        outfile: target.outfile,
        sourcemap: target.config.sourcemap,
        minify: target.config.minify,
        drop: target.config.drop,
    };
}

async function buildTarget(target) {
    const config = resolveConfig(target);
    const result = await esbuild.build(config);
    if (typeof target.postprocess === 'function') {
        await target.postprocess(result);
    }
}

async function watchTarget(target) {
    const config = resolveConfig(target);
    const ctx = await esbuild.context(config);
    await ctx.watch();
    console.log(`👀 Watching ${target.label} bundle...`);
    return ctx;
}

async function build() {
    try {
        if (isWatch) {
            const contexts = [];
            for (const target of targets) {
                contexts.push(await watchTarget(target));
            }
            console.log('👀 Watching for changes...');
            return contexts;
        } else {
            for (const target of targets) {
                await buildTarget(target);
            }

            console.log('✨ Build complete!');
        }
    } catch (error) {
        console.error('❌ Build failed:', error);
        process.exit(1);
    }
}

build();
