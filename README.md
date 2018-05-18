# X-Chat
命令行里的聊天工具, 使用Go语言， 基于TCP协议实现

## Usage
```shell
#服务器启动监听 9001端口
./X-Chat -l [port]

#客户端连接服务器, 支持多客户连接
./X-Chat [ip] [port]

# 文件传输
<< filename

#文件接收
>> filename
```
