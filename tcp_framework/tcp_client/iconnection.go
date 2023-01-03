// Package ziface 主要提供zinx全部抽象层接口定义.
// 包括:
//
//			IServer 服务mod接口
//			IRouter 路由mod接口
//			IConnection 连接mod层接口
//	     IMessage 消息mod接口
//			IDataPack 消息拆解接口
//	     IMsgHandler 消息处理及协程池接口
//
// 当前文件描述:
// @Title  iconnection.go
// @Description  全部连接相关方法声明
// @Author  Aceld - Thu Mar 11 10:32:29 CST 2019
package tcp_client

import (
	"context"
	"net"
	"time"
)

// 定义连接接口
type IConnection interface {
	Start()                   //启动连接，让当前连接开始工作
	Stop()                    //停止连接，结束当前连接状态M
	Context() context.Context //返回ctx，用于用户自定义的go程获取连接退出状态

	GetTCPConnection() *net.TCPConn //从当前连接获取原始的socket TCPConn
	GetConnID() string              //获取当前连接ID
	SetConnID(connID string)        //设置连接ID
	RemoteAddr() net.Addr           //获取远程客户端地址信息

	Send(data []byte, timeout time.Duration) IMessage //发送同步消息
	SendNoReplay(data []byte)                         //发送异步消息

	Replay(seq uint8, data interface{}) error //回复对应的消息

	SetProperty(key string, value interface{})   //设置链接属性
	GetProperty(key string) (interface{}, error) //获取链接属性
	RemoveProperty(key string)                   //移除链接属性

	SetMaxPacketSize(maxPacketSize uint32) //设置链接属性

	OnConnect(cb OnConnectFunc)
	OnDisConnect(cb OnDisconnectFunc)
	OnConnIDChange(onConnIDChangeFunc OnConnIDChangeFunc)
}
