# 构建期前端皮肤系统设计

## 1. 背景

当前 `web/default` 是一个 React 19 单页应用，使用 TanStack Router 文件路由、React Query、Zustand、i18next、Base UI、shadcn/ui 风格组件和 Tailwind CSS v4。

项目已经具备浅层主题能力：

- `ThemeProvider` 管理 light、dark 和 system 模式；
- `ThemeCustomizationProvider` 管理配色预设、圆角、缩放和内容宽度；
- `theme.css` 与 `theme-presets.css` 通过语义 CSS 变量实现视觉切换；
- `PublicLayout` 与 `AuthenticatedLayout` 区分公开页面和登录后应用外壳。

这些能力适合颜色和密度调整，但不足以支持完整产品级皮肤。首页、模型广场和其他 feature 仍直接拥有页面结构、组件组合和局部视觉实现。当前约有 70 个 TypeScript/TSX 文件使用具体色阶类，说明仅覆盖全局 token 无法完整改变产品外观。

本设计增加一套构建期皮肤插件架构，使新的完整产品皮肤可以重做主要公开业务页面、提供独立公共外壳并增加独占路由，同时继续复用原项目的后端接口、登录状态、国际化和业务规则。

## 2. 已确认需求

### 2.1 生效方式

- 每次构建或部署固定选择一套皮肤；
- 不提供用户在线切换；
- 一个部署只产生一个 SPA 构建产物；
- 初始选择方式使用构建变量，例如 `APP_SKIN=custom`。

### 2.2 皮肤覆盖范围

新皮肤覆盖主要公开业务页面：

- `/`；
- `/pricing`；
- `/pricing/$modelId`；
- `/compare/ai-api-pricing`；
- `/rankings`；
- `/about`；
- `/privacy-policy`；
- `/user-agreement`；
- 后续新增的其他主要公开业务页面。

以下页面继续使用原实现，不纳入皮肤覆盖：

- 认证流程：`/sign-in`、`/sign-up`、`/forgot-password`、`/reset`、`/user/reset`、`/otp` 和 OAuth 回调；
- 系统初始化：`/setup`；
- 异常页面：`/401`、`/403`、`/404`、`/500`、`/503`；
- 登录后的用户控制台和管理后台。

### 2.3 路由扩展

- 皮肤可以提供宿主不存在的新公开页面；
- 皮肤可以声明任意合法路径，例如 `/solutions`、`/compare`、`/guides/$slug`；
- 新路由应继续具备 TanStack Router 的类型检查、懒加载、搜索参数校验和 404 行为；
- 皮肤不得覆盖认证、系统、异常和登录后受保护路由。

### 2.4 代码隔离

- 允许对原项目增加一次性稳定接入点；
- 允许按需抽取共享业务层；
- 后续开发皮肤页面时，不应持续修改原页面实现；
- 不要求严格零修改，但要求依赖方向清晰并避免两套业务规则漂移。

## 3. 方案选择

### 3.1 不采用：仅 CSS Token 主题

仅扩展现有主题预设不能改变页面结构、组件组合、交互方式或路由集合，无法满足完整产品皮肤要求。

### 3.2 不采用：完全独立的第二个 SPA

独立 SPA 隔离最强，但会让认证、异常页面和公开页面分属两个应用。部署需要按路径拆分静态资源或反向代理，跨应用导航、缓存、错误边界和构建发布都更复杂，并容易复制认证与基础设施代码。

### 3.3 采用：宿主内的构建期皮肤插件

保留 `web/default` 作为唯一宿主应用。构建期加载一份 `ThemeManifest`，公开业务路由通过皮肤运行时选择页面；皮肤独占路由在构建前生成 TanStack 文件路由代理。

这个方案满足：

- 单 SPA、单构建产物；
- 认证和系统页面继续使用原实现；
- 公开页面可以彻底重写；
- 皮肤可以增加新路由；
- 新旧页面可以共享同一份业务规则；
- 默认前端可以继续独立构建和发布。

## 4. 总体架构

```text
Browser URL
    |
    v
Existing TanStack Router
    |
    +-- reserved route -----------------> existing auth/system/error page
    |
    +-- public overridable route --------> active skin page
    |
    +-- generated skin-only route -------> active skin route component

Skin page
    |
    v
Shared public-domain contracts
    |
    v
Existing API, stores, React Query, i18n and backend
```

依赖方向必须保持单向：

```text
skin pages -> public domain contracts -> shared infrastructure -> backend API
```

宿主只能加载皮肤的公开 manifest 和契约，不能依赖某个皮肤内部页面或组件。原有 feature 也不能直接 import `skins/custom` 的内部实现。

## 5. 推荐目录

```text
web/default/src/
├── domains/
│   └── public-site/
│       ├── system-config/
│       ├── model-catalog/
│       ├── pricing/
│       ├── rankings/
│       └── legal-content/
├── skins/
│   ├── runtime/
│   │   ├── active-skin.ts
│   │   ├── contracts.ts
│   │   ├── reserved-routes.ts
│   │   ├── skin-boundary.tsx
│   │   └── skin-page.tsx
│   ├── default/
│   │   └── manifest.ts
│   └── custom/
│       ├── manifest.ts
│       ├── shell/
│       ├── pages/
│       ├── routes/
│       ├── ui/
│       ├── styles/
│       └── assets/
├── routes/
├── features/
├── stores/
└── lib/

web/default/scripts/
└── generate-skin-routes.mjs
```

`default/manifest.ts` 是原公开页面的适配器。选择 default 皮肤时，行为和当前应用保持一致。

## 6. ThemeManifest 契约

manifest 只声明皮肤能力，不承载 API 请求或业务实现。

示意类型：

```ts
type PublicPageKey =
  | 'home'
  | 'pricing'
  | 'modelDetails'
  | 'rankings'
  | 'about'
  | 'privacyPolicy'
  | 'userAgreement'

type SkinRouteContribution = {
  path: string
  component: string
  validateSearch?: string
  loader?: string
  navigation?: {
    labelKey: string
    position: 'header' | 'footer'
    order?: number
  }
  seo?: {
    titleKey: string
    descriptionKey?: string
  }
}

type ThemeManifest = {
  id: string
  pages: Partial<Record<PublicPageKey, React.LazyExoticComponent<React.ComponentType>>>
  routes: SkinRouteContribution[]
  shell: {
    layout: React.ComponentType
    header: React.ComponentType
    footer: React.ComponentType
  }
  navigation: SkinNavigationConfig
}
```

实际实现应避免在 JSON 中保存组件，manifest 是受 TypeScript 检查的源码模块。构建脚本只读取可静态分析的路由元数据。

## 7. 页面覆盖机制

已有公开路由仍由宿主拥有路径，只进行一次性委托改造。

例如 `/pricing` 的路由文件继续负责搜索参数 schema，但页面组件改为稳定出口：

```text
routes/pricing/index.tsx
    -> SkinPage page="pricing"
        -> activeSkin.pages.pricing
        -> defaultSkin.pages.pricing when no override exists
```

规则如下：

- 路由 schema、参数类型和根级错误处理属于宿主；
- 页面视觉和组合属于皮肤；
- 如果 active skin 未覆盖某个允许覆盖的公开页面，则回退到 default manifest；
- 认证、系统、异常和受保护路由不经过 `SkinPage`；
- 生产构建不得对声明了但无法导入的页面静默回退。

## 8. 新增路由机制

TanStack Router 当前通过 `@tanstack/router-plugin` 扫描 `src/routes` 并生成 `routeTree.gen.ts`。因此新增皮肤路由采用构建期代码生成，不使用运行时 catch-all 或任意动态注入。

构建流程：

1. 读取 `APP_SKIN`；
2. 解析对应 `manifest.ts` 的静态路由定义；
3. 校验路径、组件引用、参数和导航目标；
4. 在受控的 generated 目录生成 TanStack 文件路由代理；
5. 运行现有 TanStack Router 插件生成类型路由树；
6. 执行 TypeScript 检查和 Rsbuild 构建。

生成文件必须：

- 带有明确的 generated 标记；
- 不由开发者手工编辑；
- 在每次生成前清理旧的皮肤路由文件；
- 不进入长期源码维护范围；
- 只能 import active skin 的公开 route module。

构建校验必须拒绝：

- 与已有宿主路由重复的新路径；
- 覆盖 `/sign-in`、`/setup`、异常页面或 `_authenticated` 子树；
- 重复的皮肤路由；
- 不存在的导航目标；
- 非法动态参数；
- 无法解析的 route component、loader 或 search schema。

## 9. 共享业务层

共享层只承载与视觉无关、需要保持单一事实来源的能力。

应共享：

- API 请求函数与 React Query keys；
- TypeScript DTO 和领域类型；
- 模型元数据转换；
- 价格、计费表达式、单位和汇率计算；
- 搜索、筛选、排序等纯逻辑；
- 站点配置、用户登录状态和语言状态；
- 排行榜数据转换；
- 法律内容获取；
- SEO 元数据生成规则。

不应共享：

- 页面 JSX；
- Header、Footer、Hero 和布局组合；
- 模型卡片、筛选面板和详情面板外观；
- 页面专属 Tailwind class；
- 品牌图片、字体和动画；
- 皮肤独占交互。

迁移采用按需抽取：新皮肤需要某项能力时，先给现有行为补充或确认回归测试，再把无 UI 逻辑移动到 `domains/public-site`，最后让旧页面和新皮肤共同消费。禁止先创建一个覆盖全部 feature 的大而全共享框架。

## 10. 样式隔离

认证和系统页面必须保持默认视觉，因此皮肤不能直接覆盖全局 `:root`、`body`、`.dark` 或通用 shadcn token。

所有皮肤公开页面通过 `SkinBoundary` 渲染：

```tsx
<div data-skin="custom">
  <SkinPortalProvider>
    <CustomPublicShell>{page}</CustomPublicShell>
  </SkinPortalProvider>
</div>
```

样式分三层：

1. Design tokens：挂在 `[data-skin='custom']` 下的颜色、字体、圆角、间距和阴影；
2. Skin UI：位于 `skins/custom/ui` 的按钮、卡片、区块和领域展示组件；
3. Page composition：只使用皮肤 UI 和共享业务 hooks 的页面组合。

约束：

- custom CSS 必须被 `[data-skin='custom']` 限定；
- 不为皮肤修改 `src/components/ui` 的通用组件；
- 可以复用 Base UI 或 shadcn primitives 的行为能力，但皮肤负责自己的视觉包装；
- Dialog、Popover、Tooltip、Select 等 Portal 浮层必须挂到 `SkinBoundary` 创建的 portal root；
- 离开皮肤路由时必须清理所有临时 body attribute 和 portal container；
- dark mode 和语言切换由宿主状态驱动，皮肤仅提供对应 token 和内容。

## 11. 导航、SEO 与公开外壳

皮肤 manifest 是公开导航的单一配置来源：

- Header 和 Footer 从同一配置读取链接；
- 皮肤独占路由可贡献导航项；
- 导航 label 使用 i18n key；
- 构建校验确保内部链接存在；
- 外部链接必须显式标记；
- 公开页面 SEO 默认值由皮肤提供，路由参数页面可通过共享 SEO helper 生成动态元数据。

后台已有的 Header 模块开关和 `requireAuth` 仍是业务约束。皮肤导航适配层必须消费该配置，不能因为重做 Header 而绕过管理员对模型广场、排行榜等入口的控制。

## 12. 错误与回退策略

### 12.1 构建期

- 未知 `APP_SKIN`：构建失败；
- manifest schema 无效：构建失败；
- 路由冲突或组件引用无效：构建失败；
- 导航目标不存在：构建失败；
- generated 文件与 manifest 不一致：构建失败。

### 12.2 运行时

- active skin 未覆盖允许覆盖的公开页面：使用 default manifest 对应页面；
- 页面 lazy import 失败：进入宿主根路由错误边界；
- 共享 API 请求失败：由共享查询契约提供错误状态，皮肤负责渲染符合自身视觉的错误 UI；
- 不为皮肤创建独立认证错误处理，401/403 继续遵循宿主现有逻辑；
- 皮肤新增路径未匹配时继续进入宿主 404 页面。

## 13. 实施阶段

### 阶段 1：皮肤运行时

- 定义 contracts、reserved routes 和 manifest validation；
- 增加 `APP_SKIN` 构建选择；
- 实现 default manifest；
- 增加公开页面稳定出口；
- 页面外观和行为保持不变。

### 阶段 2：首批共享业务层

- 抽取 system config；
- 抽取 model catalog 和 pricing contracts；
- 抽取价格、筛选、排序和模型元数据逻辑；
- 让原模型广场继续使用抽取后的共享模块。

### 阶段 3：custom 公共外壳与首页

- 实现 SkinBoundary 和 portal root；
- 实现 custom tokens、Header、Footer 和 layout；
- 实现 custom 首页；
- 验证认证和异常页面无视觉变化。

### 阶段 4：模型广场与详情

- 实现 custom 模型列表、搜索、筛选和排序；
- 实现模型详情；
- 保留 URL search 参数和深链接行为；
- 覆盖 loading、empty、error 和移动端状态。

### 阶段 5：其余页面与新路由

- 排行榜；
- 关于；
- 隐私政策与用户协议；
- `/compare/ai-api-pricing` 兼容路径；
- 皮肤独占新路由和导航贡献。

## 14. 测试与验收

### 14.1 自动化测试

- manifest schema 单元测试；
- reserved route 冲突测试；
- generated route snapshot 或结构测试；
- 导航目标完整性测试；
- 价格、筛选、排序和数据转换单元测试；
- default manifest 页面回退测试；
- SkinBoundary attribute 和 cleanup 测试；
- portal root 测试；
- `APP_SKIN=default` 类型检查与生产构建；
- `APP_SKIN=custom` 类型检查与生产构建；
- ESLint、format check 和 `git diff --check`。

### 14.2 浏览器验收

- desktop 和 mobile 首页；
- 模型广场搜索、筛选、排序、卡片/表格与详情；
- 模型详情直接访问和返回；
- 排行榜、关于和法律页面；
- 皮肤独占新路由和刷新恢复；
- light/dark 模式；
- 所有支持语言；
- Header/Footer 内部和外部链接；
- Dialog、Popover、Tooltip 等浮层样式；
- `/sign-in`、`/setup`、`/404`、`/500` 与默认构建视觉对比；
- 从皮肤页面跳转到认证页面后无残留 class、attribute 或 portal。

## 15. 主要风险

### 15.1 共享层抽取过度

控制方式：按新皮肤真实需求抽取，一次只抽一个可测试的业务边界。

### 15.2 新旧业务逻辑漂移

控制方式：API、类型和业务计算只保留一份，旧页面与皮肤页面共同依赖。

### 15.3 样式泄漏

控制方式：SkinBoundary、作用域 CSS、专属 portal root 和路由离开清理测试。

### 15.4 路由代码生成复杂度

控制方式：manifest 只允许有限、明确的路由字段；生成器保持确定性；生成结果纳入结构测试；禁止运行时 catch-all 替代。

### 15.5 默认前端退化

控制方式：CI 永远同时构建 default 和 custom，不允许只验证新皮肤。

## 16. 成功标准

架构实施完成后应满足：

1. default 构建的公开页面、认证页面和控制台行为保持不变；
2. custom 构建可以完全重做主要公开业务页面；
3. custom 可以只修改自身目录来新增公开页面和导航；
4. custom 不复制模型、价格、筛选和站点配置业务规则；
5. 认证、系统和异常页面不受 custom 样式影响；
6. 两套构建都通过类型、测试、Lint 和生产构建；
7. 新增第三套皮肤时不需要再次修改宿主路由和业务 feature，只需实现 manifest、页面与样式契约。

## 17. 非目标

- 用户运行时切换皮肤；
- 多租户或按域名动态选择皮肤；
- 后端下发可执行前端组件；
- 重做认证、系统初始化、异常页面或登录后控制台；
- 一次性重构全部 `features`；
- 建立跨仓库或可发布 npm 的通用主题框架。
