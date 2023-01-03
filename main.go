package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/zhhx/tcp_framework/tcp_client"
	"github.com/zhhx/zhhx_event"
)

func main() {
	d := zhhx_event.EventWaitter[uint8, uint8]{}
	log.Errorln(d)
	var dd tcp_client.IConnection
	log.Warningln(dd)
}
