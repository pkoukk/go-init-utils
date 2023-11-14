# go-init-utils
Competitive programming utility library for go
It is not a framework, just a set of utilities, so you can use it with any other framework.  
This library is inspired by [go-spring], but in go style rather than java style.

基本理念
对常见的库进行interface封装，让这些interface之间互相配套和适配
同时，提供一些对这些库进行简易初始化的方法
然后，给出使用不同的ioc框架进行初始化的方法，暂定uber的fx和google的wire
最底层的依赖interface: config, logger, metrics
然后数据相关的： gorm redis influxdb kafka
插入一些简单的工具件： http client
然后是server相关的： gin grpc