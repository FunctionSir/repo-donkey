<!--
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2024-11-18 18:05:22
 * @LastEditTime: 2025-08-02 21:03:42
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/README-SC.md
-->

# repo-donkey

**[\[ Let's speak English \]](README.md)**

## 这是什么

帮您构建自己的Arch Linux Repo.

### 版本信息

当前版本: 0.0.3, 代号: UiharuKazari (初春饰利).

### What's new

1. 解除了对paru及proxychains的依赖.
2. 支持了自定义PKGBUILD.
3. 现在设置代理是通过环境变量而不是proxychains了.
4. 现在所有包都将是"build in a clean chroot"的.

## 用法

### 安全警示

恶意的PKGBUILD可能会造成安全问题, 请保证您可以确保相关PKGBUILD的安全性! 请务必多关注相关新闻或消息! 建议使用容器或虚拟机运行本程序以进一步保证安全性!

### 先决条件

安装有 bash, sudo, pacman, devtools, gpg 的 Arch Linux 实体机, VM, 或容器.

### 命令行参数

``` bash
repo-donkey path-to-config-file.conf
```

### 优雅退出

向程序传递一个SIGINT信号, 即可让程序开始优雅退出过程. 具体过程如下:

1. 收到SIGINT信号.
2. 停止Ticker.
3. 不再发起新的任务, 并等待现有任务结束.
4. 优雅退出.

### 暴力退出

传递SIGKILL信号直接杀死即可.

## 配置文件

``` ini
# 新版本, 暂无样例.
```
