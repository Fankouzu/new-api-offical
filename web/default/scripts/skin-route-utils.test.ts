import assert from 'node:assert/strict'
import {
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rm,
  stat,
  symlink,
  writeFile,
} from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { describe, it } from 'node:test'
import { pathToFileURL } from 'node:url'
import ts from 'typescript'
import { generateSkin } from './generate-skin'
import {
  getGeneratedRouteFile,
  normalizeSkinId,
  renderActiveBuildModule,
  renderActiveSkinModule,
  renderGeneratedRoute,
  validateSkinRoutes,
} from './skin-route-utils'

describe('skin build selection utilities', () => {
  it('defaults missing and empty skin IDs to default', () => {
    assert.equal(normalizeSkinId(undefined), 'default')
    assert.equal(normalizeSkinId(''), 'default')
  })

  it('rejects skin IDs that cannot be used as path segments', () => {
    assert.throws(() => normalizeSkinId('../custom'))
    assert.throws(() => normalizeSkinId('custom skin'))
  })

  it('renders the selected runtime manifest as the activeSkin default export', () => {
    const source = renderActiveSkinModule('custom')

    assert.match(source, /@\/skins\/custom\/manifest/)
    assert.match(source, /export \{ default as activeSkin \}/)
    assert.doesNotMatch(source, /customSkin/)
  })

  it('renders the selected build manifest as the activeSkinBuild default export', () => {
    const source = renderActiveBuildModule('custom')

    assert.match(source, /@\/skins\/custom\/build-manifest/)
    assert.match(source, /export \{ default as activeSkinBuild \}/)
    assert.doesNotMatch(source, /customSkinBuild/)
  })
})

describe('skin route generation utilities', () => {
  it('maps a top-level route to the generated route group', () => {
    assert.equal(getGeneratedRouteFile('/solutions'), '(skin-generated)/solutions.tsx')
  })

  it('keeps dynamic route segments literal', () => {
    assert.equal(
      getGeneratedRouteFile('/guides/$slug'),
      '(skin-generated)/guides/$slug.tsx'
    )
  })

  it('rejects segments reserved by TanStack file routing', () => {
    const rejectedPaths = [
      '/docs/index',
      '/docs/route',
      '/docs.foo',
      '/(group)/docs',
      '/docs[preview]',
      '/_docs',
      '/docs_',
      '/docs/$',
      '/docs/$bad-name',
      '/docs/$9slug',
    ]

    for (const routePath of rejectedPaths) {
      assert.throws(
        () => getGeneratedRouteFile(routePath),
        /invalid skin route file segment/i,
        routePath
      )
    }
  })

  it('accepts conservative static and named dynamic segments', () => {
    assert.equal(
      getGeneratedRouteFile('/guides-v2/$model_id'),
      '(skin-generated)/guides-v2/$model_id.tsx'
    )
  })

  it('rejects duplicate route IDs', () => {
    assert.throws(
      () =>
        validateSkinRoutes([
          { id: 'solutions', path: '/solutions', componentImport: './routes/solutions' },
          { id: 'solutions', path: '/guides', componentImport: './routes/guides' },
        ]),
      /duplicate skin route id: solutions/i
    )
  })

  it('rejects duplicate route paths', () => {
    assert.throws(
      () =>
        validateSkinRoutes([
          { id: 'solutions', path: '/solutions', componentImport: './routes/solutions' },
          { id: 'other', path: '/solutions', componentImport: './routes/other' },
        ]),
      /duplicate skin route path: \/solutions/i
    )
  })

  it('rejects effective generated file collisions', () => {
    assert.throws(
      () =>
        validateSkinRoutes([
          { id: 'upper', path: '/Docs', componentImport: './routes/upper' },
          { id: 'lower', path: '/docs', componentImport: './routes/lower' },
        ]),
      /generated route file collision/i
    )
  })

  it('rejects host-owned routes through the shared route policy', () => {
    assert.throws(
      () =>
        validateSkinRoutes([
          { id: 'pricing', path: '/pricing', componentImport: './routes/pricing' },
        ]),
      /host-owned path: \/pricing/i
    )
  })

  it('rejects unsafe and non-canonical routes through the shared route policy', () => {
    assert.throws(
      () =>
        validateSkinRoutes([
          { id: 'unsafe', path: '/guides/../pricing', componentImport: './routes/unsafe' },
        ]),
      /invalid skin route path/i
    )
  })

  it('rejects empty route IDs and component imports', () => {
    assert.throws(
      () => validateSkinRoutes([{ id: '', path: '/solutions', componentImport: './route' }]),
      /empty skin route id/i
    )
    assert.throws(
      () => validateSkinRoutes([{ id: 'solutions', path: '/solutions', componentImport: '' }]),
      /empty component import/i
    )
  })

  it('renders a route proxy through ActiveSkinRoute', () => {
    const source = renderGeneratedRoute({
      id: 'solutions',
      path: '/solutions',
      componentImport: './routes/solutions',
    })

    assert.ok(source.includes("createFileRoute('/(skin-generated)/solutions')"))
    assert.ok(source.includes("from '@/skins/runtime/skin-route'"))
    assert.ok(source.includes('routeId={"solutions"}'))
    assert.doesNotMatch(source, /\.\/routes\/solutions/)
  })

  it('resolves the generated proxy runtime import through TypeScript', () => {
    const source = renderGeneratedRoute({
      id: 'solutions',
      path: '/solutions',
      componentImport: './routes/solutions',
    })
    const runtimeImport = source.match(
      /from ['"](@\/skins\/runtime\/skin-route)['"]/
    )?.[1]
    const containingFile = path.join(
      process.cwd(),
      'src',
      'routes',
      '(skin-generated)',
      'solutions.tsx'
    )

    assert.equal(runtimeImport, '@/skins/runtime/skin-route')
    const resolution = ts.resolveModuleName(
      runtimeImport,
      containingFile,
      {
        baseUrl: process.cwd(),
        moduleResolution: ts.ModuleResolutionKind.Bundler,
        paths: { '@/*': ['./src/*'] },
      },
      ts.sys
    )

    assert.equal(
      resolution.resolvedModule?.resolvedFileName,
      path.join(process.cwd(), 'src', 'skins', 'runtime', 'skin-route.tsx')
    )
  })

  it('renders quote and control characters in route IDs as safe JSX expressions', () => {
    const source = renderGeneratedRoute({
      id: "quote'\nline",
      path: '/safe-id',
      componentImport: './routes/safe-id',
    })

    assert.ok(source.includes(`routeId={${JSON.stringify("quote'\nline")}}`))
    assert.doesNotMatch(source, /routeId='quote'/)
  })

  it('returns routes sorted by path without mutating the input', () => {
    const routes = [
      { id: 'solutions', path: '/solutions' as const, componentImport: './routes/solutions' },
      { id: 'guides', path: '/guides' as const, componentImport: './routes/guides' },
    ]

    const validated = validateSkinRoutes(routes)

    assert.deepEqual(
      validated.map((route) => route.path),
      ['/guides', '/solutions']
    )
    assert.deepEqual(
      routes.map((route) => route.path),
      ['/solutions', '/guides']
    )
  })
})

async function createSkinProjectFixture(): Promise<{
  projectRoot: URL
  rootPath: string
  skinDirectory: string
  runtimeDirectory: string
}> {
  const rootPath = await mkdtemp(path.join(os.tmpdir(), 'generate-skin-'))
  const skinDirectory = path.join(rootPath, 'src', 'skins', 'custom')
  const runtimeDirectory = path.join(rootPath, 'src', 'skins', 'runtime')
  await Promise.all([
    mkdir(skinDirectory, { recursive: true }),
    mkdir(runtimeDirectory, { recursive: true }),
  ])

  return {
    projectRoot: pathToFileURL(`${rootPath}${path.sep}`),
    rootPath,
    skinDirectory,
    runtimeDirectory,
  }
}

describe('skin generator', () => {
  it('validates runtime then build manifests without creating outputs', async () => {
    const fixture = await createSkinProjectFixture()
    try {
      await assert.rejects(
        generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' }),
        /missing required file: src\/skins\/custom\/manifest\.ts/
      )
      assert.deepEqual(await readdir(fixture.runtimeDirectory), [])

      await writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n')
      await assert.rejects(
        generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' }),
        /missing required file: src\/skins\/custom\/build-manifest\.ts/
      )
      assert.deepEqual(await readdir(fixture.runtimeDirectory), [])
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('replaces stale generated modules as a pair', async () => {
    const fixture = await createSkinProjectFixture()
    try {
      await Promise.all([
        writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n'),
        writeFile(
          path.join(fixture.skinDirectory, 'build-manifest.ts'),
          "export default { id: 'custom', routes: [] }\n"
        ),
        writeFile(path.join(fixture.runtimeDirectory, 'active-skin.gen.ts'), 'stale\n'),
        writeFile(path.join(fixture.runtimeDirectory, 'active-skin-build.gen.ts'), 'stale\n'),
      ])

      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })

      assert.equal(
        await readFile(path.join(fixture.runtimeDirectory, 'active-skin.gen.ts'), 'utf8'),
        renderActiveSkinModule('custom')
      )
      assert.equal(
        await readFile(path.join(fixture.runtimeDirectory, 'active-skin-build.gen.ts'), 'utf8'),
        renderActiveBuildModule('custom')
      )
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('does not rewrite unchanged generated modules', async () => {
    const fixture = await createSkinProjectFixture()
    try {
      await Promise.all([
        writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n'),
        writeFile(
          path.join(fixture.skinDirectory, 'build-manifest.ts'),
          "export default { id: 'custom', routes: [] }\n"
        ),
      ])
      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })

      const runtimePath = path.join(fixture.runtimeDirectory, 'active-skin.gen.ts')
      const buildPath = path.join(fixture.runtimeDirectory, 'active-skin-build.gen.ts')
      const before = await Promise.all([stat(runtimePath), stat(buildPath)])
      await new Promise((resolve) => setTimeout(resolve, 50))

      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })

      const after = await Promise.all([stat(runtimePath), stat(buildPath)])
      assert.equal(after[0].mtimeMs, before[0].mtimeMs)
      assert.equal(after[1].mtimeMs, before[1].mtimeMs)
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('regenerates nested proxies while preserving files outside generated ownership', async () => {
    const fixture = await createSkinProjectFixture()
    const generatedDirectory = path.join(fixture.rootPath, 'src', 'routes', '(skin-generated)')
    try {
      await Promise.all([
        mkdir(path.join(generatedDirectory, 'stale'), { recursive: true }),
        writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n'),
        writeFile(
          path.join(fixture.skinDirectory, 'build-manifest.ts'),
          `export default {
  id: 'custom',
  routes: [
    { id: 'solutions', path: '/solutions', componentImport: './routes/solutions' },
    { id: 'guide', path: '/guides/$slug', componentImport: './routes/guide' },
  ],
}\n`
        ),
      ])
      await Promise.all([
        writeFile(path.join(generatedDirectory, '.gitignore'), '*.tsx\n!/.gitignore\n'),
        writeFile(path.join(generatedDirectory, 'notes.txt'), 'preserve\n'),
        writeFile(path.join(generatedDirectory, 'stale', 'old.tsx'), 'stale\n'),
        writeFile(path.join(generatedDirectory, 'stale', 'keep.json'), '{}\n'),
      ])

      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })

      assert.match(
        await readFile(path.join(generatedDirectory, 'guides', '$slug.tsx'), 'utf8'),
        /routeId={"guide"}/
      )
      assert.match(
        await readFile(path.join(generatedDirectory, 'solutions.tsx'), 'utf8'),
        /routeId={"solutions"}/
      )
      await assert.rejects(readFile(path.join(generatedDirectory, 'stale', 'old.tsx'), 'utf8'))
      assert.equal(await readFile(path.join(generatedDirectory, '.gitignore'), 'utf8'), '*.tsx\n!/.gitignore\n')
      assert.equal(await readFile(path.join(generatedDirectory, 'notes.txt'), 'utf8'), 'preserve\n')
      assert.equal(await readFile(path.join(generatedDirectory, 'stale', 'keep.json'), 'utf8'), '{}\n')
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('does not replace unchanged route proxies', async () => {
    const fixture = await createSkinProjectFixture()
    const generatedRoutePath = path.join(
      fixture.rootPath,
      'src',
      'routes',
      '(skin-generated)',
      'solutions.tsx'
    )
    try {
      await Promise.all([
        writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n'),
        writeFile(
          path.join(fixture.skinDirectory, 'build-manifest.ts'),
          `export default {
  id: 'custom',
  routes: [{ id: 'solutions', path: '/solutions', componentImport: './routes/solutions' }],
}\n`
        ),
      ])
      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })
      const before = await stat(generatedRoutePath)
      await new Promise((resolve) => setTimeout(resolve, 50))

      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })

      const after = await stat(generatedRoutePath)
      assert.equal(after.mtimeMs, before.mtimeMs)
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('rejects symlinks without touching the previous generated tree', async () => {
    const fixture = await createSkinProjectFixture()
    const generatedDirectory = path.join(fixture.rootPath, 'src', 'routes', '(skin-generated)')
    try {
      await Promise.all([
        mkdir(path.join(generatedDirectory, 'nested'), { recursive: true }),
        writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n'),
        writeFile(
          path.join(fixture.skinDirectory, 'build-manifest.ts'),
          "export default { id: 'custom', routes: [] }\n"
        ),
      ])
      await writeFile(path.join(generatedDirectory, 'previous.tsx'), 'previous\n')
      await symlink(path.join(generatedDirectory, 'previous.tsx'), path.join(generatedDirectory, 'nested', 'link'))

      await assert.rejects(
        generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' }),
        /symlink.*skin-generated/i
      )

      assert.equal(await readFile(path.join(generatedDirectory, 'previous.tsx'), 'utf8'), 'previous\n')
      assert.equal(await readFile(path.join(generatedDirectory, 'nested', 'link'), 'utf8'), 'previous\n')
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('rolls back the previous generated tree when staged activation fails', async () => {
    const fixture = await createSkinProjectFixture()
    const generatedDirectory = path.join(fixture.rootPath, 'src', 'routes', '(skin-generated)')
    try {
      await Promise.all([
        mkdir(generatedDirectory, { recursive: true }),
        writeFile(path.join(fixture.skinDirectory, 'manifest.ts'), 'export default {}\n'),
        writeFile(
          path.join(fixture.skinDirectory, 'build-manifest.ts'),
          `export default {
  id: 'custom',
  routes: [{ id: 'solutions', path: '/solutions', componentImport: './routes/solutions' }],
}\n`
        ),
      ])
      await writeFile(path.join(generatedDirectory, 'previous.tsx'), 'previous\n')

      await assert.rejects(
        generateSkin({
          projectRoot: fixture.projectRoot,
          skinId: 'custom',
          afterRouteBackup: async () => {
            throw new Error('injected activation failure')
          },
        }),
        /injected activation failure/
      )

      assert.equal(await readFile(path.join(generatedDirectory, 'previous.tsx'), 'utf8'), 'previous\n')
      await assert.rejects(readFile(path.join(generatedDirectory, 'solutions.tsx'), 'utf8'))
      assert.deepEqual(await readdir(fixture.runtimeDirectory), [])
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })
})
