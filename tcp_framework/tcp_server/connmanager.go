package tcp_server

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/zhhx/tcp_framework/tcp_client"
	"sync"
)

// ConnManager 连接管理模块
type ConnManager struct {
	connections map[string]tcp_client.IConnection
	connLock    sync.RWMutex
}

// NewConnManager 创建一个链接管理
func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: make(map[string]tcp_client.IConnection),
	}
}

// Add 添加链接
func (connMgr *ConnManager) Add(conn tcp_client.IConnection) {

	connMgr.connLock.Lock()
	//将conn连接添加到ConnMananger中
	connMgr.connections[conn.GetConnID()] = conn
	conn.OnConnIDChange(func(old string, new string) {
		connMgr.connLock.Lock()
		delete(connMgr.connections, old)
		connMgr.connections[conn.GetConnID()] = conn
		connMgr.connLock.Unlock()
	})
	connMgr.connLock.Unlock()

	log.Println("[当前连接个数]:", connMgr.Len())
}

// Remove 删除连接
func (connMgr *ConnManager) Remove(conn tcp_client.IConnection) {

	connMgr.connLock.Lock()
	if nowConn, ok := connMgr.connections[conn.GetConnID()]; ok && nowConn == conn {
		delete(connMgr.connections, conn.GetConnID())
	}
	//删除连接信息
	connMgr.connLock.Unlock()
	log.Println("[当前连接个数]:", connMgr.Len())
}

// Get 利用ConnID获取链接
func (connMgr *ConnManager) Get(connID string) (tcp_client.IConnection, error) {
	connMgr.connLock.RLock()
	defer connMgr.connLock.RUnlock()

	if conn, ok := connMgr.connections[connID]; ok {
		return conn, nil
	}

	return nil, errors.New("connection not found")

}

// Len 获取当前连接
func (connMgr *ConnManager) Len() int {
	connMgr.connLock.RLock()
	length := len(connMgr.connections)
	connMgr.connLock.RUnlock()
	return length
}

// ClearConn 清除并停止所有连接
func (connMgr *ConnManager) ClearConn() {
	connMgr.connLock.Lock()

	//停止并删除全部的连接信息
	for connID, conn := range connMgr.connections {
		//停止
		conn.Stop()
		//删除
		delete(connMgr.connections, connID)
	}
	connMgr.connLock.Unlock()
	log.Println("Clear All Connections successfully: connection num = ", connMgr.Len())
}

// ClearOneConn  利用ConnID获取一个链接 并且删除
func (connMgr *ConnManager) ClearOneConn(connID string) {
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	connections := connMgr.connections
	if conn, ok := connections[connID]; ok {
		//停止
		conn.Stop()
		//删除
		delete(connections, connID)
		log.Println("Clear Connections Type:  ", connID, "succeed")
		return
	}

	log.Println("Clear Connections Type:  ", connID, "err")
	return
}
