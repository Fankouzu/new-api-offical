import React from 'react';

import Ai360Color from '@lobehub/icons/es/Ai360/components/Color';
import Ai360Mono from '@lobehub/icons/es/Ai360/components/Mono';
import AzureAIColor from '@lobehub/icons/es/AzureAI/components/Color';
import AzureAIMono from '@lobehub/icons/es/AzureAI/components/Mono';
import ClaudeColor from '@lobehub/icons/es/Claude/components/Color';
import ClaudeMono from '@lobehub/icons/es/Claude/components/Mono';
import CloudflareColor from '@lobehub/icons/es/Cloudflare/components/Color';
import CloudflareMono from '@lobehub/icons/es/Cloudflare/components/Mono';
import CohereColor from '@lobehub/icons/es/Cohere/components/Color';
import CohereMono from '@lobehub/icons/es/Cohere/components/Mono';
import CozeMono from '@lobehub/icons/es/Coze/components/Mono';
import DeepSeekColor from '@lobehub/icons/es/DeepSeek/components/Color';
import DeepSeekMono from '@lobehub/icons/es/DeepSeek/components/Mono';
import DifyColor from '@lobehub/icons/es/Dify/components/Color';
import DifyMono from '@lobehub/icons/es/Dify/components/Mono';
import DoubaoColor from '@lobehub/icons/es/Doubao/components/Color';
import DoubaoMono from '@lobehub/icons/es/Doubao/components/Mono';
import FalColor from '@lobehub/icons/es/Fal/components/Color';
import FalMono from '@lobehub/icons/es/Fal/components/Mono';
import FastGPTColor from '@lobehub/icons/es/FastGPT/components/Color';
import FastGPTMono from '@lobehub/icons/es/FastGPT/components/Mono';
import GeminiColor from '@lobehub/icons/es/Gemini/components/Color';
import GeminiMono from '@lobehub/icons/es/Gemini/components/Mono';
import GrokMono from '@lobehub/icons/es/Grok/components/Mono';
import HunyuanColor from '@lobehub/icons/es/Hunyuan/components/Color';
import HunyuanMono from '@lobehub/icons/es/Hunyuan/components/Mono';
import JimengColor from '@lobehub/icons/es/Jimeng/components/Color';
import JimengMono from '@lobehub/icons/es/Jimeng/components/Mono';
import JinaMono from '@lobehub/icons/es/Jina/components/Mono';
import KlingColor from '@lobehub/icons/es/Kling/components/Color';
import KlingMono from '@lobehub/icons/es/Kling/components/Mono';
import MidjourneyMono from '@lobehub/icons/es/Midjourney/components/Mono';
import MinimaxColor from '@lobehub/icons/es/Minimax/components/Color';
import MinimaxMono from '@lobehub/icons/es/Minimax/components/Mono';
import MistralColor from '@lobehub/icons/es/Mistral/components/Color';
import MistralMono from '@lobehub/icons/es/Mistral/components/Mono';
import MoonshotMono from '@lobehub/icons/es/Moonshot/components/Mono';
import OllamaMono from '@lobehub/icons/es/Ollama/components/Mono';
import OpenAIMono from '@lobehub/icons/es/OpenAI/components/Mono';
import OpenRouterMono from '@lobehub/icons/es/OpenRouter/components/Mono';
import PerplexityColor from '@lobehub/icons/es/Perplexity/components/Color';
import PerplexityMono from '@lobehub/icons/es/Perplexity/components/Mono';
import QingyanColor from '@lobehub/icons/es/Qingyan/components/Color';
import QingyanMono from '@lobehub/icons/es/Qingyan/components/Mono';
import QwenColor from '@lobehub/icons/es/Qwen/components/Color';
import QwenMono from '@lobehub/icons/es/Qwen/components/Mono';
import ReplicateMono from '@lobehub/icons/es/Replicate/components/Mono';
import SiliconCloudColor from '@lobehub/icons/es/SiliconCloud/components/Color';
import SiliconCloudMono from '@lobehub/icons/es/SiliconCloud/components/Mono';
import SparkColor from '@lobehub/icons/es/Spark/components/Color';
import SparkMono from '@lobehub/icons/es/Spark/components/Mono';
import SunoMono from '@lobehub/icons/es/Suno/components/Mono';
import VolcengineColor from '@lobehub/icons/es/Volcengine/components/Color';
import VolcengineMono from '@lobehub/icons/es/Volcengine/components/Mono';
import WenxinColor from '@lobehub/icons/es/Wenxin/components/Color';
import WenxinMono from '@lobehub/icons/es/Wenxin/components/Mono';
import XaiMono from '@lobehub/icons/es/XAI/components/Mono';
import XinferenceColor from '@lobehub/icons/es/Xinference/components/Color';
import XinferenceMono from '@lobehub/icons/es/Xinference/components/Mono';
import YiColor from '@lobehub/icons/es/Yi/components/Color';
import YiMono from '@lobehub/icons/es/Yi/components/Mono';
import ZhipuColor from '@lobehub/icons/es/Zhipu/components/Color';
import ZhipuMono from '@lobehub/icons/es/Zhipu/components/Mono';

const createAvatar = (Mono, Color) => {
  const Avatar = ({
    size = 24,
    shape = 'circle',
    background = 'var(--semi-color-fill-0)',
    color = 'currentColor',
    style,
    ...rest
  }) => {
    const Icon = Color || Mono;
    const numericSize = Number(size) || 24;
    return (
      <span
        {...rest}
        style={{
          alignItems: 'center',
          background,
          borderRadius: shape === 'square' ? Math.max(4, numericSize * 0.1) : '50%',
          color,
          display: 'inline-flex',
          flex: 'none',
          height: numericSize,
          justifyContent: 'center',
          lineHeight: 1,
          overflow: 'hidden',
          width: numericSize,
          ...style,
        }}
      >
        <Icon size={Math.round(numericSize * 0.72)} />
      </span>
    );
  };
  return Avatar;
};

const createIcon = (Mono, Color) => {
  const Icon = (props) => <Mono {...props} />;
  if (Color) {
    Icon.Color = (props) => <Color {...props} />;
  }
  Icon.Avatar = createAvatar(Mono, Color);
  return Icon;
};

export const Ai360 = createIcon(Ai360Mono, Ai360Color);
export const AzureAI = createIcon(AzureAIMono, AzureAIColor);
export const Claude = createIcon(ClaudeMono, ClaudeColor);
export const Cloudflare = createIcon(CloudflareMono, CloudflareColor);
export const Cohere = createIcon(CohereMono, CohereColor);
export const Coze = createIcon(CozeMono);
export const DeepSeek = createIcon(DeepSeekMono, DeepSeekColor);
export const Dify = createIcon(DifyMono, DifyColor);
export const Doubao = createIcon(DoubaoMono, DoubaoColor);
export const Fal = createIcon(FalMono, FalColor);
export const FastGPT = createIcon(FastGPTMono, FastGPTColor);
export const Gemini = createIcon(GeminiMono, GeminiColor);
export const Grok = createIcon(GrokMono);
export const Hunyuan = createIcon(HunyuanMono, HunyuanColor);
export const Jimeng = createIcon(JimengMono, JimengColor);
export const Jina = createIcon(JinaMono);
export const Kling = createIcon(KlingMono, KlingColor);
export const Midjourney = createIcon(MidjourneyMono);
export const Minimax = createIcon(MinimaxMono, MinimaxColor);
export const Mistral = createIcon(MistralMono, MistralColor);
export const Moonshot = createIcon(MoonshotMono);
export const Ollama = createIcon(OllamaMono);
export const OpenAI = createIcon(OpenAIMono);
export const OpenRouter = createIcon(OpenRouterMono);
export const Perplexity = createIcon(PerplexityMono, PerplexityColor);
export const Qingyan = createIcon(QingyanMono, QingyanColor);
export const Qwen = createIcon(QwenMono, QwenColor);
export const Replicate = createIcon(ReplicateMono);
export const SiliconCloud = createIcon(SiliconCloudMono, SiliconCloudColor);
export const Spark = createIcon(SparkMono, SparkColor);
export const Suno = createIcon(SunoMono);
export const Volcengine = createIcon(VolcengineMono, VolcengineColor);
export const Wenxin = createIcon(WenxinMono, WenxinColor);
export const XAI = createIcon(XaiMono);
export const Xinference = createIcon(XinferenceMono, XinferenceColor);
export const Yi = createIcon(YiMono, YiColor);
export const Zhipu = createIcon(ZhipuMono, ZhipuColor);

export const LobeIcons = {
  Ai360,
  AzureAI,
  Claude,
  Cloudflare,
  Cohere,
  Coze,
  DeepSeek,
  Dify,
  Doubao,
  Fal,
  FastGPT,
  Gemini,
  Grok,
  Hunyuan,
  Jimeng,
  Jina,
  Kling,
  Midjourney,
  Minimax,
  Mistral,
  Moonshot,
  Ollama,
  OpenAI,
  OpenRouter,
  Perplexity,
  Qingyan,
  Qwen,
  Replicate,
  SiliconCloud,
  Spark,
  Suno,
  Volcengine,
  Wenxin,
  XAI,
  Xinference,
  Yi,
  Zhipu,
};
