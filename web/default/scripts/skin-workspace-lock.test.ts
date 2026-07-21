import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { describe, it } from 'node:test'
import {
  acquireSkinWorkspaceLease,
  assertSkinWorkspaceLeaseOwner,
  getSkinWorkspaceLockPaths,
} from './skin-workspace-lock'

async function createRoot(): Promise<string> {
  return mkdtemp(path.join(os.tmpdir(), 'skin-workspace-lock-'))
}

describe('skin workspace lease', { concurrency: false }, () => {
  it('acquires reentrantly for the same process and skin, then releases by reference count', async () => {
    const projectRoot = await createRoot()
    try {
      const first = await acquireSkinWorkspaceLease({ projectRoot, skinId: 'default' })
      const second = await acquireSkinWorkspaceLease({ projectRoot, skinId: 'default' })
      const { lockDirectory } = getSkinWorkspaceLockPaths(projectRoot)

      await second.release()
      assert.equal(JSON.parse(await readFile(path.join(lockDirectory, 'owner.json'), 'utf8')).skinId, 'default')

      await first.release()
      await assert.rejects(readFile(path.join(lockDirectory, 'owner.json'), 'utf8'))
      await first.release()
    } finally {
      await rm(projectRoot, { recursive: true, force: true })
    }
  })

  it('fails fast for a live owner, including the same requested skin', async () => {
    const projectRoot = await createRoot()
    const { lockDirectory, metadataPath } = getSkinWorkspaceLockPaths(projectRoot)
    try {
      await mkdir(lockDirectory, { recursive: true })
      await writeFile(
        metadataPath,
        JSON.stringify({ pid: process.pid, skinId: 'custom', startedAt: 'now', token: 'other' })
      )

      await assert.rejects(
        acquireSkinWorkspaceLease({ projectRoot, skinId: 'custom' }),
        /skin "custom".*PID.*requested skin "custom"/i
      )
      await assert.rejects(
        acquireSkinWorkspaceLease({ projectRoot, skinId: 'default' }),
        /skin "custom".*PID.*requested skin "default"/i
      )
    } finally {
      await rm(projectRoot, { recursive: true, force: true })
    }
  })

  it('reclaims one stale lock and records the new owner', async () => {
    const projectRoot = await createRoot()
    const { lockDirectory, metadataPath } = getSkinWorkspaceLockPaths(projectRoot)
    try {
      await mkdir(lockDirectory, { recursive: true })
      await writeFile(
        metadataPath,
        JSON.stringify({ pid: 2147483647, skinId: 'custom', startedAt: 'old', token: 'stale' })
      )

      const lease = await acquireSkinWorkspaceLease({ projectRoot, skinId: 'default' })
      const metadata = JSON.parse(await readFile(metadataPath, 'utf8')) as {
        pid: number
        skinId: string
      }
      assert.equal(metadata.pid, process.pid)
      assert.equal(metadata.skinId, 'default')
      await lease.release()
    } finally {
      await rm(projectRoot, { recursive: true, force: true })
    }
  })

  it('does not remove a lock whose ownership metadata no longer matches', async () => {
    const projectRoot = await createRoot()
    const { metadataPath } = getSkinWorkspaceLockPaths(projectRoot)
    try {
      const lease = await acquireSkinWorkspaceLease({ projectRoot, skinId: 'default' })
      const replacement = {
        pid: process.pid,
        skinId: 'custom',
        startedAt: 'replacement',
        token: 'replacement-token',
      }
      await writeFile(metadataPath, JSON.stringify(replacement))

      await lease.release()

      assert.deepEqual(JSON.parse(await readFile(metadataPath, 'utf8')), replacement)
    } finally {
      await rm(projectRoot, { recursive: true, force: true })
    }
  })

  it('delegates only with the live owner PID, token, and skin', async () => {
    const projectRoot = await createRoot()
    const lease = await acquireSkinWorkspaceLease({
      projectRoot,
      skinId: 'default',
    })
    try {
      await assertSkinWorkspaceLeaseOwner({
        ownerPid: process.pid,
        projectRoot,
        skinId: 'default',
        token: lease.token,
      })
      for (const override of [
        { token: 'forged' },
        { skinId: 'custom' },
        { ownerPid: 2147483647 },
      ]) {
        await assert.rejects(
          assertSkinWorkspaceLeaseOwner({
            ownerPid: process.pid,
            projectRoot,
            skinId: 'default',
            token: lease.token,
            ...override,
          }),
          /lease delegation is invalid/
        )
      }
    } finally {
      await lease.release()
      await rm(projectRoot, { recursive: true, force: true })
    }
  })
})
