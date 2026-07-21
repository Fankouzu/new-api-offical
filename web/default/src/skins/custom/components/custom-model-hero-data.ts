export const MODEL_HERO_SLIDES = [
  {
    name: 'Kimi K3',
    image: 'https://image.lizh.ai/b3523655-97e9-4999-0df5-e30b8c7b7b00/large',
    description:
      'Built for long-context research, document understanding, and complex knowledge work.',
    tags: ['Long context', 'Deep research', 'Document intelligence'],
  },
  {
    name: 'GLM 5.2',
    image: 'https://image.lizh.ai/65d343eb-ada0-43fb-96dc-92ffce53f500/large',
    description:
      'Designed for complex reasoning, task planning, and coordinated agent workflows.',
    tags: ['Complex reasoning', 'AI agents', 'Task planning'],
  },
  {
    name: 'Qwen 3.8',
    image: 'https://image.lizh.ai/e2f24db5-f241-42b5-f797-32dea382af00/large',
    description:
      'A versatile model for multilingual, multimodal, and creative production scenarios.',
    tags: ['Multilingual', 'Multimodal', 'Creative work'],
  },
  {
    name: 'DeepSeek V4',
    image: 'https://image.lizh.ai/c8a31d22-d3c6-47da-4d2b-9c0a92c0d500/large',
    description:
      'Focused on coding, engineering reasoning, and demanding technical problem solving.',
    tags: ['Code generation', 'Engineering', 'Problem solving'],
  },
  {
    name: 'Mimo 2.5',
    image: 'https://image.lizh.ai/dadc22b8-d439-4415-1f55-de2da8648f00/large',
    description:
      'An efficient model for everyday productivity, practical automation, and fast execution.',
    tags: ['Efficient execution', 'Productivity', 'Automation'],
  },
] as const

export function getNextModelSlideIndex(current: number, total: number) {
  return (current + 1) % total
}
