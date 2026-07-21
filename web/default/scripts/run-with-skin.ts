import { spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { generateSkin } from './generate-skin'
import { normalizeSkinId } from './skin-route-utils'
import { acquireSkinWorkspaceLease } from './skin-workspace-lock'

const projectRoot = fileURLToPath(new URL('../', import.meta.url))

type ChildResult = {
  code: number | null
  signal: NodeJS.Signals | null
  terminationSignal: NodeJS.Signals | null
}

function forwardSignal(child: ReturnType<typeof spawn>, signal: NodeJS.Signals): void {
  if (child.pid === undefined) return
  try {
    if (process.platform !== 'win32') {
      process.kill(-child.pid, signal)
    } else {
      child.kill(signal)
    }
  } catch (error: unknown) {
    if (!(error instanceof Error && 'code' in error && error.code === 'ESRCH')) throw error
  }
}

async function runChild(
  executablePath: string,
  args: string[],
  env?: NodeJS.ProcessEnv
): Promise<ChildResult> {
  const child = spawn(executablePath, args, {
    cwd: projectRoot,
    detached: process.platform !== 'win32',
    env: { ...process.env, ...env },
    stdio: 'inherit',
  })
  const configuredTimeout = Number(process.env.SKIN_CHILD_KILL_TIMEOUT_MS)
  const killTimeoutMs = Number.isFinite(configuredTimeout) && configuredTimeout > 0
    ? configuredTimeout
    : 5_000

  return new Promise((resolve, reject) => {
    const signals: NodeJS.Signals[] = ['SIGINT', 'SIGTERM', 'SIGHUP']
    let terminationSignal: NodeJS.Signals | null = null
    let killTimer: NodeJS.Timeout | undefined
    const handlers = new Map<NodeJS.Signals, () => void>()
    const removeHandlers = () => {
      for (const [signal, handler] of handlers) process.off(signal, handler)
    }

    for (const signal of signals) {
      const handler = () => {
        if (terminationSignal) {
          forwardSignal(child, 'SIGKILL')
          return
        }
        terminationSignal = signal
        forwardSignal(child, signal)
        killTimer = setTimeout(() => forwardSignal(child, 'SIGKILL'), killTimeoutMs)
        killTimer.unref()
      }
      handlers.set(signal, handler)
      process.on(signal, handler)
    }

    child.once('error', (error) => {
      if (killTimer) clearTimeout(killTimer)
      removeHandlers()
      reject(error)
    })
    child.once('close', (code, signal) => {
      if (killTimer) clearTimeout(killTimer)
      removeHandlers()
      resolve({ code, signal, terminationSignal })
    })
  })
}

async function main(): Promise<void> {
  const mode = process.argv[2]
  if (mode !== 'generate' && mode !== 'typecheck' && mode !== 'build') {
    throw new Error(`Unknown skin command mode: ${mode ?? '(missing)'}`)
  }

  const skinId = normalizeSkinId(process.env.APP_SKIN)
  const lockProjectRoot = process.env.SKIN_LOCK_PROJECT_ROOT || projectRoot
  const lease = await acquireSkinWorkspaceLease({
    projectRoot: lockProjectRoot,
    skinId,
  })
  const onExit = () => lease.releaseSync()
  process.once('exit', onExit)
  let exitSignal: NodeJS.Signals | null = null

  try {
    await generateSkin({ skinId })
    if (mode === 'typecheck' || mode === 'build') {
      const tscPath =
        process.env.SKIN_TYPECHECK_BIN ||
        path.join(projectRoot, 'node_modules', '.bin', 'tsc')
      const result = await runChild(tscPath, ['-b'])
      exitSignal = result.terminationSignal ?? result.signal
      if (!exitSignal) process.exitCode = result.code ?? 1
      if (!exitSignal && result.code === 0 && mode === 'build') {
        const rsbuildPath =
          process.env.SKIN_BUILD_BIN ||
          path.join(projectRoot, 'node_modules', '.bin', 'rsbuild')
        const buildResult = await runChild(rsbuildPath, ['build'], {
          SKIN_WORKSPACE_LEASE_OWNER_PID: String(process.pid),
          SKIN_WORKSPACE_LEASE_PROJECT_ROOT: lockProjectRoot,
          SKIN_WORKSPACE_LEASE_TOKEN: lease.token,
        })
        exitSignal = buildResult.terminationSignal ?? buildResult.signal
        if (!exitSignal) process.exitCode = buildResult.code ?? 1
      }
    }
  } finally {
    await lease.release()
    process.off('exit', onExit)
  }

  if (exitSignal) process.kill(process.pid, exitSignal)
}

main().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : String(error)
  process.stderr.write(`Unable to run frontend skin command: ${message}\n`)
  process.exitCode = 1
})
