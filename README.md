# SMSBoomGUI

<p align="center">
<img src="https://socialify.git.ci/JDDKCN/SMSBoomGUI/image?description=1&forks=1&issues=1&language=1&logo=https%3A%2F%2Favatars.githubusercontent.com/u/103011451?v=4&name=1&owner=1&pulls=1&stargazers=1&theme=Light" align="center" alt="Github Stats" />
</p>

---
## 软件截图
- v1.3.0 - 2023/03/01
![Pic1](https://raw.githubusercontent.com/JDDKCN/SMSBoomGUI/main/Resources/APP01.png)

---
## 使用方法
 1. 第一次使用请先到设置界面点击升级接口(更新API)
 2. 填入手机号 (目前暂时只支持 +86 区号的电话号码)
 3. 选择线程数量/执行次数/执行间隔
 4. 点击一键启动服务运行程序

## Linux 命令行二进制

项目新增了一个跨平台的命令行工具，便于在 Linux 环境下直接执行短信轰炸逻辑。

### 编译

```bash
go build -o smsboom cmd/smsboom/main.go
```

### 使用

```bash
./smsboom -p 13800138000 -a /path/to/api.json
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `-p` | 目标手机号（必填） |
| `-a` | API 定义文件路径（必填） |
| `-c` | 并发 worker 数量，默认 4 |
| `-n` | 执行完整 API 列表的次数，默认 1 |
| `--delay` | 每次请求之间的延迟，例如 `--delay=500ms` |
| `--dry-run` | 仅打印将要执行的请求而不真正发送 |
| `--placeholder` | API 文件中用于替换手机号的占位符，默认 `{{phone}}` |

API 文件需要是 JSON 格式，可以是数组或包含 `requests` 字段的对象。示例参见仓库根目录的 `api.sample.json`。

## 注意事项
- 运行时请务必关闭系统代理和代理软件(如Clash等)
- 若接口失效，请到设置界面升级接口再试。

## 版权及免责声明
- 本程序是基于Github开源项目 [SMSBoom](https://github.com/OpenEthan/SMSBoom) 制作的GUI图形化界面，遵循GPL开源协议。本程序完全免费，禁止用于商业及非法用途。使用本软件造成的事故与损失，与作者无关。
