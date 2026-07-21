import { createHash } from 'node:crypto'
import {
  copyFile,
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rename,
  rm,
  stat,
  unlink,
  writeFile,
} from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import type { SkinBuildManifest } from '../src/skins/runtime/build-contracts'
import {
  getGeneratedRouteFile,
  normalizeSkinId,
  renderActiveBuildModule,
  renderActiveSkinModule,
  renderGeneratedRoute,
  validateSkinRoutes,
} from './skin-route-utils'

const defaultProjectRoot = new URL('../', import.meta.url)
let temporaryFileSequence = 0

type GenerateSkinOptions = {
  afterRouteBackup?: () => Promise<void>
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

async function loadBuildManifest(filePath: string): Promise<SkinBuildManifest> {
  const manifestContent = await readFile(filePath)
  const contentHash = createHash('sha256').update(manifestContent).digest('hex')
  const moduleUrl = pathToFileURL(filePath)
  moduleUrl.searchParams.set('version', contentHash)
  const loadedModule: unknown = await import(moduleUrl.href)

  if (
    typeof loadedModule !== 'object' ||
    loadedModule === null ||
    !('default' in loadedModule) ||
    typeof loadedModule.default !== 'object' ||
    loadedModule.default === null ||
    !('routes' in loadedModule.default) ||
    !Array.isArray(loadedModule.default.routes)
  ) {
    throw new Error(`Invalid skin build manifest: ${filePath}`)
  }

  return loadedModule.default as SkinBuildManifest
}

async function pathExists(filePath: string): Promise<boolean> {
  try {
    await lstat(filePath)
    return true
  } catch (error: unknown) {
    if (error instanceof Error && 'code' in error && error.code === 'ENOENT') {
      return false
    }
    throw error
  }
}

async function copyPreservedFilesAndReadOwnedRoutes(
  sourceDirectory: string,
  destinationDirectory: string,
  relativeDirectory = ''
): Promise<Map<string, string>> {
  const ownedRoutes = new Map<string, string>()
  if (!(await pathExists(sourceDirectory))) {
    return ownedRoutes
  }

  const sourceStat = await lstat(sourceDirectory)
  if (sourceStat.isSymbolicLink()) {
    throw new Error(`Symlink found under skin-generated routes: ${sourceDirectory}`)
  }
  if (!sourceStat.isDirectory()) {
    throw new Error(`Generated skin route root is not a directory: ${sourceDirectory}`)
  }

  const entries = await readdir(sourceDirectory)
  entries.sort((left, right) => left.localeCompare(right))

  for (const entry of entries) {
    const sourcePath = path.join(sourceDirectory, entry)
    const destinationPath = path.join(destinationDirectory, entry)
    const relativePath = path.join(relativeDirectory, entry)
    const entryStat = await lstat(sourcePath)

    if (entryStat.isSymbolicLink()) {
      throw new Error(`Symlink found under skin-generated routes: ${sourcePath}`)
    }
    if (entryStat.isDirectory()) {
      await mkdir(destinationPath, { recursive: true })
      const nestedRoutes = await copyPreservedFilesAndReadOwnedRoutes(
        sourcePath,
        destinationPath,
        relativePath
      )
      for (const [routePath, content] of nestedRoutes) {
        ownedRoutes.set(routePath, content)
      }
    } else if (entryStat.isFile() && entry.endsWith('.tsx')) {
      ownedRoutes.set(relativePath, await readFile(sourcePath, 'utf8'))
    } else if (entryStat.isFile()) {
      await copyFile(sourcePath, destinationPath)
    } else {
      throw new Error(`Unsupported entry under skin-generated routes: ${sourcePath}`)
    }
  }

  return ownedRoutes
}

function routeTreesMatch(
  liveRoutes: ReadonlyMap<string, string>,
  desiredRoutes: ReadonlyMap<string, string>
): boolean {
  if (liveRoutes.size !== desiredRoutes.size) {
    return false
  }

  for (const [routePath, content] of desiredRoutes) {
    if (liveRoutes.get(routePath) !== content) {
      return false
    }
  }

  return true
}

async function regenerateRouteProxies(
  projectRoot: string,
  routes: readonly ReturnType<typeof validateSkinRoutes>[number][],
  afterRouteBackup?: () => Promise<void>
): Promise<void> {
  const routesDirectory = path.join(projectRoot, 'src', 'routes')
  const generatedDirectory = path.join(routesDirectory, '(skin-generated)')
  await mkdir(routesDirectory, { recursive: true })
  const stagingDirectory = await mkdtemp(path.join(routesDirectory, '.skin-routes-stage-'))
  const backupDirectory = `${stagingDirectory}-backup`
  let backupCreated = false
  let stagedRoutesActivated = false

  try {
    const liveRoutes = await copyPreservedFilesAndReadOwnedRoutes(
      generatedDirectory,
      stagingDirectory
    )
    const desiredRoutes = new Map<string, string>()
    const stagedOutputs = routes.map((route) => {
      const relativePath = getGeneratedRouteFile(route.path).replace(
        /^\(skin-generated\)\//,
        ''
      )
      return {
        relativePath,
        filePath: path.join(stagingDirectory, relativePath),
        content: renderGeneratedRoute(route),
      }
    })
    for (const output of stagedOutputs) {
      desiredRoutes.set(output.relativePath, output.content)
      await mkdir(path.dirname(output.filePath), { recursive: true })
    }
    await writeChangedOutputs(stagedOutputs)

    if (routeTreesMatch(liveRoutes, desiredRoutes)) {
      return
    }

    if (await pathExists(generatedDirectory)) {
      await rename(generatedDirectory, backupDirectory)
      backupCreated = true
    }
    await afterRouteBackup?.()
    await rename(stagingDirectory, generatedDirectory)
    stagedRoutesActivated = true

    if (backupCreated) {
      await rm(backupDirectory, { recursive: true, force: true })
    }
  } catch (error: unknown) {
    if (stagedRoutesActivated) {
      await rm(generatedDirectory, { recursive: true, force: true })
    }
    if (backupCreated) {
      await rename(backupDirectory, generatedDirectory)
    }
    throw error
  } finally {
    await rm(stagingDirectory, { recursive: true, force: true })
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

  const buildManifest = await loadBuildManifest(buildManifestPath)
  const routes = validateSkinRoutes(buildManifest.routes)

  await regenerateRouteProxies(projectRoot, routes, options.afterRouteBackup)

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
