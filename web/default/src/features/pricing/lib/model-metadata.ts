/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { Modality, ModelCapability, PricingModel } from '../types'
import { hashStringToSeed, seededRandom } from './seed'

// ----------------------------------------------------------------------------
// Model metadata inference
// ----------------------------------------------------------------------------
//
// Spec metadata (context_length, max_output_tokens, knowledge_cutoff,
// release_date) is now sourced from the models.dev catalog on the backend and
// returned on `model.*` (see model/model_catalog.go). When the backend has no
// entry for a model, those fields are left empty (no client-side mock).
//
// Modalities still fall back to a lightweight heuristic derived from the model's
// real endpoint / ratio configuration when the backend does not provide them.
// Capabilities are never guessed: only catalog-reported values are displayed.

const TEXT_INPUT_ENDPOINTS = new Set([
  'openai',
  'openai-response',
  'anthropic',
  'gemini',
  'embeddings',
  'jina-rerank',
])

const IMAGE_OUTPUT_ENDPOINTS = new Set(['image-generation'])
const VIDEO_OUTPUT_ENDPOINTS = new Set(['openai-video'])
const EMBEDDING_ENDPOINTS = new Set(['embeddings', 'jina-rerank'])

const TAG_TO_MODALITY: Record<string, Modality> = {
  text: 'text',
  image: 'image',
  audio: 'audio',
  video: 'video',
  file: 'file',
  document: 'file',
  pdf: 'file',
}

function parseModelTags(tagsString?: string): string[] {
  if (!tagsString) return []
  return tagsString
    .split(/[,;|\s]+/)
    .map((t) => t.trim().toLowerCase())
    .filter(Boolean)
}

function inferInputModalities(
  model: PricingModel,
  tags: string[],
  endpoints: string[]
): Modality[] {
  const set = new Set<Modality>()

  if (
    endpoints.length === 0 ||
    endpoints.some((e) => TEXT_INPUT_ENDPOINTS.has(e))
  ) {
    set.add('text')
  }

  if (model.image_ratio != null) {
    set.add('image')
  }
  if (model.audio_ratio != null) {
    set.add('audio')
  }

  for (const tag of tags) {
    const m = TAG_TO_MODALITY[tag]
    if (m) set.add(m)
  }

  if (set.size === 0) set.add('text')
  return ordered(set)
}

function inferOutputModalities(
  model: PricingModel,
  endpoints: string[]
): Modality[] {
  const set = new Set<Modality>()

  if (endpoints.some((e) => IMAGE_OUTPUT_ENDPOINTS.has(e))) set.add('image')
  if (endpoints.some((e) => VIDEO_OUTPUT_ENDPOINTS.has(e))) set.add('video')
  if (endpoints.some((e) => EMBEDDING_ENDPOINTS.has(e))) set.add('text')

  if (model.audio_completion_ratio != null) {
    set.add('audio')
  }

  if (set.size === 0) set.add('text')
  return ordered(set)
}

function ordered(modalities: Set<Modality>): Modality[] {
  const order: Modality[] = ['text', 'image', 'audio', 'video', 'file']
  return order.filter((m) => modalities.has(m))
}

export type ModelMetadata = {
  context_length: number
  max_output_tokens: number
  knowledge_cutoff: string
  release_date: string
  parameter_count: string
  input_modalities: Modality[]
  output_modalities: Modality[]
  capabilities: ModelCapability[]
}

/**
 * Build model metadata for display. Spec fields (context_length,
 * max_output_tokens, knowledge_cutoff, release_date, parameter_count) come
 * straight from the backend models.dev catalog (`model.*`) and are left empty
 * when the backend has no entry. Modalities prefer the backend and fall back to
 * real configuration; capabilities remain empty when the catalog reports none.
 */
export function inferModelMetadata(model: PricingModel): ModelMetadata {
  const tags = parseModelTags(model.tags)
  const endpoints = model.supported_endpoint_types || []

  const inputs =
    model.input_modalities ?? inferInputModalities(model, tags, endpoints)
  const outputs =
    model.output_modalities ?? inferOutputModalities(model, endpoints)
  const capabilities = model.capabilities ?? []

  return {
    context_length: model.context_length ?? 0,
    max_output_tokens: model.max_output_tokens ?? 0,
    knowledge_cutoff: model.knowledge_cutoff ?? '',
    release_date: model.release_date ?? '',
    parameter_count: model.parameter_count ?? '',
    input_modalities: inputs,
    output_modalities: outputs,
    capabilities,
  }
}

const TOKEN_FORMAT = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 1,
})

/** Format a token count compactly: 128_000 → "128K", 1_000_000 → "1M". */
export function formatTokenCount(tokens: number): string {
  if (!Number.isFinite(tokens) || tokens <= 0) return '—'
  if (tokens >= 1_000_000) {
    const value = tokens / 1_000_000
    return `${TOKEN_FORMAT.format(value)}M`
  }
  if (tokens >= 1_000) {
    const value = tokens / 1_000
    return `${TOKEN_FORMAT.format(value)}K`
  }
  return TOKEN_FORMAT.format(tokens)
}

/** Format a YYYY-MM (or YYYY-MM-DD) date as `Mon YYYY` for display. */
export function formatYearMonth(value: string): string {
  if (!value) return '—'
  const [yearStr, monthStr] = value.split('-')
  const year = Number(yearStr)
  const month = Number(monthStr)
  if (!Number.isFinite(year) || !Number.isFinite(month)) return value
  const date = new Date(Date.UTC(year, month - 1, 1))
  return date.toLocaleString(undefined, { year: 'numeric', month: 'short' })
}

// ---------------------------------------------------------------------------
// Provider / vendor / tokenizer / license inference
// ---------------------------------------------------------------------------
//
// These helpers derive vendor-style metadata from the model name. They are
// purely heuristic and serve only the API-info display until the backend
// returns explicit fields.

export type ModelVendor =
  | 'openai'
  | 'anthropic'
  | 'google'
  | 'meta'
  | 'mistral'
  | 'qwen'
  | 'deepseek'
  | 'xai'
  | 'cohere'
  | 'baidu'
  | 'zhipu'
  | 'moonshot'
  | 'minimax'
  | 'tencent'
  | 'bytedance'
  | 'midjourney'
  | 'stability'
  | 'unknown'

export type ApiInfo = {
  vendor: ModelVendor
  vendor_label: string
  tokenizer: string
  tokenizer_note?: string
  license: string
  license_kind: 'proprietary' | 'open' | 'open-weight' | 'unknown'
  data_retention_days: number
  training_opt_out: boolean
  homepage?: string
}

const VENDOR_LABELS: Record<ModelVendor, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  meta: 'Meta',
  mistral: 'Mistral AI',
  qwen: 'Alibaba (Qwen)',
  deepseek: 'DeepSeek',
  xai: 'xAI',
  cohere: 'Cohere',
  baidu: 'Baidu',
  zhipu: 'Zhipu AI',
  moonshot: 'Moonshot AI',
  minimax: 'MiniMax',
  tencent: 'Tencent',
  bytedance: 'ByteDance',
  midjourney: 'Midjourney',
  stability: 'Stability AI',
  unknown: 'Unknown',
}

function detectVendor(name: string): ModelVendor {
  const n = name.toLowerCase()
  if (/^gpt|^o[1-4]|davinci|babbage|whisper|tts|dall.?e|sora|^omni/.test(n))
    return 'openai'
  if (/claude/.test(n)) return 'anthropic'
  if (/gemini|gemma|imagen|veo|palm/.test(n)) return 'google'
  if (/llama|^codellama/.test(n)) return 'meta'
  if (/mistral|mixtral|codestral|magistral|pixtral/.test(n)) return 'mistral'
  if (/qwen|qwq|qvq/.test(n)) return 'qwen'
  if (/deepseek/.test(n)) return 'deepseek'
  if (/grok/.test(n)) return 'xai'
  if (/command|cohere|aya/.test(n)) return 'cohere'
  if (/ernie|wenxin/.test(n)) return 'baidu'
  if (/glm|chatglm|cogview|cogvideo/.test(n)) return 'zhipu'
  if (/kimi|moonshot/.test(n)) return 'moonshot'
  if (/abab|minimax|hailuo/.test(n)) return 'minimax'
  if (/hunyuan/.test(n)) return 'tencent'
  if (/doubao|seed|jimeng/.test(n)) return 'bytedance'
  if (/midjourney|niji/.test(n)) return 'midjourney'
  if (/^sd-|stable[-_]?diffusion|sdxl/.test(n)) return 'stability'
  return 'unknown'
}

const TOKENIZER_BY_VENDOR: Partial<Record<ModelVendor, string>> = {
  openai: 'o200k_base',
  anthropic: 'Anthropic Claude tokenizer',
  google: 'SentencePiece (Gemini)',
  meta: 'Llama 3 tokenizer',
  mistral: 'Mistral tokenizer (BPE)',
  qwen: 'Qwen tokenizer (tiktoken-compat)',
  deepseek: 'DeepSeek tokenizer (BPE)',
  xai: 'Grok tokenizer (BPE)',
  cohere: 'Cohere tokenizer',
  baidu: 'Ernie tokenizer',
  zhipu: 'GLM tokenizer',
  moonshot: 'Kimi tokenizer',
  minimax: 'ABAB tokenizer',
  tencent: 'Hunyuan tokenizer',
  bytedance: 'Doubao tokenizer',
}

function inferTokenizer(
  model: PricingModel,
  vendor: ModelVendor
): {
  tokenizer: string
  note?: string
} {
  const name = model.model_name.toLowerCase()
  if (vendor === 'openai') {
    if (/gpt-3|davinci|babbage|whisper|tts/.test(name)) {
      return { tokenizer: 'cl100k_base', note: 'Older GPT-3.5 family' }
    }
    return { tokenizer: 'o200k_base' }
  }
  return { tokenizer: TOKENIZER_BY_VENDOR[vendor] ?? 'BPE (vendor-specific)' }
}

const LICENSE_BY_VENDOR: Record<
  ModelVendor,
  { license: string; kind: ApiInfo['license_kind'] }
> = {
  openai: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  anthropic: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  google: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  meta: { license: 'Llama Community License', kind: 'open-weight' },
  mistral: { license: 'Apache 2.0 / Commercial', kind: 'open-weight' },
  qwen: { license: 'Tongyi Qianwen License', kind: 'open-weight' },
  deepseek: { license: 'DeepSeek License', kind: 'open-weight' },
  xai: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  cohere: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  baidu: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  zhipu: { license: 'GLM-4 License', kind: 'open-weight' },
  moonshot: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  minimax: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  tencent: { license: 'Hunyuan License', kind: 'open-weight' },
  bytedance: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  midjourney: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  stability: { license: 'Stability AI Community License', kind: 'open-weight' },
  unknown: { license: 'Provider-specific', kind: 'unknown' },
}

const HOMEPAGE_BY_VENDOR: Partial<Record<ModelVendor, string>> = {
  openai: 'https://platform.openai.com/docs/models',
  anthropic: 'https://docs.anthropic.com/claude/docs/models-overview',
  google: 'https://ai.google.dev/models',
  meta: 'https://llama.meta.com/',
  mistral: 'https://docs.mistral.ai/getting-started/models/',
  qwen: 'https://qwenlm.github.io/',
  deepseek: 'https://api-docs.deepseek.com/',
  xai: 'https://x.ai/api',
  cohere: 'https://docs.cohere.com/docs/models',
  baidu: 'https://cloud.baidu.com/product/wenxinworkshop',
  zhipu: 'https://open.bigmodel.cn/dev/api',
  moonshot: 'https://platform.moonshot.cn/docs',
  minimax: 'https://platform.minimaxi.com/document/notice',
  tencent: 'https://cloud.tencent.com/document/product/1729',
  bytedance: 'https://www.volcengine.com/docs/82379',
  midjourney: 'https://www.midjourney.com/',
  stability: 'https://platform.stability.ai/',
}

/**
 * Build vendor / tokenizer / license / privacy metadata for the model.
 * Returns deterministic values keyed off the model name so each render is
 * stable.
 */
export function inferApiInfo(model: PricingModel): ApiInfo {
  const vendor = detectVendor(model.model_name || '')
  const tk = inferTokenizer(model, vendor)
  const license = LICENSE_BY_VENDOR[vendor]
  const rand = seededRandom(hashStringToSeed(`${model.model_name}:api`))
  const retention = vendor === 'openai' ? 30 : Math.round(rand() * 90)
  return {
    vendor,
    vendor_label: VENDOR_LABELS[vendor],
    tokenizer: tk.tokenizer,
    tokenizer_note: tk.note,
    license: license.license,
    license_kind: license.kind,
    data_retention_days: retention,
    training_opt_out: true,
    homepage: HOMEPAGE_BY_VENDOR[vendor],
  }
}
