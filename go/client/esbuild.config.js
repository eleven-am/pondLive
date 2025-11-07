import * as esbuild from 'esbuild';
import {fileURLToPath} from 'url';
import {basename, dirname, resolve} from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const isProduction = process.env.NODE_ENV === 'production';
const isWatch = process.argv.includes('--watch');

const outDir = resolve(__dirname, '../pkg/live/server/static');

const baseConfig = {
    entryPoints: [resolve(__dirname, 'src/entry.ts')],
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
    footer: {
        js: `
// Export to global window object
if (typeof window !== 'undefined') {
  window.LiveUI = LiveUIModule.default;
  window.LiveUI.dom = LiveUIModule.dom;
  window.LiveUI.applyOps = LiveUIModule.applyOps;
}
`,
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
            passes: 2,
            unsafe: true,
            unsafe_math: true,
            unsafe_methods: true,
            drop_console: true,
        },
        mangle: {
            toplevel: true,
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
