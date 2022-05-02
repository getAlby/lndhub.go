package service

import (
	"sync"

	"github.com/getAlby/lndhub.go/db/models"
)

type Pubsub struct {
	mu   sync.RWMutex
	subs map[int64]map[string]chan models.Invoice
}

func NewPubsub() *Pubsub {
	ps := &Pubsub{}
	ps.subs = make(map[int64]map[string]chan models.Invoice)
	return ps
}

func (ps *Pubsub) Subscribe(topic int64, ch chan models.Invoice) (subId string, err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		ps.subs[topic] = make(map[string]chan models.Invoice)
	}
	//re-use preimage code for a uuid
	preImageHex, err := makePreimageHex()
	if err != nil {
		return "", err
	}
	subId = string(preImageHex)
	ps.subs[topic][subId] = ch
	return subId, nil
}

func (ps *Pubsub) Unsubscribe(id string, topic int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		return
	}
	if ps.subs[topic][id] == nil {
		return
	}
	close(ps.subs[topic][id])
	delete(ps.subs[topic], id)
}

func (ps *Pubsub) Publish(topic int64, msg models.Invoice) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.subs[topic] == nil {
		return
	}

	for _, ch := range ps.subs[topic] {
		ch <- msg
	}
}
