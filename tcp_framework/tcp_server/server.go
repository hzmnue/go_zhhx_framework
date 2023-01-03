package tcp_server

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"zhhx_transmit_server/zhhx_tcp_framework/connection"
)

// Server 接口实现，定义一个Server服务类
type Server struct {
	//服务器的名称
	Name string
	//tcp4 or other
	IPVersion string
	//服务绑定的IP地址
	IP string
	//服务绑定的端口
	Port int
	//当前Server的消息管理模块，用来绑定MsgID和对应的处理方法
	msgHandler IMsgHandle
	//当前Server的链接管理器
	ConnMgr IConnManager
	//该Server的连接创建时Hook函数
	OnConnStart func(conn connection.IConnection)
	//该Server的连接断开时的Hook函数
	OnConnStop func(conn connection.IConnection)
}

// NewServer 创建一个服务器句柄
func NewServer(onRequest connection.OnRequestFunc, opts ...Option) IServer {
	s := &Server{
		Name:       GlobalObject.Name,
		IPVersion:  "tcp4",
		IP:         GlobalObject.Host,
		Port:       GlobalObject.TCPPort,
		msgHandler: NewMsgHandle(onRequest),
		ConnMgr:    NewConnManager(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

//============== 实现 ziface.IServer 里的全部接口方法 ========

// Start 开启网络服务
func (this *Server) Start() {
	log.Printf("[START] Server name: %s,listenner at IP: %s, Port %d is starting\n", this.Name, this.IP, this.Port)

	//开启一个go去做服务端Linster业务
	go func() {
		//0 启动worker工作池机制
		this.msgHandler.StartWorkerPool()

		//1 获取一个TCP的Addr
		addr, err := net.ResolveTCPAddr(this.IPVersion, fmt.Sprintf("%s:%d", this.IP, this.Port))
		if err != nil {
			log.Println("resolve tcp addr err: ", err)
			return
		}

		//2 监听服务器地址
		listener, err := net.ListenTCP(this.IPVersion, addr)
		if err != nil {
			panic(err)
		}

		//已经监听成功
		log.Println("start Zinx server  ", this.Name, " succ, now listenning...")

		//TODO main.go 应该有一个自动生成ID的方法
		var cID uint32
		cID = 0

		//3 启动server网络连接业务
		for {
			//3.1 阻塞等待客户端建立连接请求
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Println("Accept err ", err)
				continue
			}
			log.Println("Get connection remote addr = ", conn.RemoteAddr().String())

			//3.2 设置服务器最大连接控制,如果超过最大连接，那么则关闭此新的连接
			if this.ConnMgr.Len() >= GlobalObject.MaxConn {
				conn.Close()
				continue
			}

			//3.3 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
			dealConn := connection.NewConnection(conn, fmt.Sprintf("tranmit-%d", cID), GlobalObject.MaxMsgChanLen, func(request connection.IRequest) {
				this.msgHandler.SendMsgToTaskQueue(request)
			})
			dealConn.SetMaxPacketSize(GlobalObject.MaxPacketSize)
			dealConn.OnConnect(func(connection connection.IConnection) {
				this.ConnMgr.Add(connection)
				this.callOnConnStart(connection)
			})
			dealConn.OnDisConnect(func(connection connection.IConnection) {
				this.ConnMgr.Remove(connection)
				this.callOnConnStop(connection)
			})

			cID++

			//3.4 启动当前链接的处理业务
			go dealConn.Start()
		}
	}()
}

// Stop 停止服务
func (this *Server) Stop() {
	log.Println("[STOP] Zinx server , name ", this.Name)

	//将其他需要清理的连接信息或者其他信息 也要一并停止或者清理
	this.ConnMgr.ClearConn()
}

// Serve 运行服务
func (this *Server) Serve() {
	this.Start()

	//TODO Server.Serve() 是否在启动服务的时候 还要处理其他的事情呢 可以在这里添加

	//阻塞,否则主Go退出， listenner的go将会退出
	select {}
}

// GetConnMgr 得到链接管理
func (this *Server) GetConnMgr() IConnManager {
	return this.ConnMgr
}

// SetOnConnStart 设置该Server的连接创建时Hook函数
func (this *Server) SetOnConnStart(hookFunc func(connection.IConnection)) {
	this.OnConnStart = hookFunc
}

// SetOnConnStop 设置该Server的连接断开时的Hook函数
func (this *Server) SetOnConnStop(hookFunc func(connection.IConnection)) {
	this.OnConnStop = hookFunc
}

// CallOnConnStart 调用连接OnConnStart Hook函数
func (this *Server) callOnConnStart(conn connection.IConnection) {
	if this.OnConnStart != nil {
		this.OnConnStart(conn)
	}
}

// CallOnConnStop 调用连接OnConnStop Hook函数
func (this *Server) callOnConnStop(conn connection.IConnection) {
	if this.OnConnStop != nil {
		this.OnConnStop(conn)
	}
}

func init() {
}
