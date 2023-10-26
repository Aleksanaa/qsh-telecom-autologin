# 电子科技大学清水河校区宿舍电信自动登录脚本

这下不需要Webview，可以在硬路由上跑了😋

请注意：与[UESTC-QingShuiHe-SRUN-TELECOM](https://github.com/coolmoon327/UESTC-QingShuiHe-SRUN-TELECOM)不同，这个脚本并不会在后台保持登陆状态。你需要写一个简单的wrapper来检查联网状态，并在适当的时候启动这个脚本。为什么不检查联网状态呢？因为

1. 我不想间断地ping登陆网页的地址，同时各个系统都应该有方便的检查联网状态的方式。
2. 掉登录后，仍要等到当前DHCP获取的IP过期后才能重新登录，并且刷新DHCP这个动作没有特权是没法完成的。

## 使用说明
1. 登录时，只需要在`-name`和`-passwd`参数后填写用户名和密码，如果电信某一天又更改了登录地址，在`-host`后面填写更改后的IP地址。
2. 如果需要存储账号和密码（密码由电信下发的RSA公钥加密存储），在`-cache`后填写文件的地址（包括文件名），例如`-cache ./cache.json`，在登录成功后会保存。下次仍需手动指定`-cache`读取。
3. 如果需要登出（应该不会需要），添加`-logout`参数，不需要指定账号和密码，但是需要指定`-index`（要抓包才知道），但是也可以直接用上次登录的cache。
4. 编译好的二进制文件可以在Actions里面下载，如果你需要其他架构可以提issue。

## 友情链接
如果你需要校园网登陆，可以使用[go-nd-portal](https://github.com/fumiama/go-nd-portal)。这个项目的作者[@fumiama](https://github.com/fumiama)是我最崇敬的网络工程师与Golang程序员。同时，你也可以使用由世界上最棒的语言®重写的版本[nd-portal](https://github.com/NetUnion/nd-portal)。