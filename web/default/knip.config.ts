import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  entry: [
    'src/main.tsx',
    'scripts/skin-lifecycle.test.ts',
    'scripts/skin-workspace-lock.test.ts',
  ],
  ignore: ['src/components/ui/**', 'src/routeTree.gen.ts'],
  ignoreDependencies: ['tailwindcss', 'tw-animate-css'],
}

export default config
