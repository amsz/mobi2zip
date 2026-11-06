# 漫画解包

把漫画电子书里的图片抽出来，重新打包成同名 `.zip` 的 Windows 桌面工具。

## 依赖

| 依赖 | 说明 |
|---|---|
| [calibre](https://calibre-ebook.com/) | 处理 `.mobi` 和 `.pdf` 需要；只转 `.epub` 不需要。设置里可验证路径 |

## 构建

需要 Go 1.27+：

```bash
go tool mygo dev     # 开发运行
go tool mygo build   # 打包到 build/windows-amd64（exe + NSIS 安装包）
go test ./...
```

首次构建会拉取 MyGo 依赖；若 `proxy.golang.org` 不通：

```bash
GOPROXY=https://goproxy.cn,direct go tool mygo build
```

## 配置

设置保存在用户数据目录：

```
%APPDATA%\漫画解包\settings.json
```

（用 `go build` 出来的裸 exe 运行时，目录名取 exe 名 `mobi2zip`。）

字段：

```json
{
  "archive": "D:\\归档",
  "calibrePath": "C:\\Program Files\\Calibre2\\ebook-convert.exe",
  "notify": true
}
```

## 目录结构

```
.
├─ main.go                 # 入口
├─ mygo.json               # 应用元信息（名称、标识、版本）
├─ resources/icon.png      # 应用图标（1024×1024，构建时生成 .ico/.icns）
├─ icon/make_icons.py      # 图标生成脚本（Pillow）
├─ internal/
│  ├─ app/                 # 界面与状态
│  │  ├─ app.go            # 应用状态、post、窗口
│  │  ├─ ui_main.go        # 主窗口：拖放区、列表、底栏
│  │  ├─ ui_settings.go    # 设置窗口
│  │  ├─ ui_kit.go         # 配色与小组件
│  │  ├─ ui_icons.go       # SVG 图标
│  │  ├─ model.go          # 分组/文件模型与扫描
│  │  ├─ convert.go        # 转换队列（2 并发）
│  │  ├─ archive.go        # 归档移动
│  │  ├─ settings.go       # 设置读写
│  │  └─ dev.go            # 调试用的环境变量注入
│  └─ convert/             # 转换内核
│     ├─ convert.go        # 管线与 zip 打包
│     └─ epub.go           # EPUB 解析（spine 页序取图）
└─ docs/screenshot.png
```

## 开发辅助

设置环境变量后启动，可以不用手动拖拽就检查界面：

| 变量 | 作用 |
|---|---|
| `MOBI2ZIP_DEMO=<目录>` | 启动时把该目录加入列表 |
| `MOBI2ZIP_DEMO_FAKE=1` | 把列表里的文件标记成各种结束状态，检查图标 |
| `MOBI2ZIP_DEMO_AUTOSTART=1` | 加入列表后自动开始转换 |
