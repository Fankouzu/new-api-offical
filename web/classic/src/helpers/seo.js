const DEFAULT_META = {
  title: 'Lizh AI | GPT、Gemini、DeepSeek、Qwen 多模型 API 聚合平台',
  description:
    'Lizh AI 提供 OpenAI 兼容的大模型 API 聚合服务，支持 GPT、Gemini、DeepSeek、Qwen、豆包、GLM、MiniMax、Kimi 等模型，统一计费、统一接口、快速接入。',
  robots: 'index,follow',
};

export function syncRouteSEO(pathname) {
  const meta = getRouteMeta(pathname);
  document.title = meta.title;
  upsertMeta('name', 'title', meta.title);
  upsertMeta('name', 'description', meta.description);
  upsertMeta('name', 'robots', meta.robots);
  upsertCanonical(`${window.location.origin}${pathname}`);
}

function getRouteMeta(pathname) {
  const path = pathname.replace(/\/+$/, '') || '/';
  if (path === '/') return DEFAULT_META;
  if (path === '/pricing') {
    return {
      title:
        'AI 大模型 API 价格广场 | GPT、Gemini、DeepSeek、Qwen、豆包模型价格 - Lizh AI',
      description:
        '查看 Lizh AI 在售大模型 API 价格，覆盖 GPT、Gemini、DeepSeek、Qwen、GLM、豆包、MiniMax、Kimi 等 50+ 模型，支持文本、图像、工具调用和结构化输出。',
      robots: 'index,follow',
    };
  }
  if (path.startsWith('/pricing/')) {
    const modelId = safeDecodeURIComponent(path.slice('/pricing/'.length));
    return {
      title: `${formatModelName(modelId)} API 价格 | Lizh AI`,
      description: `查看 ${formatModelName(modelId)} API 在 Lizh AI 的模型价格、能力和 OpenAI 兼容接入信息。`,
      robots: 'index,follow',
    };
  }
  if (path === '/about') {
    return {
      title: '关于 Lizh AI | 多模型 API 聚合与 OpenAI 兼容网关',
      description:
        '了解 Lizh AI 的多模型 API 聚合服务、OpenAI 兼容接口、统一计费能力和面向开发者的模型接入体验。',
      robots: 'index,follow',
    };
  }
  if (path === '/privacy-policy') {
    return {
      title: '隐私政策 | Lizh AI',
      description:
        '查看 Lizh AI 隐私政策，了解账号、API 调用、计费与服务数据的处理方式。',
      robots: 'index,follow',
    };
  }
  if (path === '/user-agreement') {
    return {
      title: '用户协议 | Lizh AI',
      description:
        '查看 Lizh AI 用户协议，了解 API 聚合服务使用、账号、计费与合规要求。',
      robots: 'index,follow',
    };
  }
  return {
    title: utilityTitle(path),
    description: '该页面用于账号、控制台或系统流程，不建议作为搜索结果收录。',
    robots: 'noindex,nofollow',
  };
}

function utilityTitle(path) {
  if (path === '/login') return '登录 | Lizh AI';
  if (path === '/register') return '注册 | Lizh AI';
  if (path.includes('reset')) return '找回密码 | Lizh AI';
  if (path.startsWith('/oauth')) return '授权登录 | Lizh AI';
  if (path.startsWith('/console')) return '控制台 | Lizh AI';
  if (path === '/setup') return '系统初始化 | Lizh AI';
  if (path === '/forbidden') return '无权访问 | Lizh AI';
  return '页面未找到 | Lizh AI';
}

function formatModelName(modelId) {
  return modelId
    .split(/[-_/\s]+/)
    .filter(Boolean)
    .map((part) => {
      const lower = part.toLowerCase();
      if (
        ['gpt', 'glm', 'api', 'json', 'vl', 'tts', 'ocr', 'ai'].includes(lower)
      ) {
        return part.toUpperCase();
      }
      if (lower === 'qwen') return 'Qwen';
      if (lower === 'deepseek') return 'DeepSeek';
      if (lower === 'gemini') return 'Gemini';
      if (lower === 'doubao') return 'Doubao';
      if (lower === 'minimax') return 'MiniMax';
      if (lower === 'kimi') return 'Kimi';
      return part.charAt(0).toUpperCase() + part.slice(1);
    })
    .join(' ');
}

function safeDecodeURIComponent(value) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function upsertMeta(attrName, attrValue, content) {
  const selector = `meta[${attrName}="${attrValue}"]`;
  let element = document.head.querySelector(selector);
  if (!element) {
    element = document.createElement('meta');
    element.setAttribute(attrName, attrValue);
    document.head.appendChild(element);
  }
  element.content = content;
}

function upsertCanonical(href) {
  let element = document.head.querySelector('link[rel="canonical"]');
  if (!element) {
    element = document.createElement('link');
    element.rel = 'canonical';
    document.head.appendChild(element);
  }
  element.href = href;
}
