import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { describe, it } from 'node:test'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { generateSkin } from './generate-skin'

const frontendRoot = fileURLToPath(new URL('../', import.meta.url))
const packageJsonPath = path.join(frontendRoot, 'package.json')
const skinWrapperPath = path.join(frontendRoot, 'scripts', 'run-with-skin.ts')
const dockerfilePath = path.join(frontendRoot, '..', '..', 'Dockerfile')

async function createLifecycleFixture() {
  const rootPath = await mkdtemp(path.join(os.tmpdir(), 'skin-lifecycle-'))
  const runtimeDirectory = path.join(rootPath, 'src', 'skins', 'runtime')
  const routesDirectory = path.join(rootPath, 'src', 'routes', '(skin-generated)')
  await Promise.all([
    mkdir(runtimeDirectory, { recursive: true }),
    mkdir(routesDirectory, { recursive: true }),
  ])
  for (const skinId of ['default', 'custom']) {
    const skinDirectory = path.join(rootPath, 'src', 'skins', skinId)
    await mkdir(skinDirectory, { recursive: true })
    await Promise.all([
      writeFile(
        path.join(skinDirectory, 'manifest.ts'),
        `const build = { id: '${skinId}', routes: [] }; export default { id: '${skinId}', build, pages: {}, routes: [] }\n`
      ),
      writeFile(
        path.join(skinDirectory, 'build-manifest.ts'),
        `export default { id: '${skinId}', routes: [] }\n`
      ),
    ])
  }
  await writeFile(path.join(routesDirectory, '.gitignore'), '*.tsx\n')

  const readState = async () => ({
    runtime: await Promise.all([
      readFile(path.join(runtimeDirectory, 'active-skin.gen.ts'), 'utf8'),
      readFile(path.join(runtimeDirectory, 'active-skin-build.gen.ts'), 'utf8'),
    ]),
    routes: (await readdir(routesDirectory)).sort(),
  })

  return {
    projectRoot: pathToFileURL(`${rootPath}${path.sep}`),
    readState,
    rootPath,
  }
}

describe('skin build lifecycle', { concurrency: false }, () => {
  it('keeps the required package script contract', async () => {
    const packageJson = JSON.parse(await readFile(packageJsonPath, 'utf8')) as {
      scripts: Record<string, string>
    }

    assert.equal(packageJson.scripts['skin:generate'], 'bun scripts/run-with-skin.ts generate')
    assert.equal(packageJson.scripts.dev, 'rsbuild dev')
    assert.equal(packageJson.scripts.build, 'bun scripts/run-with-skin.ts build')
    assert.equal(packageJson.scripts.typecheck, 'bun scripts/run-with-skin.ts typecheck')
    assert.equal(packageJson.scripts['build:check'], 'bun run build')
    assert.equal(packageJson.scripts['build:default'], 'APP_SKIN=default bun run build')
    assert.equal(packageJson.scripts['build:custom'], 'APP_SKIN=custom bun run build')
    assert.match(packageJson.scripts['test:unit'], /"scripts\/\*\*\/\*\.test\.ts"/)
    assert.match(packageJson.scripts['test:unit'], /--test-concurrency=1/)
    assert.doesNotMatch(
      await readFile(skinWrapperPath, 'utf8'),
      new RegExp(['SKIN', 'LOCK', 'PROJECT', 'ROOT'].join('_'))
    )
  })

  it('passes the selected skin into the Docker frontend build', async () => {
    const dockerfile = await readFile(dockerfilePath, 'utf8')

    assert.match(dockerfile, /ARG APP_SKIN=default/)
    assert.match(
      dockerfile,
      /APP_SKIN="\$\{APP_SKIN\}"[\s\S]*?bun run build/
    )
  })

  it('switches default to custom and back deterministically in an isolated project', async () => {
    const fixture = await createLifecycleFixture()
    try {
      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'default' })
      const firstDefault = await fixture.readState()
      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })
      const custom = await fixture.readState()
      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'default' })

      assert.deepEqual(await fixture.readState(), firstDefault)
      assert.match(custom.runtime[0], /@\/skins\/custom\/manifest/)
      assert.match(custom.runtime[1], /@\/skins\/custom\/build-manifest/)
      assert.deepEqual(custom.routes, ['.gitignore'])
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })

  it('keeps isolated generated state unchanged when the requested skin is missing', async () => {
    const fixture = await createLifecycleFixture()
    try {
      await generateSkin({ projectRoot: fixture.projectRoot, skinId: 'custom' })
      const before = await fixture.readState()

      await assert.rejects(
        generateSkin({ projectRoot: fixture.projectRoot, skinId: 'missing' }),
        /Skin "missing" is missing required file/
      )

      assert.deepEqual(await fixture.readState(), before)
    } finally {
      await rm(fixture.rootPath, { recursive: true, force: true })
    }
  })
})
