package tcp_client

import (
	"context"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/zhhx/zhhx_event"
	"io"
	"net"
	"sync"
	"time"
)

const (
	MSG_TYPE_REQUEST           = 1
	MSG_TYPE_REQUEST_NO_REPLAY = 2
	MSG_TYPE_REPLAY            = 3
)

type OnRequestFunc func(request IRequest)
type OnConnectFunc func(connection IConnection)
type OnDisconnectFunc func(connection IConnection)
type OnConnIDChangeFunc func(old string, new string)

// Connection 链接
type Connection struct {
	//当前连接的socket TCP套接字
	Conn *net.TCPConn
	//当前连接的ID 也可以称作为SessionID，ID全局唯一
	ConnID string
	//告知该链接已经退出/停止的channel
	ctx    context.Context
	cancel context.CancelFunc
	//有缓冲管道，用于读、写两个goroutine之间的消息通信
	msgBuffChan chan []byte

	sync.RWMutex
	//链接属性
	property map[string]interface{}
	////保护当前property的锁
	propertyLock sync.Mutex
	//当前连接的关闭状态
	isClosed bool

	seq           MutexCounter[uint8]
	maxPacketSize uint32
	waiter        *zhhx_event.EventWaitter[uint8, IMessage]

	onRequest          OnRequestFunc
	onConnectFunc      OnConnectFunc
	onDisconnectFunc   OnDisconnectFunc
	onConnIDChangeFunc OnConnIDChangeFunc
}

// NewConnection 创建连接的方法
func NewConnection(conn *net.TCPConn, connID string, MaxMsgChanLen uint32, onRequest OnRequestFunc) *Connection {
	//初始化Conn属性
	c := &Connection{
		Conn:          conn,
		ConnID:        connID,
		isClosed:      false,
		msgBuffChan:   make(chan []byte, MaxMsgChanLen),
		property:      nil,
		waiter:        zhhx_event.NewEventWaitter[uint8, IMessage](),
		onRequest:     onRequest,
		maxPacketSize: 0,
	}
	return c
}

// StartWriter 写消息Goroutine， 用户将数据发送给客户端
func (c *Connection) StartWriter() {
	defer log.Warningln(c.RemoteAddr().String(), "[connection Writer exit!]")

	for {
		select {
		case data, ok := <-c.msgBuffChan:
			if ok {
				//有数据要写给客户端
				if _, err := c.Conn.Write(data); err != nil {
					log.Errorln("Send Buff Data error:, ", err, " Conn Writer exit")
					return
				}
			} else {
				log.Println("msgBuffChan is Closed")
				break
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// StartReader 读消息Goroutine，用于从客户端中读取数据
func (c *Connection) StartReader() {
	defer log.Warningln(c.RemoteAddr().String(), "[connection Reader exit!]")
	defer c.Stop()

	// 创建拆包解包的对象
	for {
		select {
		case <-c.ctx.Done():
			return
		default:

			//读取客户端的Msg head
			msg := &Message{}
			headData := make([]byte, int(msg.GetHeadLen()))
			if _, err := io.ReadFull(c.Conn, headData); err != nil {
				log.Errorln("read msg head error ", err)
				return
			}
			//fmt.Printf("read headData %+v\n", headData)

			//拆包，得到msgID 和 datalen 放在msg中
			err := msg.Unpack(headData)
			if err != nil {
				log.Errorln("unpack error ", err)
				return
			}

			//判断dataLen的长度是否超出我们允许的最大包长度
			if c.maxPacketSize > 0 && msg.DataLen > c.maxPacketSize {
				log.Errorln("too large msg data received")
				return
			}

			//根据 dataLen 读取 data，放在msg.Data中
			var data []byte
			if msg.GetDataLen() > 0 {
				data = make([]byte, msg.GetDataLen())
				if _, err := io.ReadFull(c.Conn, data); err != nil {
					log.Errorln("read msg data error ", err)
					return
				}
			}
			msg.SetData(data)
			c.msgHandle(msg)

		}
	}
}

// Start 启动连接，让当前连接开始工作
func (c *Connection) Start() {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	//1 开启用户从客户端读取数据流程的Goroutine
	go c.StartReader()
	//2 开启用于写回客户端数据流程的Goroutine
	go c.StartWriter()
	//按照用户传递进来的创建连接时需要处理的业务，执行钩子方法
	if c.onConnectFunc != nil {
		c.onConnectFunc(c)
	}

	select {
	case <-c.ctx.Done():
		c.finalizer()
		return
	}
}

// Stop 停止连接，结束当前连接状态M
func (c *Connection) Stop() {
	c.cancel()
}

// GetTCPConnection 从当前连接获取原始的socket TCPConn
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Conn
}

// GetConnID 获取当前连接ID
func (c *Connection) GetConnID() string {
	return c.ConnID
}

// RemoteAddr 获取远程客户端地址信息
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

func (c *Connection) write(msgID uint32, seq uint8, data []byte, channel chan []byte) error {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	c.RLock()
	defer c.RUnlock()
	if c.isClosed == true {
		return errors.New("connection closed when send")
	}

	//将data封包，并且发送
	msg := NewMsgPackage(msgID, seq, data)
	data, err := msg.Pack()
	if err != nil {
		log.Println("Pack error msg Type = ", msgID)
		return errors.New("Pack error msg ")
	}
	//写回客户端
	select {
	case channel <- data:
	default:
		c.Stop()
		return errors.New("写入过快,超过channle缓存个数,抛弃消息")
	}

	return nil
}

func (c *Connection) Send(data []byte, timeout time.Duration) IMessage {
	seq := c.seq.Add(1)
	resp, err := c.waiter.Wait(seq, timeout, func() bool {
		err := c.write(MSG_TYPE_REQUEST, seq, data, c.msgBuffChan)
		if err != nil {
			log.Warningf("客户端[%s]:%s", c.GetConnID(), err.Error())
			return false
		}
		return true
	})
	if err != nil {
		if err.Error() == "sync waitter tag exist" {
			log.Warningf("客户端[%s] 超过同步消息同时等待的最大数量", c.GetConnID())
		}
	}
	return resp
}

func (c *Connection) SendNoReplay(data []byte) {
	seq := c.seq.Add(1)
	c.write(MSG_TYPE_REQUEST_NO_REPLAY, seq, data, c.msgBuffChan)
}

func (c *Connection) Replay(seq uint8, data interface{}) error {
	buffer, _ := json.Marshal(data)
	return c.write(MSG_TYPE_REPLAY, seq, buffer, c.msgBuffChan)
}

// SetProperty 设置链接属性
func (c *Connection) SetProperty(key string, value interface{}) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	if c.property == nil {
		c.property = make(map[string]interface{})
	}

	c.property[key] = value
}

// GetProperty 获取链接属性
func (c *Connection) GetProperty(key string) (interface{}, error) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	if value, ok := c.property[key]; ok {
		return value, nil
	}

	return nil, errors.New("no property found")
}

// RemoveProperty 移除链接属性
func (c *Connection) RemoveProperty(key string) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	delete(c.property, key)
}

func (c *Connection) SetMaxPacketSize(maxPacketSize uint32) {
	c.maxPacketSize = maxPacketSize
}

func (c *Connection) SetConnID(connID string) {
	old := c.ConnID
	c.ConnID = connID
	if c.onConnIDChangeFunc != nil {
		c.onConnIDChangeFunc(old, connID)
	}
}

// 返回ctx，用于用户自定义的go程获取连接退出状态
func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) OnConnect(cb OnConnectFunc) {
	c.onConnectFunc = cb
}

func (c *Connection) OnDisConnect(cb OnDisconnectFunc) {
	c.onDisconnectFunc = cb
}

func (c *Connection) OnConnIDChange(onConnIDChangeFunc OnConnIDChangeFunc) {
	c.onConnIDChangeFunc = onConnIDChangeFunc
}

func (c *Connection) finalizer() {
	//如果用户注册了该链接的关闭回调业务，那么在此刻应该显示调用
	if c.onDisconnectFunc != nil {
		c.onDisconnectFunc(c)
	}
	c.Lock()
	defer c.Unlock()

	//如果当前链接已经关闭
	if c.isClosed == true {
		return
	}

	log.Warningln("Conn Stop()...ConnID = ", c.ConnID)

	// 关闭socket链接
	err := c.Conn.Close()
	if err != nil {
		log.Errorln("无关关闭链接:", err.Error())
	}

	//关闭该链接全部管道
	close(c.msgBuffChan)

	//设置标志位
	c.isClosed = true
}

func (c *Connection) msgHandle(msg IMessage) {
	if msg.GetMsgType() == MSG_TYPE_REPLAY {
		c.waiter.Report(msg.GetSeq(), msg)
		return
	} else if msg.GetMsgType() == MSG_TYPE_REQUEST || msg.GetMsgType() == MSG_TYPE_REQUEST_NO_REPLAY {
		request := Request{
			connection: c,
			IMessage:   msg,
		}
		c.onRequest(&request)
		//if utils.GlobalObject.WorkerPoolSize > 0 {
		//	//已经启动工作池机制，将消息交给Worker处理
		//	//c.MsgHandler.SendMsgToTaskQueue(&req)
		//} else {
		//	//从绑定好的消息和对应的处理方法中执行对应的Handle方法
		//	go c.onRequest(&request)
		//}
	}
}
