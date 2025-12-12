import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['cjs', 'esm'],
  dts: true,
  splitting: false,
  sourcemap: true,
  clean: true,
  outDir: 'dist',
  outExtension({ format }) {
    return {
      js: format === 'cjs' ? '.js' : '.js',
    };
  },
  esbuildOptions(options, context) {
    if (context.format === 'cjs') {
      options.outdir = 'dist/cjs';
    } else {
      options.outdir = 'dist/esm';
    }
  },
});
