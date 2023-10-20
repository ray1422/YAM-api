package signaling

import (
	"encoding/json"
	"log"
	"time"
)

type simpleData struct {
	toID string
	data []byte
}

type Addr struct {
	Network string
	Address string
}
type Member struct {
	Addr Addr   `json:"addr"`
	ID   string `json:"id"`
}

type HubInfo struct {
	ID      string
	Members []Member
}

// Hub hub
type Hub struct {
	ID             string
	clients        map[string]*Client
	RegisterChan   chan *Client // must be unbuffered chan to make sure send await register
	UnregisterChan chan *Client // must be unbuffered chan

	simpleChan chan *simpleData

	RequestInfoChan chan *chan *HubInfo

	cleanTicker time.Ticker
}

var (
	cleanerTimeout = 30 * time.Second
)

// CreateHub CreateHub
func CreateHub(roomID string) *Hub {
	globalHubsLock.Lock()
	defer globalHubsLock.Unlock()
	if hubs[roomID] != nil {
		return hubs[roomID]
	}
	h := &Hub{
		ID:              roomID,
		clients:         map[string]*Client{},
		RegisterChan:    make(chan *Client),
		UnregisterChan:  make(chan *Client),
		simpleChan:      make(chan *simpleData, 8192),
		RequestInfoChan: make(chan *chan *HubInfo, 512),
		cleanTicker:     *time.NewTicker(cleanerTimeout),
	}
	hubs[roomID] = h
	go h.Loop()
	return h
}

// Loop loop for hub. should be create in goroutine
func (h *Hub) Loop() {
	defer func() {
		log.Println("hub closed")
	}()
	for {
		select {
		case client := <-h.RegisterChan:
			h.cleanTicker.Reset(cleanerTimeout)
			clientsID := []string{}
			for i := range h.clients {
				clientsID = append(clientsID, i)
			}
			clientsIDBytes, err := json.Marshal(map[string]interface{}{"clients": clientsID, "self_client_id": client.id})
			if err != nil {
				log.Println("something went wrong", err)
				continue
			}
			action := actionWrapper{Action: "list_client", Data: json.RawMessage(clientsIDBytes)}
			clientListBytes, err := json.Marshal(action)
			if err != nil {
				log.Println("something went wrong")
				continue
			}
			client.send <- clientListBytes
			h.clients[client.id] = client
		case client := <-h.UnregisterChan:
			client.close()
			delete(h.clients, client.id)
			h.cleanTicker.Reset(5 * time.Second)
			for _, c := range h.clients {
				b, err := json.Marshal(map[string]interface{}{
					"action": "client_event",
					"data": map[string]string{
						"remote_id": client.id,
						"event":     "leave",
					},
				})
				if err == nil {
					c.send <- b
				} else {
					log.Println(err)
				}
			}
		case dat := <-h.simpleChan:
			if _, ok := h.clients[dat.toID]; ok {
				h.clients[dat.toID].send <- dat.data
			}

		case ch := <-h.RequestInfoChan:
			ms := []Member{}
			for idx, c := range h.clients {
				ms = append(ms, Member{
					ID: idx,
					Addr: Addr{
						Network: c.conn.RemoteAddr().Network(),
						Address: c.conn.RemoteAddr().String(),
					},
				})
			}
			*ch <- &HubInfo{
				ID:      h.ID,
				Members: ms,
			}
		case <-h.cleanTicker.C:
			if len(h.clients) > 0 {
				h.cleanTicker.Reset(cleanerTimeout)
				continue
			}
			log.Println("hub " + h.ID + " close due to no clients in room")
			for _, c := range h.clients {
				c.hubClosed <- true
			}
			select {
			case ch := <-h.RequestInfoChan:
				*ch <- nil
			default:
			}
			globalHubsLock.Lock()
			delete(hubs, h.ID)
			globalHubsLock.Unlock()
			return
		}

	}
}
