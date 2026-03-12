# WinClaw AI Agent

一个运行在Windows上的AI Agent，类似于OpenClaw。

## 功能特性

- 🤖 文字对话 - 与AI助手交流
- 🎤 语音交互 - 支持语音输入输出
- 📁 文件操作 - 读取、写入、列出目录
- 🌐 浏览器控制 - 打开网页
- 📝 Word文档 - 创建和编辑文档
- 🔧 Skills扩展 - 支持Python技能扩展
- ⚙️ 权限控制 - 多级权限管理

## 运行方式

### 开发模式
```bash
wails dev
```

### 构建发布
```bash
wails build
```

构建后的可执行文件位于 `build/bin/WinClaw.exe`

## 首次配置

1. 运行 WinClaw.exe
2. 点击右上角设置按钮
3. 在AI设置中输入你的OpenAI API Key
4. 选择合适的权限级别

## 权限级别

- **低**: 仅对话和查询
- **中**: 允许浏览器、文件查看
- **高**: 允许大部分操作，需要确认
- **完全控制**: 所有操作自动执行

## Skills

内置Skills:
- file_read - 读取文件
- file_write - 写入文件
- file_list - 列出目录
- system_info - 系统信息
- calculator - 计算器

自定义Python Skills放在 `skills/python/` 目录，兼容openclaw。

## 技术栈

- 后端: Go + Wails
- 前端: React + TypeScript + TailwindCSS
- AI: OpenAI API
"# winclaw" 
