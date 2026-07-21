import { randomUUID } from 'node:crypto'
import {
  mkdir,
  readFile,
  realpath,
  rm,
  writeFile,
} from 'node:fs/promises'
import {
  readFileSync,
  rmSync,
} from 'node:fs'
import path from 'node:path'

type LockMetadata = {
  pid: number
  skinId: string
  startedAt: string
  token: string
}

type AcquireOptions = {
  projectRoot: string
  skinId: string
}

export type SkinWorkspaceLease = {
  release: () => Promise<void>
  releaseSync: () => void
  skinId: string
  token: string
}

type ActiveLease = {
  metadata: LockMetadata
  references: number
}

const activeLeases = new Map<string, ActiveLease>()
const cleanupLeases = new Set<SkinWorkspaceLease>()
let processCleanupInstalled = false

export function getSkinWorkspaceLockPaths(projectRoot: string) {
  const cacheDirectory = path.join(projectRoot, 'node_modules', '.cache')
  const lockDirectory = path.join(cacheDirectory, 'frontend-skin-workspace.lock')
  return {
    cacheDirectory,
    lockDirectory,
    metadataPath: path.join(lockDirectory, 'owner.json'),
    reclaimDirectory: `${lockDirectory}.reclaim`,
  }
}

function isProcessAlive(pid: number): boolean {
  try {
    process.kill(pid, 0)
    return true
  } catch (error: unknown) {
    return error instanceof Error && 'code' in error && error.code === 'EPERM'
  }
}

async function readMetadata(metadataPath: string): Promise<LockMetadata | undefined> {
  try {
    const parsed = JSON.parse(await readFile(metadataPath, 'utf8')) as Partial<LockMetadata>
    if (
      typeof parsed.pid === 'number' &&
      typeof parsed.skinId === 'string' &&
      typeof parsed.startedAt === 'string' &&
      typeof parsed.token === 'string'
    ) {
      return parsed as LockMetadata
    }
  } catch {
    return undefined
  }
  return undefined
}

function conflictMessage(owner: LockMetadata | undefined, requestedSkin: string): string {
  if (!owner) {
    return `Frontend skin workspace is locked by another process with unreadable metadata; requested skin "${requestedSkin}". Remove the lock only after confirming no frontend command is running.`
  }
  return `Frontend skin workspace is already owned by skin "${owner.skinId}" (PID ${owner.pid}); requested skin "${requestedSkin}". Stop the active frontend command and retry.`
}

function metadataMatchesSync(metadataPath: string, token: string): boolean {
  try {
    const metadata = JSON.parse(readFileSync(metadataPath, 'utf8')) as Partial<LockMetadata>
    return metadata.token === token && metadata.pid === process.pid
  } catch {
    return false
  }
}

function createLease(lockDirectory: string, metadataPath: string, active: ActiveLease): SkinWorkspaceLease {
  let released = false

  const decrement = (): boolean => {
    if (released) return false
    released = true
    active.references -= 1
    return active.references === 0
  }

  return {
    skinId: active.metadata.skinId,
    token: active.metadata.token,
    release: async () => {
      if (!decrement()) return
      activeLeases.delete(lockDirectory)
      if (metadataMatchesSync(metadataPath, active.metadata.token)) {
        await rm(lockDirectory, { recursive: true, force: true })
      }
    },
    releaseSync: () => {
      if (!decrement()) return
      activeLeases.delete(lockDirectory)
      if (metadataMatchesSync(metadataPath, active.metadata.token)) {
        rmSync(lockDirectory, { recursive: true, force: true })
      }
    },
  }
}

export async function assertSkinWorkspaceLeaseOwner(options: {
  ownerPid: number
  projectRoot: string
  skinId: string
  token: string
}): Promise<void> {
  const metadata = await readMetadata(
    getSkinWorkspaceLockPaths(options.projectRoot).metadataPath
  )
  if (
    !metadata ||
    metadata.pid !== options.ownerPid ||
    metadata.skinId !== options.skinId ||
    metadata.token !== options.token ||
    !isProcessAlive(metadata.pid)
  ) {
    throw new Error(
      `Frontend skin workspace lease delegation is invalid for skin "${options.skinId}"`
    )
  }
}

export async function assertSkinLeaseDelegationProject(options: {
  delegatedProjectRoot: string
  projectRoot: string
}): Promise<string> {
  const [canonicalProjectRoot, canonicalDelegatedRoot] = await Promise.all([
    realpath(options.projectRoot),
    realpath(options.delegatedProjectRoot),
  ])
  if (canonicalDelegatedRoot !== canonicalProjectRoot) {
    throw new Error(
      `Frontend skin workspace lease delegation project mismatch: expected ${canonicalProjectRoot}`
    )
  }
  return canonicalProjectRoot
}

export async function acquireSkinWorkspaceLease(
  options: AcquireOptions
): Promise<SkinWorkspaceLease> {
  const paths = getSkinWorkspaceLockPaths(options.projectRoot)
  const existing = activeLeases.get(paths.lockDirectory)
  if (existing) {
    if (existing.metadata.skinId !== options.skinId) {
      throw new Error(conflictMessage(existing.metadata, options.skinId))
    }
    existing.references += 1
    return createLease(paths.lockDirectory, paths.metadataPath, existing)
  }

  await mkdir(paths.cacheDirectory, { recursive: true })

  for (let attempt = 0; attempt < 2; attempt += 1) {
    try {
      await mkdir(paths.lockDirectory)
      const metadata: LockMetadata = {
        pid: process.pid,
        skinId: options.skinId,
        startedAt: new Date().toISOString(),
        token: randomUUID(),
      }
      try {
        await writeFile(paths.metadataPath, `${JSON.stringify(metadata)}\n`, 'utf8')
      } catch (error: unknown) {
        await rm(paths.lockDirectory, { recursive: true, force: true })
        throw error
      }
      const active = { metadata, references: 1 }
      activeLeases.set(paths.lockDirectory, active)
      return createLease(paths.lockDirectory, paths.metadataPath, active)
    } catch (error: unknown) {
      if (!(error instanceof Error && 'code' in error && error.code === 'EEXIST')) {
        throw error
      }
    }

    const owner = await readMetadata(paths.metadataPath)
    if (!owner || isProcessAlive(owner.pid)) {
      throw new Error(conflictMessage(owner, options.skinId))
    }
    if (attempt > 0) {
      throw new Error(conflictMessage(owner, options.skinId))
    }

    try {
      await mkdir(paths.reclaimDirectory)
    } catch (error: unknown) {
      if (error instanceof Error && 'code' in error && error.code === 'EEXIST') {
        throw new Error(
          `Frontend skin workspace stale lock is already being reclaimed; requested skin "${options.skinId}". Retry shortly.`,
          { cause: error }
        )
      }
      throw error
    }

    try {
      const currentOwner = await readMetadata(paths.metadataPath)
      if (!currentOwner || currentOwner.token !== owner.token || isProcessAlive(currentOwner.pid)) {
        throw new Error(conflictMessage(currentOwner, options.skinId))
      }
      await rm(paths.lockDirectory, { recursive: true, force: true })
    } finally {
      await rm(paths.reclaimDirectory, { recursive: true, force: true })
    }
  }

  throw new Error(`Unable to acquire frontend skin workspace for "${options.skinId}"`)
}

export function installSkinLeaseProcessCleanup(lease: SkinWorkspaceLease): void {
  cleanupLeases.add(lease)
  if (processCleanupInstalled) return
  processCleanupInstalled = true

  const signals: NodeJS.Signals[] = ['SIGINT', 'SIGTERM', 'SIGHUP']
  const releaseAll = () => {
    for (const activeLease of cleanupLeases) activeLease.releaseSync()
  }
  const onExit = () => releaseAll()
  const signalHandlers = new Map<NodeJS.Signals, () => void>()

  process.once('exit', onExit)
  for (const signal of signals) {
    const handler = () => {
      releaseAll()
      process.off('exit', onExit)
      for (const [registeredSignal, registeredHandler] of signalHandlers) {
        process.off(registeredSignal, registeredHandler)
      }
      process.kill(process.pid, signal)
    }
    signalHandlers.set(signal, handler)
    process.once(signal, handler)
  }
}
