import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['cjs', 'esm'],
  dts: true,
  sourcemap: true,
  clean: true,
  splitting: false,
  treeshake: true,
  minify: false, // Keep readable for debugging
  target: 'node18',
  outDir: 'dist',
  external: [],
  bundle: true,
  skipNodeModulesBundle: true,
});
