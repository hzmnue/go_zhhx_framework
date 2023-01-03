package tcp_client

// Request 请求
type Request struct {
	connection IConnection //已经和客户端建立好的 链接
	IMessage               //客户端请求的数据
}

func (this *Request) Response(data interface{}) {
	this.connection.Replay(this.GetSeq(), data)
}

func (this *Request) GetConnection() IConnection {
	return this.connection
}
