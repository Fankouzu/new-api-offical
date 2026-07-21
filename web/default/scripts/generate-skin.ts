import { readFile, rename, stat, unlink, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  normalizeSkinId,
  renderActiveBuildModule,
  renderActiveSkinModule,
} from './skin-route-utils'

const defaultProjectRoot = new URL('../', import.meta.url)
let temporaryFileSequence = 0

type GenerateSkinOptions = {
  projectRoot?: URL
  skinId?: string
}

type GeneratedOutput = {
  content: string
  filePath: string
}

async function assertManifestFile(
  filePath: string,
  projectRoot: string,
  skinId: string
): Promise<void> {
  let fileStat: Awaited<ReturnType<typeof stat>>
  try {
    fileStat = await stat(filePath)
  } catch (error: unknown) {
    if (error instanceof Error && 'code' in error && error.code === 'ENOENT') {
      throw new Error(
        `Skin "${skinId}" is missing required file: ${path.relative(projectRoot, filePath)}`,
        { cause: error }
      )
    }

    const message = error instanceof Error ? error.message : String(error)
    throw new Error(
      `Unable to inspect required skin file ${path.relative(projectRoot, filePath)}: ${message}`,
      { cause: error }
    )
  }

  if (!fileStat.isFile()) {
    throw new Error(
      `Skin "${skinId}" required path is not a file: ${path.relative(projectRoot, filePath)}`
    )
  }
}

async function hasChanged(output: GeneratedOutput): Promise<boolean> {
  try {
    return (await readFile(output.filePath, 'utf8')) !== output.content
  } catch (error: unknown) {
    if (error instanceof Error && 'code' in error && error.code === 'ENOENT') {
      return true
    }

    const message = error instanceof Error ? error.message : String(error)
    throw new Error(`Unable to read generated skin file ${output.filePath}: ${message}`, {
      cause: error,
    })
  }
}

async function writeChangedOutputs(outputs: GeneratedOutput[]): Promise<void> {
  const changedOutputs: GeneratedOutput[] = []
  for (const output of outputs) {
    if (await hasChanged(output)) {
      changedOutputs.push(output)
    }
  }

  if (changedOutputs.length === 0) {
    return
  }

  const temporaryOutputs = changedOutputs.map((output) => ({
    ...output,
    temporaryPath: `${output.filePath}.${process.pid}.${Date.now()}.${temporaryFileSequence++}.tmp`,
  }))

  try {
    for (const output of temporaryOutputs) {
      await writeFile(output.temporaryPath, output.content, 'utf8')
    }
    for (const output of temporaryOutputs) {
      await rename(output.temporaryPath, output.filePath)
    }
  } finally {
    await Promise.all(
      temporaryOutputs.map((output) => unlink(output.temporaryPath).catch(() => undefined))
    )
  }
}

export async function generateSkin(options: GenerateSkinOptions = {}): Promise<void> {
  const projectRoot = fileURLToPath(options.projectRoot ?? defaultProjectRoot)
  const skinId = normalizeSkinId(options.skinId ?? process.env.APP_SKIN)
  const selectedSkinDirectory = path.join(projectRoot, 'src', 'skins', skinId)
  const runtimeManifestPath = path.join(selectedSkinDirectory, 'manifest.ts')
  const buildManifestPath = path.join(selectedSkinDirectory, 'build-manifest.ts')

  await assertManifestFile(runtimeManifestPath, projectRoot, skinId)
  await assertManifestFile(buildManifestPath, projectRoot, skinId)

  const runtimeDirectory = path.join(projectRoot, 'src', 'skins', 'runtime')
  await writeChangedOutputs([
    {
      filePath: path.join(runtimeDirectory, 'active-skin.gen.ts'),
      content: renderActiveSkinModule(skinId),
    },
    {
      filePath: path.join(runtimeDirectory, 'active-skin-build.gen.ts'),
      content: renderActiveBuildModule(skinId),
    },
  ])
}

const entryPath = process.argv[1] ? path.resolve(process.argv[1]) : undefined
if (entryPath === fileURLToPath(import.meta.url)) {
  generateSkin().catch((error: unknown) => {
    const message = error instanceof Error ? error.message : String(error)
    process.stderr.write(`Unable to generate active skin: ${message}\n`)
    process.exitCode = 1
  })
}
