package handlers

import (
	"encoding/json"
)

func (r *watchRoom) add(c *watchClient) {
	r.mu.Lock()
	r.clients[c] = true
	if c.peerID != "" {
		r.byPeer[c.peerID] = c
	}
	r.mu.Unlock()
}

func (r *watchRoom) remove(c *watchClient) {
	r.mu.Lock()
	delete(r.clients, c)
	if c.peerID != "" {
		delete(r.byPeer, c.peerID)
	}
	r.mu.Unlock()
}

// sendToPeer 向指定 peerID 发送定向消息（WebRTC 信令）
func (r *watchRoom) sendToPeer(peerID string, msg watchBroadcast) {
	data, _ := json.Marshal(msg)
	r.mu.RLock()
	c, ok := r.byPeer[peerID]
	if ok {
		select {
		case c.send <- data:
		default:
		}
	}
	r.mu.RUnlock()
}

// voicePeers 返回当前已加入语音的成员（排除 exclude）
func (r *watchRoom) voicePeers(exclude *watchClient) []voicePeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var peers []voicePeerInfo
	for c := range r.clients {
		if c != exclude && c.voiceActive {
			peers = append(peers, voicePeerInfo{PeerID: c.peerID, Nickname: c.nickname})
		}
	}
	return peers
}

func (r *watchRoom) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

func (r *watchRoom) broadcast(msg watchBroadcast, exclude *watchClient) {
	data, _ := json.Marshal(msg)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for c := range r.clients {
		if c == exclude {
			continue
		}
		select {
		case c.send <- data:
		default:
		}
	}
}

func (r *watchRoom) broadcastAll(msg watchBroadcast) {
	r.broadcast(msg, nil)
}

// WatchWS 处理一起看 WebSocket 连接
