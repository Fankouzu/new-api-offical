import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { readFile, readdir } from 'node:fs/promises'
import path from 'node:path'
import { after, describe, it } from 'node:test'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'

const execFileAsync = promisify(execFile)
const frontendRoot = fileURLToPath(new URL('../', import.meta.url))
const packageJsonPath = path.join(frontendRoot, 'package.json')
const runtimeDirectory = path.join(frontendRoot, 'src', 'skins', 'runtime')
const generatedRoutesDirectory = path.join(
  frontendRoot,
  'src',
  'routes',
  '(skin-generated)'
)
const selectorPaths = [
  path.join(runtimeDirectory, 'active-skin.gen.ts'),
  path.join(runtimeDirectory, 'active-skin-build.gen.ts'),
]

async function runBun(args: string[], env: NodeJS.ProcessEnv = {}) {
  return execFileAsync('bun', args, {
    cwd: frontendRoot,
    env: { ...process.env, ...env },
  })
}

async function inspectRsbuild(env: NodeJS.ProcessEnv = {}) {
  return execFileAsync('bunx', ['rsbuild', 'inspect', '--mode', 'development'], {
    cwd: frontendRoot,
    env: { ...process.env, ...env },
  })
}

async function readGeneratedState() {
  return {
    selectors: await Promise.all(selectorPaths.map((filePath) => readFile(filePath, 'utf8'))),
    routes: (await readdir(generatedRoutesDirectory)).sort(),
  }
}

after(async () => {
  await runBun(['run', 'skin:generate'], { APP_SKIN: 'default' })
})

describe('skin build lifecycle', { concurrency: false }, () => {
  it('keeps the required package script contract', async () => {
    const packageJson = JSON.parse(await readFile(packageJsonPath, 'utf8')) as {
      scripts: Record<string, string>
    }

    assert.equal(packageJson.scripts['skin:generate'], 'bun scripts/generate-skin.ts')
    assert.equal(packageJson.scripts.dev, 'rsbuild dev')
    assert.equal(packageJson.scripts.build, 'rsbuild build')
    assert.equal(packageJson.scripts.typecheck, 'bun run skin:generate && tsc -b')
    assert.equal(packageJson.scripts['build:check'], 'bun run typecheck && rsbuild build')
    assert.equal(packageJson.scripts['build:default'], 'APP_SKIN=default bun run build:check')
    assert.equal(packageJson.scripts['build:custom'], 'APP_SKIN=custom bun run build:check')
    assert.match(packageJson.scripts['test:unit'], /"scripts\/\*\*\/\*\.test\.ts"/)
  })

  it('switches selectors deterministically while empty manifests keep routes clean', async () => {
    await runBun(['run', 'skin:generate'], { APP_SKIN: 'default' })
    const firstDefault = await readGeneratedState()

    await runBun(['run', 'skin:generate'], { APP_SKIN: 'custom' })
    const custom = await readGeneratedState()

    await runBun(['run', 'skin:generate'], { APP_SKIN: 'default' })
    const secondDefault = await readGeneratedState()

    assert.deepEqual(secondDefault, firstDefault)
    assert.match(custom.selectors[0], /@\/skins\/custom\/manifest/)
    assert.match(custom.selectors[1], /@\/skins\/custom\/build-manifest/)
    assert.deepEqual(firstDefault.routes, ['.gitignore'])
    assert.deepEqual(custom.routes, ['.gitignore'])
  })

  it('rejects an unknown skin without altering prior generated outputs', async () => {
    await runBun(['run', 'skin:generate'], { APP_SKIN: 'custom' })
    const before = await readGeneratedState()

    await assert.rejects(
      runBun(['run', 'skin:generate'], { APP_SKIN: 'missing' }),
      (error: unknown) => {
        assert.ok(error instanceof Error)
        assert.match(
          'stderr' in error && typeof error.stderr === 'string' ? error.stderr : '',
          /Skin "missing" is missing required file/
        )
        return true
      }
    )

    assert.deepEqual(await readGeneratedState(), before)
  })

  it('generates the selected aliases while Rsbuild evaluates its config', async () => {
    await runBun(['run', 'skin:generate'], { APP_SKIN: 'default' })

    await inspectRsbuild({ APP_SKIN: 'custom' })

    const state = await readGeneratedState()
    assert.match(state.selectors[0], /@\/skins\/custom\/manifest/)
    assert.match(state.selectors[1], /@\/skins\/custom\/build-manifest/)
    assert.deepEqual(state.routes, ['.gitignore'])
  })
})
