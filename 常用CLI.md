服务器初始化

#批量创建服务器中间件到分组
jpy middleware auth create

#选择活动分组服务器
jpy middleware auth select default

#移除连接失败的服务器（软删除）
jpy middleware remove

#尝试登录失败的服务器
jpy middleware relogin

#列出登录失败的服务器
jpy middleware list --has-fail

#清空服务器地址（永久）
jpy middleware remove --force --all
-------------------
服务器列表筛选

#获取未授权服务器
jpy middleware device status --auth-failed

#获取未全部上线服务器
jpy middleware device status --biz-online-lt 20
------------
服务器控制

#全自动授权服务器
jpy middleware admin auto-auth

#全自动设置集控平台地址
jpy middleware admin update-cluster

#查看链接中间件shell到root密码
jpy middleware ssh 192.168.10.201
------------
设备列表筛选

#获取已开启USB设备
jpy middleware device list --filter-adb true

#获取没有IP的设备
jpy middleware device list --filter-has-ip false

#获取没有SN的设备
jpy middleware device list --filter-uuid false
-------------
设备控制

#将无SN的设备切换到USB模式
jpy middleware device usb --mode usb --filter-uuid false

#将无SN的设备切换到OTG模式
jpy middleware device usb --mode host --filter-uuid false

#重启所有无sn到设备
jpy middleware device reboot --filter-uuid false

#将无IP的设备切换到USB模式
jpy middleware device usb --mode usb --filter-has-ip false

#将无IP的设备切换到OTG模式
jpy middleware device usb --mode host --filter-has-ip false

#重启所有无IP的设备
jpy middleware device reboot --filter-has-ip false

#查看指定设备guard日志
jpy middleware device log --server "192.168.30.203" --seat 1

