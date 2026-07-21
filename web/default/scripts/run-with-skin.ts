import { spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { generateSkin } from './generate-skin'
import { normalizeSkinId } from './skin-route-utils'
import {
  acquireSkinWorkspaceLease,
  installSkinLeaseProcessCleanup,
} from './skin-workspace-lock'

const projectRoot = fileURLToPath(new URL('../', import.meta.url))

async function runTypecheck(): Promise<number> {
  const tscPath = path.join(projectRoot, 'node_modules', '.bin', 'tsc')
  const child = spawn(tscPath, ['-b'], {
    cwd: projectRoot,
    stdio: 'inherit',
  })

  return new Promise((resolve, reject) => {
    child.once('error', reject)
    child.once('exit', (code, signal) => {
      if (signal) {
        process.kill(process.pid, signal)
        return
      }
      resolve(code ?? 1)
    })
  })
}

async function main(): Promise<void> {
  const mode = process.argv[2]
  if (mode !== 'generate' && mode !== 'typecheck') {
    throw new Error(`Unknown skin command mode: ${mode ?? '(missing)'}`)
  }

  const skinId = normalizeSkinId(process.env.APP_SKIN)
  const lease = await acquireSkinWorkspaceLease({ projectRoot, skinId })
  installSkinLeaseProcessCleanup(lease)

  try {
    await generateSkin({ skinId })
    if (mode === 'typecheck') {
      process.exitCode = await runTypecheck()
    }
  } finally {
    await lease.release()
  }
}

main().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : String(error)
  process.stderr.write(`Unable to run frontend skin command: ${message}\n`)
  process.exitCode = 1
})
