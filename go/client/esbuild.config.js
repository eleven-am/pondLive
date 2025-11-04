import * as esbuild from 'esbuild';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const isProduction = process.env.NODE_ENV === 'production';
const isWatch = process.argv.includes('--watch');

const baseConfig = {
  entryPoints: [resolve(__dirname, 'src/entry.ts')],
  bundle: true,
  outfile: resolve(__dirname, '../pkg/live/server/static/pondlive.js'),
  format: 'iife',
  platform: 'browser',
  target: ['es2020'],
  globalName: 'LiveUIModule',
  minify: isProduction,
  minifyWhitespace: true,
  minifyIdentifiers: true,
  minifySyntax: true,
  sourcemap: !isProduction,
  treeShaking: true,
  metafile: true,
  legalComments: 'none',
  drop: isProduction ? ['console', 'debugger'] : [],
  // Resolve browser version of pondsocket-client
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

async function build() {
  try {
    if (isWatch) {
      const ctx = await esbuild.context(baseConfig);
      await ctx.watch();
      console.log('👀 Watching for changes...');
    } else {
      const result = await esbuild.build(baseConfig);

      // Print bundle size before terser
      const fs = await import('fs');
      let stats = fs.statSync(baseConfig.outfile);
      let sizeKB = (stats.size / 1024).toFixed(2);
      console.log(`📦 esbuild size: ${sizeKB} KB`);

      // Further minify with terser for production
      if (isProduction) {
        const { minify } = await import('terser');
        const code = fs.readFileSync(baseConfig.outfile, 'utf-8');
        const minified = await minify(code, {
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
          fs.writeFileSync(baseConfig.outfile, minified.code);
          stats = fs.statSync(baseConfig.outfile);
          sizeKB = (stats.size / 1024).toFixed(2);
          console.log(`📦 terser size: ${sizeKB} KB`);
        }
      }

      console.log(`✨ Build complete!`);

      if (result.metafile) {
        console.log('📊 Analysis:', await esbuild.analyzeMetafile(result.metafile));
      }
    }
  } catch (error) {
    console.error('❌ Build failed:', error);
    process.exit(1);
  }
}

build();
