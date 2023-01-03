package tcp_server

import (
	log "github.com/sirupsen/logrus"
	"zhhx_transmit_server/zhhx_tcp_framework/connection"
)

// MsgHandle -
type MsgHandle struct {
	handle         connection.OnRequestFunc
	WorkerPoolSize uint32                     //业务工作Worker池的数量
	TaskQueue      []chan connection.IRequest //Worker负责取任务的消息队列
	seq            uint32
}

// NewMsgHandle 创建MsgHandle
func NewMsgHandle(handle connection.OnRequestFunc) *MsgHandle {
	return &MsgHandle{
		handle:         handle,
		WorkerPoolSize: GlobalObject.WorkerPoolSize,
		//一个worker对应一个queue
		TaskQueue: make([]chan connection.IRequest, GlobalObject.WorkerPoolSize),
	}
}

// SendMsgToTaskQueue 将消息交给TaskQueue,由worker进行处理
func (mh *MsgHandle) SendMsgToTaskQueue(request connection.IRequest) {
	//根据ConnID来分配当前的连接应该由哪个worker负责处理
	//轮询的平均分配法则
	seq := mh.seq
	mh.seq++
	//得到需要处理此条连接的workerID
	workerID := seq % mh.WorkerPoolSize
	log.Println("Add ConnID=", request.GetConnection().GetConnID(), " request msgID=", request.GetMsgType(), "to workerID=", workerID)
	//将请求消息发送给任务队列

	mh.TaskQueue[workerID] <- request
}

// DoMsgHandler 马上以非阻塞方式处理消息
func (mh *MsgHandle) DoMsgHandler(request connection.IRequest) {
	mh.handle(request)
}

// StartOneWorker 启动一个Worker工作流程
func (mh *MsgHandle) StartOneWorker(workerID int, taskQueue chan connection.IRequest) {
	log.Println("Worker Type = ", workerID, " is started.")
	//不断的等待队列中的消息
	for {
		select {
		//有消息则取出队列的Request，并执行绑定的业务方法
		case request := <-taskQueue:
			mh.DoMsgHandler(request)
		}
		if len(taskQueue) > 1000 {
			log.Warningf("消息队列[%d]过大,建议扩大工作线程个数:", len(taskQueue))
		}
	}
}

// StartWorkerPool 启动worker工作池
func (mh *MsgHandle) StartWorkerPool() {
	//遍历需要启动worker的数量，依此启动
	for i := 0; i < int(mh.WorkerPoolSize); i++ {
		//一个worker被启动
		//给当前worker对应的任务队列开辟空间
		mh.TaskQueue[i] = make(chan connection.IRequest, GlobalObject.MaxWorkerTaskLen)
		//启动当前Worker，阻塞的等待对应的任务队列是否有消息传递进来
		go mh.StartOneWorker(i, mh.TaskQueue[i])
	}
}
