import React from 'react';

const priceClarifier =
  'Actual prices depend on account groups and settlement configuration.';

const topics = {
  '/use-cases/openai-compatible-api': {
    h1: 'OpenAI-Compatible API for Multiple AI Models',
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
  '/compare/ai-api-pricing': {
    h1: 'AI API Pricing Comparison',
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
  '/providers/gemini-api': providerTopic('Gemini'),
  '/providers/deepseek-api': providerTopic('DeepSeek'),
  '/providers/qwen-api': providerTopic('Qwen'),
  '/guides/openai-sdk-compatible': {
    h1: 'OpenAI SDK Compatible AI Model Access',
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
};

function providerTopic(provider) {
  return {
    h1: `${provider} API Pricing and Access`,
    intro: `Explore ${provider} API model options, approximate prices, and OpenAI-compatible access paths through Lizh AI.`,
    sections: [
      {
        title: `${provider} model marketplace`,
        body: `Use Lizh AI to compare available ${provider} models, prices, and model IDs from one AI model marketplace.`,
      },
      {
        title: 'API access',
        body: `${provider} model access can be evaluated alongside GPT, DeepSeek, Qwen, and other mainstream model families.`,
      },
    ],
    links: [
      {
        label: `${provider} model prices`,
        href: `/pricing?search=${provider.toLowerCase()}`,
      },
      { label: 'AI API pricing comparison', href: '/compare/ai-api-pricing' },
    ],
    faqs: [
      {
        question: `What is ${provider} API access in Lizh AI?`,
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
    ],
  };
}

const PublicTopic = ({ path }) => {
  const topic = topics[path];
  if (!topic) return null;

  return (
    <main className='mt-[72px] px-4 pb-16'>
      <article className='mx-auto max-w-5xl'>
        <header className='max-w-3xl space-y-4'>
          <p className='text-sm font-semibold uppercase tracking-wide text-semi-color-text-2'>
            Lizh AI
          </p>
          <h1 className='text-3xl font-bold leading-tight text-semi-color-text-0 md:text-5xl'>
            {topic.h1}
          </h1>
          <p className='text-base leading-7 text-semi-color-text-1'>
            {topic.intro}
          </p>
        </header>

        <div className='mt-10 grid gap-8 lg:grid-cols-[minmax(0,1fr)_280px]'>
          <div className='space-y-8'>
            {topic.sections.map((section) => (
              <section key={section.title} className='space-y-3'>
                <h2 className='text-2xl font-semibold text-semi-color-text-0'>
                  {section.title}
                </h2>
                <p className='leading-7 text-semi-color-text-1'>
                  {section.body}
                </p>
              </section>
            ))}

            <section className='space-y-4'>
              <h2 className='text-2xl font-semibold text-semi-color-text-0'>
                FAQ
              </h2>
              <div className='divide-y divide-semi-color-border rounded-lg border border-semi-color-border'>
                {topic.faqs.map((faq) => (
                  <div key={faq.question} className='space-y-2 p-5'>
                    <h3 className='font-medium text-semi-color-text-0'>
                      {faq.question}
                    </h3>
                    <p className='text-sm leading-6 text-semi-color-text-1'>
                      {faq.answer}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          </div>

          <aside className='space-y-3 lg:sticky lg:top-20 lg:self-start'>
            <h2 className='text-sm font-semibold uppercase tracking-wide text-semi-color-text-0'>
              Continue exploring
            </h2>
            <nav className='flex flex-col gap-2'>
              {topic.links.map((link) => (
                <a
                  key={link.href}
                  className='rounded border border-semi-color-border px-4 py-2 text-sm font-medium text-semi-color-primary hover:bg-semi-color-fill-0'
                  href={link.href}
                >
                  {link.label}
                </a>
              ))}
            </nav>
          </aside>
        </div>
      </article>
    </main>
  );
};

export default PublicTopic;
