export type PublicTopic = {
  path: string
  h1: string
  title: string
  description: string
  intro: string
  sections: Array<{
    title: string
    body: string
  }>
  links: Array<{
    label: string
    href: string
  }>
  faqs: Array<{
    question: string
    answer: string
  }>
}

const priceClarifier =
  'Actual prices depend on account groups and settlement configuration.'

export const publicTopics: PublicTopic[] = [
  {
    path: '/use-cases/openai-compatible-api',
    h1: 'OpenAI-Compatible API for Multiple AI Models',
    title: 'OpenAI-Compatible API for GPT, Gemini, DeepSeek and Qwen | Lizh AI',
    description:
      'Use Lizh AI as an OpenAI-compatible API gateway for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream AI models.',
    intro:
      'Lizh AI provides an OpenAI-compatible gateway for developers who want to test and route multiple mainstream AI models without rebuilding every client integration.',
    sections: [
      {
        title: 'Why developers use Lizh AI',
        body: 'A single API marketplace makes it easier to compare GPT, Gemini, DeepSeek, Qwen, and other model families while keeping integration work predictable.',
      },
      {
        title: 'Unified model switching',
        body: 'Use model IDs from the pricing marketplace to switch workloads between providers and price tiers.',
      },
    ],
    links: [
      { label: 'Model pricing marketplace', href: '/pricing' },
      {
        label: 'OpenAI SDK compatible guide',
        href: '/guides/openai-sdk-compatible',
      },
    ],
    faqs: [
      {
        question: 'What is OpenAI-compatible API access in Lizh AI?',
        answer:
          'It is a gateway pattern for using supported AI models through familiar OpenAI-compatible client configuration.',
      },
      {
        question: 'Where can I compare model prices?',
        answer:
          'Use the pricing marketplace to compare all available model prices and detail pages.',
      },
      {
        question: 'Are prices final billing guarantees?',
        answer: priceClarifier,
      },
    ],
  },
  {
    path: '/compare/ai-api-pricing',
    h1: 'AI API Pricing Comparison',
    title: 'AI API Pricing Comparison for Mainstream Models | Lizh AI',
    description:
      'Compare approximate AI API prices across GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other models available in Lizh AI.',
    intro:
      'Compare AI API prices across mainstream models and understand input, output, multimodal, and request-based billing patterns.',
    sections: [
      {
        title: 'Compare by workload',
        body: 'Different workloads care about different prices: chat prompts, generated tokens, image generation, and structured tool workflows can each have different cost drivers.',
      },
      {
        title: 'Use price plus capability',
        body: 'The cheapest model is not always the best fit. Compare price together with model family, capability, and SDK compatibility.',
      },
    ],
    links: [
      { label: 'All AI model prices', href: '/pricing' },
      {
        label: 'OpenAI-compatible API',
        href: '/use-cases/openai-compatible-api',
      },
    ],
    faqs: [
      {
        question: 'How should I compare AI API prices?',
        answer:
          'Compare input price, output price, workload type, model capability, and account settlement rules together.',
      },
      {
        question: 'Are prices final billing guarantees?',
        answer: priceClarifier,
      },
    ],
  },
  {
    path: '/providers/gemini-api',
    h1: 'Gemini API Pricing and Access',
    title: 'Gemini API Pricing and Model Access | Lizh AI',
    description:
      'Explore Gemini API model options, approximate prices, model IDs, and OpenAI-compatible access paths through Lizh AI.',
    intro:
      'Explore Gemini API model options, approximate prices, and OpenAI-compatible access paths through Lizh AI.',
    sections: providerSections('Gemini'),
    links: [
      { label: 'Gemini model prices', href: '/pricing?search=gemini' },
      { label: 'AI API pricing comparison', href: '/compare/ai-api-pricing' },
    ],
    faqs: providerFAQ('Gemini API'),
  },
  {
    path: '/providers/deepseek-api',
    h1: 'DeepSeek API Pricing and Access',
    title: 'DeepSeek API Pricing and Model Access | Lizh AI',
    description:
      'Explore DeepSeek API options for reasoning, coding, and cost-conscious AI workloads with approximate prices and model IDs in Lizh AI.',
    intro:
      'Explore DeepSeek API options for reasoning, coding, and cost-conscious AI workloads through Lizh AI.',
    sections: providerSections('DeepSeek'),
    links: [
      { label: 'DeepSeek model prices', href: '/pricing?search=deepseek' },
      { label: 'AI API pricing comparison', href: '/compare/ai-api-pricing' },
    ],
    faqs: providerFAQ('DeepSeek API'),
  },
  {
    path: '/providers/qwen-api',
    h1: 'Qwen API Pricing and Access',
    title: 'Qwen API Pricing and Model Access | Lizh AI',
    description:
      'Explore Qwen API pricing, model IDs, multilingual and coding use cases, and OpenAI-compatible access through Lizh AI.',
    intro:
      'Explore Qwen API pricing, model IDs, and integration options in the Lizh AI model marketplace.',
    sections: providerSections('Qwen'),
    links: [
      { label: 'Qwen model prices', href: '/pricing?search=qwen' },
      { label: 'AI API pricing comparison', href: '/compare/ai-api-pricing' },
    ],
    faqs: providerFAQ('Qwen API'),
  },
  {
    path: '/guides/openai-sdk-compatible',
    h1: 'OpenAI SDK Compatible AI Model Access',
    title: 'OpenAI SDK Compatible API Guide for Multiple AI Models | Lizh AI',
    description:
      'Learn how to use OpenAI-compatible client patterns, API base URLs, API keys, and model IDs to access multiple AI model families from Lizh AI.',
    intro:
      'Use OpenAI-compatible client patterns to access multiple AI model families from Lizh AI with minimal integration changes.',
    sections: [
      {
        title: 'Integration approach',
        body: 'Configure your API base URL, API key, and model ID, then call supported chat or response endpoints according to the model capability.',
      },
      {
        title: 'Model selection',
        body: 'Start from the pricing marketplace, pick a model ID, and verify whether the model supports your text, image, or structured-output workflow.',
      },
    ],
    links: [
      {
        label: 'OpenAI-compatible API overview',
        href: '/use-cases/openai-compatible-api',
      },
      { label: 'Model pricing marketplace', href: '/pricing' },
    ],
    faqs: [
      {
        question: 'Can I use OpenAI SDK-style clients with Lizh AI?',
        answer:
          'Supported models can be accessed with OpenAI-compatible configuration patterns where the endpoint and model capability match the workload.',
      },
      {
        question: 'Where do I find model IDs?',
        answer:
          'Start from the pricing marketplace and open a model detail page for the current model ID.',
      },
    ],
  },
]

export function getPublicTopic(path: string) {
  const normalized = path.replace(/\/+$/, '') || '/'
  return publicTopics.find((topic) => topic.path === normalized)
}

function providerSections(provider: string): PublicTopic['sections'] {
  return [
    {
      title: `${provider} model marketplace`,
      body: `Use Lizh AI to compare available ${provider} models, prices, and model IDs from one AI model marketplace.`,
    },
    {
      title: 'API access',
      body: `${provider} model access can be evaluated alongside GPT, DeepSeek, Qwen, and other mainstream model families.`,
    },
  ]
}

function providerFAQ(provider: string): PublicTopic['faqs'] {
  return [
    {
      question: `What is ${provider} access in Lizh AI?`,
      answer:
        'It is a public provider page that explains model access, pricing, and integration paths in the Lizh AI marketplace.',
    },
    {
      question: 'Where can I compare prices?',
      answer:
        'Use the pricing marketplace to compare all available model prices and detail pages.',
    },
    {
      question: 'Are prices final billing guarantees?',
      answer: priceClarifier,
    },
  ]
}
