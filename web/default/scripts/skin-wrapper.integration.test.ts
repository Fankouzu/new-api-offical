import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chmod, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { after, describe, it } from 'node:test'
import { fileURLToPath } from 'node:url'

const frontendRoot = fileURLToPath(new URL('../', import.meta.url))
const lockDirectory = path.join(
  frontendRoot,
  'node_modules',
  '.cache',
  'frontend-skin-workspace.lock'
)
const timeoutMs = 8_000

function waitForExit(child: ReturnType<typeof spawn>) {
  return new Promise<{ code: number | null; signal: NodeJS.Signals | null }>(
    (resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill('SIGKILL')
        reject(new Error(`Subprocess timed out after ${timeoutMs}ms`))
      }, timeoutMs)
      child.once('error', reject)
      child.once('close', (code, signal) => {
        clearTimeout(timeout)
        resolve({ code, signal })
      })
    }
  )
}

async function waitForFile(filePath: string): Promise<string> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      return await readFile(filePath, 'utf8')
    } catch {
      await new Promise((resolve) => setTimeout(resolve, 25))
    }
  }
  throw new Error(`Timed out waiting for ${filePath}`)
}

function spawnWrapper(skinId: string, typecheckBin: string, killTimeoutMs = 2_000) {
  return spawn('bun', ['scripts/run-with-skin.ts', 'typecheck'], {
    cwd: frontendRoot,
    env: {
      ...process.env,
      APP_SKIN: skinId,
      SKIN_TYPECHECK_BIN: typecheckBin,
      SKIN_CHILD_KILL_TIMEOUT_MS: String(killTimeoutMs),
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  })
}

type ExecutableFixture = {
  childPidPath: string
  executablePath: string
}

async function createExecutable(
  directory: string,
  source: string
): Promise<ExecutableFixture> {
  const filePath = path.join(directory, `child-${Math.random().toString(16).slice(2)}.mjs`)
  const childPidPath = `${filePath}.pid`
  await writeFile(
    filePath,
    `#!/usr/bin/env node
import { writeFileSync as writeFixturePid } from 'node:fs'
writeFixturePid(${JSON.stringify(childPidPath)}, String(process.pid))
setTimeout(() => process.exit(124), 6000)
${source}`,
    'utf8'
  )
  await chmod(filePath, 0o755)
  return { childPidPath, executablePath: filePath }
}

async function killFixtureProcessGroup(childPidPath: string | undefined): Promise<void> {
  if (!childPidPath) return
  let childPid: number
  try {
    childPid = Number(await readFile(childPidPath, 'utf8'))
  } catch {
    return
  }

  try {
    process.kill(process.platform === 'win32' ? childPid : -childPid, 'SIGKILL')
  } catch (error: unknown) {
    if (!(error instanceof Error && 'code' in error && error.code === 'ESRCH')) throw error
  }

  const deadline = Date.now() + 2_000
  while (Date.now() < deadline) {
    try {
      process.kill(childPid, 0)
      await new Promise((resolve) => setTimeout(resolve, 25))
    } catch (error: unknown) {
      if (error instanceof Error && 'code' in error && error.code === 'ESRCH') return
      throw error
    }
  }
  throw new Error(`Fixture child PID ${childPid} did not exit after SIGKILL`)
}

after(async () => {
  const restore = spawn('bun', ['run', 'skin:generate'], {
    cwd: frontendRoot,
    env: { ...process.env, APP_SKIN: 'default' },
    stdio: 'ignore',
  })
  assert.deepEqual(await waitForExit(restore), { code: 0, signal: null })
})

describe('skin wrapper subprocess lifecycle', { concurrency: false }, () => {
  it('keeps the lease until the signaled child exits, then preserves signal semantics', async () => {
    const fixture = await mkdtemp(path.join(os.tmpdir(), 'skin-wrapper-'))
    const startedPath = path.join(fixture, 'started')
    const signaledPath = path.join(fixture, 'signaled')
    let wrapper: ReturnType<typeof spawn> | undefined
    let wrapperExit: ReturnType<typeof waitForExit> | undefined
    let childPidPath: string | undefined
    try {
      const childFixture = await createExecutable(
        fixture,
        `import { writeFileSync } from 'node:fs'
writeFileSync(${JSON.stringify(startedPath)}, 'started')
process.on('SIGTERM', () => {
  writeFileSync(${JSON.stringify(signaledPath)}, 'SIGTERM')
  setTimeout(() => process.exit(23), 600)
})
setInterval(() => {}, 1000)
`
      )
      childPidPath = childFixture.childPidPath
      wrapper = spawnWrapper('default', childFixture.executablePath)
      wrapperExit = waitForExit(wrapper)
      await waitForFile(startedPath)
      assert.ok(await readFile(path.join(lockDirectory, 'owner.json'), 'utf8'))

      wrapper.kill('SIGTERM')
      assert.equal(await waitForFile(signaledPath), 'SIGTERM')

      const contender = spawn('bun', ['run', 'skin:generate'], {
        cwd: frontendRoot,
        env: { ...process.env, APP_SKIN: 'custom' },
        stdio: ['ignore', 'pipe', 'pipe'],
      })
      let contenderStderr = ''
      contender.stderr?.on('data', (chunk) => (contenderStderr += String(chunk)))
      const contenderResult = await waitForExit(contender)
      assert.equal(contenderResult.code, 1)
      assert.match(contenderStderr, /already owned by skin "default"/)

      assert.deepEqual(await wrapperExit, { code: null, signal: 'SIGTERM' })
      await assert.rejects(readFile(path.join(lockDirectory, 'owner.json'), 'utf8'))
    } finally {
      await killFixtureProcessGroup(childPidPath)
      if (wrapper && wrapper.exitCode === null && wrapper.signalCode === null) {
        wrapper.kill('SIGKILL')
      }
      await wrapperExit?.catch(() => undefined)
      await rm(fixture, { recursive: true, force: true })
    }
  })

  it('propagates a normal nonzero child exit code', async () => {
    const fixture = await mkdtemp(path.join(os.tmpdir(), 'skin-wrapper-'))
    let wrapper: ReturnType<typeof spawn> | undefined
    let wrapperExit: ReturnType<typeof waitForExit> | undefined
    let childPidPath: string | undefined
    try {
      const childFixture = await createExecutable(fixture, 'process.exit(17)\n')
      childPidPath = childFixture.childPidPath
      wrapper = spawnWrapper('default', childFixture.executablePath)
      wrapperExit = waitForExit(wrapper)

      assert.deepEqual(await wrapperExit, { code: 17, signal: null })
    } finally {
      await killFixtureProcessGroup(childPidPath)
      if (wrapper && wrapper.exitCode === null && wrapper.signalCode === null) {
        wrapper.kill('SIGKILL')
      }
      await wrapperExit?.catch(() => undefined)
      await rm(fixture, { recursive: true, force: true })
    }
  })

  it('kills a child that does not exit before the signal timeout', async () => {
    const fixture = await mkdtemp(path.join(os.tmpdir(), 'skin-wrapper-'))
    const startedPath = path.join(fixture, 'started')
    const signaledPath = path.join(fixture, 'signaled')
    let wrapper: ReturnType<typeof spawn> | undefined
    let wrapperExit: ReturnType<typeof waitForExit> | undefined
    let childPidPath: string | undefined
    try {
      const childFixture = await createExecutable(
        fixture,
        `import { writeFileSync } from 'node:fs'
writeFileSync(${JSON.stringify(startedPath)}, 'started')
process.on('SIGTERM', () => writeFileSync(${JSON.stringify(signaledPath)}, 'SIGTERM'))
setInterval(() => {}, 1000)
`
      )
      childPidPath = childFixture.childPidPath
      wrapper = spawnWrapper('default', childFixture.executablePath, 200)
      wrapperExit = waitForExit(wrapper)
      await waitForFile(startedPath)

      const signaledAt = Date.now()
      wrapper.kill('SIGTERM')
      assert.equal(await waitForFile(signaledPath), 'SIGTERM')
      assert.deepEqual(await wrapperExit, { code: null, signal: 'SIGTERM' })
      assert.ok(Date.now() - signaledAt >= 150)
      await assert.rejects(readFile(path.join(lockDirectory, 'owner.json'), 'utf8'))
    } finally {
      await killFixtureProcessGroup(childPidPath)
      if (wrapper && wrapper.exitCode === null && wrapper.signalCode === null) {
        wrapper.kill('SIGKILL')
      }
      await wrapperExit?.catch(() => undefined)
      await rm(fixture, { recursive: true, force: true })
    }
  })
})
