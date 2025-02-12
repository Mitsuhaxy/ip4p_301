# ip4p_301
解析IP4P地址并301重定向，仅支持https，配合 [NATMAP](https://github.com/heiher/natmap)/Lucky使用

所有代码均由 [DEEPSEEK-R1-671B](https://chat.deepseek.com/) 生成

## 原理：
在具有公网地址的服务器搭建，当用户访问`https://vpsip:port/uuid`时返回301重定向到`https://publicip:port`，其中publicip与其后port根据解析[IP4P格式地址](https://github.com/heiher/natmap/wiki/faq)得到

## 使用：
工具文件结构：
```
/opt/ip4p_301
    |-ip4p
    |-config.yaml
    |-server.crt
    |-server.key
```
config.yaml:
```
server:
  listen_port: 8443 #VPS上监听端口
  cert_file: "server.crt" #证书
  key_file: "server.key" #私钥

mappings:
  - uuid: "550e8400-e29b-41d4-a716-446655440000" #UUID，每个映射唯一，用于区分和保护未授权访问
    domain: "site1.example.com" #IP4P DDNS后的域名
  - uuid: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
    domain: "site2.example.com"
```

访问：
```
https://vpsip:port/uuid
```

## 访问效果：
取决于你的宽带上行与运营商实际效果
