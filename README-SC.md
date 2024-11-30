<!--
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2024-11-18 18:05:22
 * @LastEditTime: 2024-11-30 21:56:21
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/README-SC.md
-->

# repo-donkey

**[\[ Let's speak English \]](README.md)**

## 这是什么

帮您基于AUR构建自己的Arch Linux Repo.

### 版本信息

当前版本: 0.0.2, 代号: UiharuKazari (初春饰利).

## 用法

### 安全警示

恶意的PKGBUILD可能会造成安全问题, 请保证您可以确保相关AUR上包的安全性! 请务必多关注相关新闻或消息! 建议使用容器或虚拟机运行本程序以进一步保证安全性!

### 先决条件

安装有paru, sudo, repo-add, gpg, bash, 以及proxychains (除paru, 均存在于Arch Linux的源中, paru可自AUR安装).

### 命令行参数

``` bash
repo-donkey -c test.conf
```

命令行参数说明:

``` text
-c --config: 指定要使用的配置文件. 这是必需的.
-l --logdir: 指定存放日志的目录. 这是必需的.
--no-color: 禁用颜色输出. 这可能对某些东西如Paru无效.
-s --sikp-init-build: 跳过启动后进行的一次非并行构建.
--debug: 调试模式.
-h --help: 输出帮助信息.
```

**注意: 任何无效参数都会被直接忽略且无任何警告!**

### 优雅退出

向程序传递一个SIGINT信号, 即可让程序开始优雅退出过程. 具体过程如下:

1. 收到SIGINT信号.
2. 不再发起新的任务, 并等待现有任务结束.
3. 清理生成的临时文件.
4. 优雅退出.

### 暴力退出

传递SIGKILL信号直接杀死即可. 注意, 一些临时文件将可能将会遗留下来!

## 配置文件

``` ini
# 这是注释, 这样的单行注释是被允许的.

# 方括号内指定包名 (必需)
[imagej]

# Schedule string (必需)
# 使用标准的或加入秒支持的cron表达式, 指定进行更新的频率.
# @every xxx, @daily等也被支持. 基于robfig/cron.v3构建.
# 详见: https://pkg.go.dev/gopkg.in/robfig/cron.v3.
# 若一次任务没有完成, 则下次任务不会开始, 及时时间已到.
Schedule = @daily

# CloneDir string (必需)
# 即Paru的--clonedir对应的值, 用于下载和运行PKGBUILD的目录.
CloneDir = /home/donkey/

# TargetDB string (必需)
# 指定对应的DB.
TargetDB = /srv/http/myrepo/myrepo.db.tar.gz

# User string (必需)
# 指定运行paru时使用的用户.
User = donkey

# Group string (必需)
# 指定运行paru时使用的组.
Group = donkey

# UseChroot bool (非必需)
# 指定是否在构建软件包时使用chroot. 默认为false.
UseChroot = true

# Sign bool (非必需)
# 指定是否在构建软件包后对软件包进行签名,
# 同时也是指明了在更新数据库前是否校验数据库签名,
# 以及是否在更新数据库后对数据库进行签名.
# 签名时使用指定用户的默认GPG密钥.
Sign = true
```
