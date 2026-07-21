import assert from 'node:assert/strict'
import {
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rm,
  stat,
  writeFile,
} from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { describe, it } from 'node:test'
import { pathToFileURL } from 'node:url'
import { generateSkin } from './generate-skin'
import {
  normalizeSkinId,
  renderActiveBuildModule,
  renderActiveSkinModule,
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
          'export default {}\n'
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
          'export default {}\n'
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
})
