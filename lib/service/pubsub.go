package service

import (
	"sync"

	"github.com/getAlby/lndhub.go/db/models"
)

const DefaultChannelBufSize = 20

type Pubsub struct {
	mu   sync.RWMutex
	subs map[string]map[string]chan models.Invoice
}

func NewPubsub() *Pubsub {
	ps := &Pubsub{}
	ps.subs = make(map[string]map[string]chan models.Invoice)
	return ps
}

func (ps *Pubsub) Subscribe(topic string) (chan models.Invoice, string, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		ps.subs[topic] = make(map[string]chan models.Invoice)
	}
	//re-use preimage code for a uuid
	preImageHex, err := makePreimageHex()
	if err != nil {
		return nil, "", err
	}
	subId := string(preImageHex)

	ch := make(chan models.Invoice, DefaultChannelBufSize)
	ps.subs[topic][subId] = ch
	return ch, subId, nil
}

func (ps *Pubsub) Unsubscribe(id string, topic string) {
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

func (ps *Pubsub) Publish(topic string, msg models.Invoice) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.subs[topic] == nil {
		return
	}

	for _, ch := range ps.subs[topic] {
		ch <- msg
	}
}
