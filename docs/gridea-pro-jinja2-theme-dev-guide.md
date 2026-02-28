---
title: Gridea Pro Jinja2 主题开发完全指南：从踩坑到精通
date: "2026-02-28 03:15:00"
tags:
    - Jinja2
    - Gridea Pro
    - Pongo2
    - Go
    - 主题开发
    - 教程
tag_ids:
    - xdJe6
    - 2osyp
    - 1YSU7
    - U0eE0M
    - nOFYP
    - kWd6a
categories:
    - 技术
published: true
hideInList: false
feature: /post-images/gridea-pro-jinja2-theme-dev-guide.png
isTop: false
---

Gridea Pro 从 v2 开始引入了 Jinja2 模板引擎支持（基于 Go 语言的 Pongo2 实现），为主题开发者提供了比 EJS 更优雅的模板继承和组件化能力。但由于 Pongo2 与标准 Python Jinja2 存在细微但关键的语法差异，初次开发者很容易踩坑。

本文基于一次完整的 EJS 到 Jinja2 主题迁移实战（amore 主题），系统性地总结了所有已知的兼容性陷阱和最佳实践，帮助你从零开始写出一个零报错的 Jinja2 主题。

<!-- more -->

---

## 一、主题目录结构

一个标准的 Jinja2 主题目录结构如下：

```
themes/my-theme/
|-- config.json              # 主题配置文件
|-- assets/                  # 静态资源
|   |-- media/
|   |   +-- images/
|   |       |-- avatar.jpg
|   |       +-- favicon.ico
|   +-- styles/
|       +-- main.less
+-- templates/               # 模板文件（核心）
    |-- base.html            # 根布局（定义 block 占位符）
    |-- index.html           # 首页
    |-- post.html            # 文章详情页
    |-- archives.html        # 归档页
    |-- tag.html             # 标签详情页
    |-- tags.html            # 标签列表页
    |-- about.html           # 关于页
    |-- friends.html         # 友链页
    |-- blog.html            # 博客列表页
    |-- memos.html           # 闪念页
    |-- 404.html             # 404 页面
    +-- partials/            # 可复用组件
        |-- head.html        # HTML head（SEO、CSS、JS）
        |-- header.html      # 导航栏
        |-- footer.html      # 页脚
        |-- comments.html    # 评论系统
        |-- post-list.html   # 文章列表组件
        |-- post-tags.html   # 文章标签
        |-- post-pagination.html # 上下篇导航
        |-- global-seo.html  # 全局 SEO 结构化数据
        |-- index-seo.html   # 首页 SEO
        |-- scripts.html     # 公共脚本
        +-- reading-progress.html # 阅读进度条
```

### config.json 配置

```json
{
  "name": "My Theme",
  "version": "1.0.0",
  "author": "Your Name",
  "engine": "jinja2",
  "customConfig": [
    {
      "name": "siteName",
      "label": "站点名称",
      "group": "基础",
      "value": "My Blog",
      "type": "input"
    }
  ]
}
```

> **关键**：`"engine": "jinja2"` 是必须的，它告诉 Gridea Pro 使用 Pongo2 引擎而非 EJS。

---

## 二、模板数据上下文

Gridea Pro 会向模板注入以下数据变量，你可以在任何模板中直接使用：

### 全局变量

| 变量名 | 类型 | 说明 |
|--------|------|------|
| `config` | Object | 站点配置（domain, siteName 等） |
| `theme_config` | Object | 主题自定义配置 |
| `menus` | Array | 导航菜单列表 |
| `posts` | Array | 文章列表 |
| `tags` | Array | 标签列表 |
| `memos` | Array | 闪念列表 |
| `now` | time.Time | 当前时间（Go 原生 time.Time 对象） |

### 文章页专有变量

| 变量名 | 类型 | 说明 |
|--------|------|------|
| `post` | Object | 当前文章 |
| `post.title` | String | 文章标题 |
| `post.content` | String | 文章 HTML 内容 |
| `post.date` | String | 发布日期（已格式化字符串） |
| `post.dateFormat` | String | 格式化日期显示 |
| `post.link` | String | 文章链接 |
| `post.tags` | Array | 文章标签列表 |
| `post.feature` | String | 特色图片 URL |
| `post.isTop` | Boolean | 是否置顶 |

### 标签页专有变量

| 变量名 | 类型 | 说明 |
|--------|------|------|
| `tag` | Object | 当前标签（tag.name, tag.link） |
| `current_tag` | Object | 同 tag |

### 分页变量

| 变量名 | 类型 | 说明 |
|--------|------|------|
| `pagination` | Object | 分页信息 |
| `pagination.prev` | String | 上一页链接 |
| `pagination.next` | String | 下一页链接 |

---

## 三、Pongo2 核心语法速查

### 3.1 模板继承

Jinja2 最强大的特性。定义一个基础布局，子模板继承并填充内容。

**base.html（基础布局）：**

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    {% include "partials/head.html" %}
    <title>{% block title %}{{ config.siteName }}{% endblock %}</title>
</head>
<body>
    {% block content %}{% endblock %}
</body>
</html>
```

**index.html（子模板）：**

```html
{% extends "base.html" %}
{% block title %}首页 | {{ config.siteName }}{% endblock %}

{% block content %}
<h1>欢迎</h1>
{% endblock %}
```

### 3.2 Include 组件

```html
{% include "partials/header.html" %}
{% include "partials/footer.html" %}
```

> **注意**：路径始终相对于 `templates/` 根目录，不是相对于当前文件。例如在 `partials/head.html` 中 include 同目录的 `global-seo.html`，仍然要写 `{% include "partials/global-seo.html" %}`。

### 3.3 变量输出

```html
{{ config.siteName }}
{{ post.title }}
{{ post.content | safe }}
```

`| safe` 用于输出原始 HTML 而不转义。

### 3.4 条件判断

```html
{% if post %}
    <h1>{{ post.title }}</h1>
{% else %}
    <h1>默认标题</h1>
{% endif %}
```

### 3.5 循环

```html
{% for post in posts %}
    <li>{{ post.title }}</li>
{% endfor %}
```

循环中可使用 `loop.index`（从1开始）和 `loop.index0`（从0开始）。

### 3.6 Set 变量

```html
{% set cdnPrefix = config.domain %}
{% if theme_config.cdnPrefix %}
    {% set cdnPrefix = theme_config.cdnPrefix %}
{% endif %}
```

---

## 四、关键差异：Pongo2 vs 标准 Jinja2

这是 **最重要的章节**。Pongo2 虽然号称兼容 Jinja2，但有多处语法不兼容。以下是我们在实战中踩过的每一个坑：

### 4.1 Filter 参数语法

标准 Jinja2 使用括号传参，Pongo2 使用冒号：

```
错误（标准 Jinja2 写法）：
  {{ value | default("fallback") }}
  {{ value | truncate(100) }}

正确（Pongo2 写法）：
  {{ value | default:"fallback" }}
  {{ value | truncate:100 }}
```

### 4.2 没有 Python 三元表达式

标准 Jinja2 支持 `value if condition else other`，Pongo2 不支持：

```
错误：
  {{ post.title if post else '默认' }}

正确：
  {% if post %}{{ post.title }}{% else %}默认{% endif %}
```

### 4.3 逻辑运算符

```
错误（JavaScript 风格）：
  {% if a && b %}
  {% if typeof pagination !== 'undefined' %}

正确（Python 风格）：
  {% if a and b %}
  {% if pagination %}
```

### 4.4 字符串连接

标准 Jinja2 使用 `~` 连接字符串，Pongo2 不支持。建议直接在输出位置拼接：

```
错误：
  {% set title = post.title ~ ' | ' ~ config.siteName %}

正确：
  <title>{{ post.title }} | {{ config.siteName }}</title>
```

### 4.5 列表长度

Pongo2 不支持 JavaScript 的 `.length`，使用 `|length` filter：

```
错误：{% if tags.length > 0 %}
正确：{% if tags|length > 0 %}
```

### 4.6 is defined 测试

Pongo2 不支持 `is defined`，直接用变量做条件判断：

```
错误：{% if commentSetting is defined %}
正确：{% if commentSetting %}
```

### 4.7 not in 操作符

Pongo2 不支持 `x not in y`，需要拆开：

```
错误：{% if 'about' not in post.link %}
正确：{% if not "about" in post.link %}
```

### 4.8 date Filter 类型要求

Pongo2 的 `date` filter 严格要求输入必须是 Go 的 `time.Time` 类型，不能是字符串：

```
错误（post.date 是字符串，不是 time.Time）：
  {{ post.date | date:"2006-01-02" }}
  {{ "now" | date:"2006" }}

正确（post.date 已经是格式化好的字符串）：
  {{ post.date }}

正确（now 是 Gridea 注入的真正 time.Time 对象）：
  {{ now | date:"2006" }}
```

### 4.9 循环变量命名

for 循环中必须显式命名循环变量：

```
错误：{% for None in posts %}
正确：{% for post in posts %}
```

---

## 五、标签内换行符问题（已在引擎层面解决）

### 问题描述

标准 Jinja2 允许在标签内换行，但 Pongo2 的 Lexer 严格禁止：

```html
<!-- 这在标准 Jinja2 中合法，但在 Pongo2 中报错 -->
<a href="{{ config.domain }}">{{
    config.siteName
}}</a>
```

会触发错误：`Newline not allowed within tag/variable`

### 解决方案

Gridea Pro 已在引擎层面内置了 SanitizingLoader（自定义模板加载器），它在模板加载时自动清理标签内的换行符，对主题开发者完全透明。

但建议保持标签内容在同一行：

```html
<a href="{{ config.domain }}">{{ config.siteName }}</a>
```

---

## 六、Gridea Pro 内置 Filter

除了 Pongo2 本身的 filter（safe, default, length, lower, upper, striptags, date 等），Gridea Pro 还提供了以下自定义 filter：

| Filter | 用法 | 说明 |
|--------|------|------|
| reading_time | `{{ post.content \| reading_time }}` | 估算阅读时间（支持中文） |
| excerpt | `{{ post.content \| excerpt }}` | 截取摘要 |
| word_count | `{{ post.content \| word_count }}` | 统计字数（CJK 感知） |
| strip_html | `{{ content \| strip_html }}` | 移除 HTML 标签 |
| relative / timeago | `{{ post.date \| relative }}` | 相对时间（"3 天前"） |
| to_json | `{{ data \| to_json }}` | 序列化为 JSON |
| group_by | `{{ posts \| group_by:"year" }}` | 按属性分组 |

---

## 七、完整模板示例

### 7.1 base.html（根布局）

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    {% include "partials/head.html" %}
    <title>{% block title %}{{ config.siteName }}{% endblock %}</title>
</head>
<body>
    {% block content %}{% endblock %}
</body>
</html>
```

### 7.2 index.html（首页）

```html
{% extends "base.html" %}
{% block title %}{{ config.siteName }}{% endblock %}

{% block content %}
{% include "partials/header.html" %}

<div class="posts-list">
    {% for post in posts %}
    <article class="post-card">
        <h2><a href="{{ post.link }}">{{ post.title }}</a></h2>
        <time>{{ post.dateFormat }}</time>
        <p>{{ post.content | excerpt:200 }}</p>
    </article>
    {% endfor %}
</div>

{% if pagination %}
<nav class="pagination">
    {% if pagination.prev %}
    <a href="{{ pagination.prev }}">上一页</a>
    {% endif %}
    {% if pagination.next %}
    <a href="{{ pagination.next }}">下一页</a>
    {% endif %}
</nav>
{% endif %}

{% include "partials/footer.html" %}
{% endblock %}
```

### 7.3 post.html（文章页）

```html
{% extends "base.html" %}
{% block title %}{{ post.title }} | {{ config.siteName }}{% endblock %}

{% block content %}
{% include "partials/header.html" %}

<article>
    <h1>{{ post.title }}</h1>
    <div class="meta">
        <time>{{ post.dateFormat }}</time>
    </div>

    <div class="content">
        {{ post.content | safe }}
    </div>

    <div class="post-tags">
        {% for tag in post.tags %}
        <a href="{{ tag.link }}">#{{ tag.name }}</a>
        {% endfor %}
    </div>
</article>

{% include "partials/comments.html" %}
{% include "partials/footer.html" %}
{% endblock %}
```

### 7.4 partials/footer.html（页脚组件）

```html
<footer>
    <nav>
        {% for menu in menus %}
        <a href="{{ menu.link }}">{{ menu.name }}</a>
        {% endfor %}
    </nav>
    <div class="copyright">
        Copyright {{ now | date:"2006" }}
        <a href="{{ config.domain }}">{{ config.siteName }}</a>
    </div>
    <div class="footer-info">
        {{ theme_config.footerInfo | safe }}
    </div>
</footer>
```

---

## 八、从 EJS 迁移的检查清单

如果你正在将现有 EJS 主题迁移到 Jinja2，请逐项检查：

- 所有 `<% %>` 改为 `{% %}`
- 所有 `<%= %>` 改为 `{{ }}`
- 所有 `<%- %>` 改为 `{{ | safe }}`
- 所有 `include('./includes/xxx')` 改为 `{% include "partials/xxx.html" %}`
- 所有 `&&` 改为 `and`，`||` 改为 `or`
- 所有 `.length` 改为 `|length`
- 所有 `default("value")` 改为 `default:"value"`
- 所有 `forEach(function(item) { })` 改为 `{% for item in list %}`
- 所有 Python 三元表达式改为 `{% if %}...{% else %}...{% endif %}`
- 所有 `typeof x !== 'undefined'` 改为 `{% if x %}`
- 所有 `not in` 改为 `not ... in`
- 确认 `date` filter 只用于 time.Time 类型（如 now），不用于字符串
- 所有循环变量显式命名（避免 `for None in`）
- 根布局改用 `{% extends %}` + `{% block %}`

---

## 九、调试技巧

### 9.1 查看渲染日志

运行 `wails dev` 时，终端会输出每个模板的渲染状态。

### 9.2 常见错误信息速查

| 错误信息 | 原因 | 解决方案 |
|----------|------|---------|
| Newline not allowed within tag/variable | 标签内有换行 | 保持标签内容在一行内 |
| '}}' expected | 使用了 Python 三元表达式 | 改用 if/else 块 |
| Malformed 'set'-tag arguments | default() 括号语法 | 改为 default:value |
| If-condition is malformed | 使用了 not in | 改为嵌套 if not ... in |
| filter input argument must be of type 'time.Time' | 对字符串使用 date filter | 移除 filter 或使用 now 变量 |
| unable to resolve template | include/extends 路径错误 | 路径相对于 templates/ 根目录 |

---

## 十、总结

Gridea Pro 的 Jinja2 主题开发体验在经过优化后已经非常流畅。核心要点：

1. **Pongo2 不等于标准 Jinja2**：最大的区别是 filter 参数用冒号、不支持 Python 三元、不支持字符串连接符
2. **善用模板继承**：extends + block 是 Jinja2 最强大的特性
3. **include 路径始终相对于 templates/ 根目录**
4. **date filter 只用于 time.Time 类型**：post.date 是字符串，now 是 time.Time
5. **利用 Gridea Pro 的内置 filter**：reading_time、excerpt、word_count 等能为你的主题增色不少

祝你开发出精美的 Jinja2 主题！