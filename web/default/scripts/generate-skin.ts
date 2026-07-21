import { access, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  normalizeSkinId,
  renderActiveBuildModule,
  renderActiveSkinModule,
} from './skin-route-utils'

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

async function assertFileExists(filePath: string, skinId: string): Promise<void> {
  try {
    await access(filePath)
  } catch {
    throw new Error(
      `Skin "${skinId}" is missing required file: ${path.relative(projectRoot, filePath)}`
    )
  }
}

async function writeIfChanged(filePath: string, content: string): Promise<void> {
  let currentContent: string | undefined
  try {
    currentContent = await readFile(filePath, 'utf8')
  } catch {
    currentContent = undefined
  }

  if (currentContent !== content) {
    await writeFile(filePath, content, 'utf8')
  }
}

export async function generateSkin(): Promise<void> {
  const skinId = normalizeSkinId(process.env.APP_SKIN)
  const selectedSkinDirectory = path.join(projectRoot, 'src', 'skins', skinId)
  const runtimeManifestPath = path.join(selectedSkinDirectory, 'manifest.ts')
  const buildManifestPath = path.join(selectedSkinDirectory, 'build-manifest.ts')

  await Promise.all([
    assertFileExists(runtimeManifestPath, skinId),
    assertFileExists(buildManifestPath, skinId),
  ])

  const runtimeDirectory = path.join(projectRoot, 'src', 'skins', 'runtime')
  await Promise.all([
    writeIfChanged(
      path.join(runtimeDirectory, 'active-skin.gen.ts'),
      renderActiveSkinModule(skinId)
    ),
    writeIfChanged(
      path.join(runtimeDirectory, 'active-skin-build.gen.ts'),
      renderActiveBuildModule(skinId)
    ),
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
