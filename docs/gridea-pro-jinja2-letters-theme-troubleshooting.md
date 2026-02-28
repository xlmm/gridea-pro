# Gridea Pro Jinja2 主题开发实战：从模板报错到 CSS 断链的奇葩排查全记录

在为 Gridea Pro 适配 `letters-theme` (Jinja2 引擎) 主题时，我们遇到了一系列阻碍渲染与样式加载的顽固问题。本篇开发手记详细记录了本次从报错分析、渲染器源码审查到浏览器抓包追踪的全过程，相信这些"踩坑"经验对其他 Gridea Pro 的主题开发者会有很大帮助。

## 第一阶段：解决满屏的 "Unable to Resolve Template" 报错

在初始启动预览时，终端（Wails Dev）疯狂输出各种 `[Error (where: fromfile) in xxx] unable to resolve template`。页面大部分渲染失败。

### 问题 1：模板文件名与渲染层约定不符
**现象**：`archives.html` 和 `friends.html` 渲染失败。
**原因**：作者在主题目录中使用了 `archive.html` 和 `links.html`，但后端的 `renderer_pages.go` 中，调用的却是 `Render("archives", ...)` 和 `Render("friends", ...)`。
**解决方案**：
直接将主题文件夹中的 `archive.html` 重命名为 `archives.html`，将 `links.html` 重命名为 `friends.html`，以匹配 Gridea Pro 约定的标准文件命名规范。

### 问题 2：误用不存在的 `partials` 组件
**现象**：文章详情页 `post.html` 渲染崩溃。
**原因**：该主题内大量引用了 `{% include "partials/gitalk.html" %}` 以及 `disqus.html`，但是 `letters-theme` 的 partials 目录下根本就没有这几套评论组件的模板文件。
**解决方案**：
清理掉 `post.html` 中所有不存在的评论系统 `include` 语句。建议通过后端的 `commentSetting` 变量做条件判断加载。

### 问题 3：Jinja2 变量无法获取导致渲染失败
**现象**：某些模板的数据空白，或者访问 `config.siteName` 失效。
**原因**：
1. 后端传递的模板上下文（`Jinja2Renderer.buildContext`）遗漏了几个重要的上下文变量（如：`commentSetting`, `tag`, `links`）。
2. `config` 对象默认只映射了基础站点配置 `SiteView`，缺少了主题特有的 `ThemeConfig`。模板中的 `config.domain` 等取不到值。
3. `tag` 页面在模板中使用的变量名是 `currentTag`，而渲染器应当注入统一规范的 `tag`（或兼容名 `current_tag`）。
**解决方案**：
修改后端的 `jinja2_renderer.go`：
- 将缺失的 `CommentSettingView` 等变量包裹注入到 Pongo2 渲染上下文中。
- 将 `ThemeConfig` 的配置项合并至 `configValue` 中，供前端统一调用。

---

## 第二阶段：HTML 标签页正常了，但 CSS 就是加载不出来！

虽然终端的报错全部消失（除了 `blog.html`，因该主题不需要博客列表页，跳过是正常行为），但页面预览时仍然是裸奔的 HTML 骨架，没有任何 CSS 样式。

### 排查线索 1：资源打包路径不一致
**现象**：主题使用了 `assets/css/style.css`。但在其他能正常工作的主题（如 `amore-jinja2` 和 `simple`）中，通常有 `assets/styles/main.less` 并被系统编译输出为 `/styles/main.css`。
**解决方案**：
创建约定的目录结构。将 `css/style.css` 移至 `assets/styles/main.css`，适应 Gridea Pro 默认的静态资源输出规范。

### 排查线索 2：CSS 引用前缀 `cdnPrefix` 缺失
**现象**：`base.html` 中引用了 `{{ cdnPrefix }}/styles/main.css`，但是因为上下文没定义，路径变成了 `/styles/main.css`。Wails App 内置的 iframe 对于无主机域名的绝对路径会出现解析寻址的问题。
**解决方案**：
在 `base.html` 头部仿照其他主题，增加对 `cdnPrefix` 的默认回退声明：
```jinja2
{% set cdnPrefix = config.domain %}
{% if theme_config.cdnPrefix %}
{% set cdnPrefix = theme_config.cdnPrefix %}
{% endif %}
<link rel="stylesheet" href="{{ cdnPrefix }}/styles/main.css">
```

---

## 第三阶段：致命的 "幽灵" Bug：修改毫无反应

在完成了上述修复后，发现了一个极其恼人的问题：**输出目录的 CSS 文件已经更新并存在，但是在浏览器请求的却是旧的路径 `/css/style.css`！不管怎么刷新，改动仿佛没生效！**

### 排查热重载链路：被遗忘的模板缓存
顺着 `ResourceWatcher` 一路追查：
1. 监控器检测到文件变动，发出 `app-site-reload` 事件。
2. 事件总线监听到该事件，触发 `rendererFacade.RenderAll()`。
3. 进入 `renderer_service.go` 的 `SetTheme()` 阶段。

问题就在这里：
```go
// 缓存检查：如果渲染器已初始化且主题未变更，直接返回
if s.renderer != nil && s.currentTheme == themeName {
    return nil
}
```
**因为主题名称没变，系统直接使用之前实例化的 Jinja2 渲染器（自带模板缓存）！从而让热重载期间所有的模板文件修改彻底失效。**

**最终解决方案**：
为主题重载加入明确的缓存清除动作，确保开发阶段的所见即所得：
```go
if s.renderer != nil && s.currentTheme == themeName {
    s.renderer.ClearCache() // <-- 关键补丁：清除缓存！
    return nil
}
```

---

## 第四阶段：为何内联 CSS 全面崩坏？

缓存问题解决后，我们满心欢喜去浏览器验收。结果 CSS **还是半残的**。
打开输出页面的源代码，我们看到了惨不忍睹的一幕：

```css
:root {
  --color-primary: {
      {
      theme_config.primaryColor|default: "#222222"
    }
  }
}
```

### 罪魁祸首：Pongo2 对内联 Style 标签的解析局限
在 `base.html` 中，为了方便主题动态配色，作者写了这样的内联样式：
```html
<style>
  :root {
    --color-primary: {{ theme_config.primaryColor|default:"#222222" }};
  }
</style>
```

**原因分析**：Go 后端底层的 Pongo2 模板引擎，无法完全正确地转义和解析包裹在 `<style>` 或 `<script>` 标签内含有大段折行的特定复杂逻辑块（或与 CSS 原生 `{}` 形成冲突）。它最终把 Jinja2 标签作为字符串原文吐出了，直接导致这部分 CSS 解析错误，并污染了页面的样式加载池。

### 完美解法：使用 Gridea Pro 标准的动态注入 `style-override.js`
Gridea Pro 设计了优雅的主题配置动态注入机制。我们抛弃了容易因为预编译引发布局崩溃的内联标签方案，转而编写一个主题专属的 `style-override.js`。

在根目录创建 `style-override.js`：
```javascript
const generateOverride = (params = {}) => {
  let result = ''
  if (params.primaryColor) {
    result += `
      :root {
        --primary-color: ${params.primaryColor};
      }
    `
  }
  return result
}
module.exports = generateOverride
```

利用系统会在构建时统一计算并将其编译压缩的特性，动态主题色得到完美加载！

## 总结
通过本次“填坑血泪史”，我们可以总结出 Gridea Pro Jinja2 主题开发的四大军规：
1. **严格遵循系统目录树结构和模板文件名约定**（如 `archives.html`, `friends.html`, 资源丢进 `assets/styles/`）。
2. **谨慎使用未定义或者未兜底的变量**，时刻留意后端的 context 注入情况。
3. **永远警惕构建缓存**，如果怀疑输出未更新，先看后端的重新编译或缓存清理机制。
4. **拒绝把模板逻辑硬塞进内联的 Style 或 Script 标签**，统统使用 `style-override.js` 来完成高级的定制化展现。
