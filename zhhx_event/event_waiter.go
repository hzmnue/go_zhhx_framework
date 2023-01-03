package zhhx_event

import (
	"errors"
	"sync"
	"time"
)

type EventWaitter[T uint8 | uint16 | uint64 | string, V any] struct {
	channMap map[T]chan V
	sync.RWMutex
}

func NewEventWaitter[T uint8 | uint16 | uint64 | string, V any]() *EventWaitter[T, V] {
	return &EventWaitter[T, V]{channMap: map[T]chan V{}}
}

func (this *EventWaitter[T, V]) Wait(tag T, duration time.Duration, handle func() bool) (V, error) {
	var zero V
	channel := make(chan V)
	if !this.addChannel(tag, channel) {
		return zero, errors.New("sync waitter tag exist")
	}
	defer this.removeChannel(tag)
	if !handle() {
		return zero, errors.New("sync waitter handle false")
	}
	timer := time.NewTicker(duration)
	defer timer.Stop()
	select {
	case result := <-channel:
		return result, nil
	case <-timer.C:
		return zero, errors.New("sync waitter timeout")
	}
	return zero, errors.New("sync waitter unknow error")
}

func (this *EventWaitter[T, V]) Report(tag T, value V) {
	this.RLock()
	defer this.RUnlock()
	channel := this.channMap[tag]
	if channel == nil {
		return
	}
	channel <- value
}

func (this *EventWaitter[T, V]) addChannel(tag T, channel chan V) bool {
	this.Lock()
	defer this.Unlock()
	if _, ok := this.channMap[tag]; ok {
		return false
	}
	this.channMap[tag] = channel
	return true
}

func (this *EventWaitter[T, V]) removeChannel(tag T) {
	this.Lock()
	defer this.Unlock()
	if channel, ok := this.channMap[tag]; ok {
		delete(this.channMap, tag)
		close(channel)
	}
}
