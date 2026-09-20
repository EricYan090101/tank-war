package game

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	UDPBroadcastPort = ":9988"
	TCPHostPort      = ":9989"
)

type RoomInfo struct {
	TeamSize       int    `json:"team_size"`
	BestOf         int    `json:"best_of"`
	InProgress     bool   `json:"in_progress"`
	RoomName       string `json:"room_name"`
	Mode           int    `json:"mode"`
	CurrentPlayers int    `json:"current_players"`
	MaxPlayers     int    `json:"max_players"`
	HostIP         string `json:"host_ip"`
}

type NetworkManager struct {
	IsHost           bool
	RoomInfo         RoomInfo
	DiscoveredRooms  map[string]RoomInfo
	roomsLock        sync.RWMutex
	stopBroadcast    chan struct{}
	tcpListener      net.Listener
	clients          []net.Conn
	connLock         sync.Mutex
	broadcastStopped bool
	listening        bool
	discoveryMu      sync.Mutex
	seenAt           map[string]time.Time
}

func NewNetworkManager() *NetworkManager {
	return &NetworkManager{DiscoveredRooms: make(map[string]RoomInfo)}
}

func (nm *NetworkManager) StartBroadcasting() {
	nm.StopBroadcasting()
	nm.stopBroadcast = make(chan struct{})
	nm.broadcastStopped = false
	stop := nm.stopBroadcast
	go func() {
		addr, err := net.ResolveUDPAddr("udp", "255.255.255.255"+UDPBroadcastPort)
		if err != nil {
			return
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return
		}
		defer conn.Close()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				nm.roomsLock.RLock()
				data, _ := json.Marshal(nm.RoomInfo)
				nm.roomsLock.RUnlock()
				_, _ = conn.Write(data)
			}
		}
	}()
}

func (nm *NetworkManager) StopBroadcasting() {
	if nm.stopBroadcast != nil && !nm.broadcastStopped {
		nm.broadcastStopped = true
		close(nm.stopBroadcast)
	}
}

func (nm *NetworkManager) StartTCPServer(onJoin, onLeave func()) {
	if nm.tcpListener != nil {
		_ = nm.tcpListener.Close()
	}
	listener, err := net.Listen("tcp", TCPHostPort)
	if err != nil {
		fmt.Println("TCP:", err)
		return
	}
	nm.tcpListener = listener
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			nm.connLock.Lock()
			if len(nm.clients)+1 >= nm.RoomInfo.MaxPlayers {
				_, _ = conn.Write([]byte("ROOM_FULL\n"))
				_ = conn.Close()
				nm.connLock.Unlock()
				continue
			}
			nm.clients = append(nm.clients, conn)
			nm.RoomInfo.CurrentPlayers = len(nm.clients) + 1
			nm.connLock.Unlock()
			if onJoin != nil {
				onJoin()
			}
			go nm.handleClient(conn, onLeave)
		}
	}()
}

func (nm *NetworkManager) handleClient(conn net.Conn, onLeave func()) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		if string(buf[:n]) == "LEAVE_ROOM\n" {
			break
		}
	}
	_ = conn.Close()
	nm.connLock.Lock()
	for i, c := range nm.clients {
		if c == conn {
			nm.clients = append(nm.clients[:i], nm.clients[i+1:]...)
			break
		}
	}
	nm.RoomInfo.CurrentPlayers = len(nm.clients) + 1
	nm.connLock.Unlock()
	if onLeave != nil {
		onLeave()
	}
}

func (nm *NetworkManager) DestroyRoom() {
	nm.StopBroadcasting()
	nm.connLock.Lock()
	for _, c := range nm.clients {
		_, _ = c.Write([]byte("ROOM_DESTROYED\n"))
		_ = c.Close()
	}
	nm.clients = nil
	nm.connLock.Unlock()
	if nm.tcpListener != nil {
		_ = nm.tcpListener.Close()
		nm.tcpListener = nil
	}
}

func (nm *NetworkManager) StartListeningRooms() {
	nm.discoveryMu.Lock()
	if nm.listening {
		nm.discoveryMu.Unlock()
		return
	}
	nm.listening = true
	nm.discoveryMu.Unlock()
	go func() {
		addr, err := net.ResolveUDPAddr("udp", UDPBroadcastPort)
		if err != nil {
			nm.discoveryMu.Lock()
			nm.listening = false
			nm.discoveryMu.Unlock()
			return
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			nm.discoveryMu.Lock()
			nm.listening = false
			nm.discoveryMu.Unlock()
			return
		}
		defer func() { conn.Close(); nm.discoveryMu.Lock(); nm.listening = false; nm.discoveryMu.Unlock() }()
		_ = conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
		buf := make([]byte, 2048)
		for {
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() {
					_ = conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
					continue
				}
				return
			}
			var info RoomInfo
			if json.Unmarshal(buf[:n], &info) == nil {
				info.HostIP = src.IP.String()
				nm.roomsLock.Lock()
				key := src.IP.String() + ":" + info.RoomName
				nm.DiscoveredRooms[key] = info
				if nm.seenAt == nil {
					nm.seenAt = map[string]time.Time{}
				}
				nm.seenAt[key] = time.Now()
				nm.roomsLock.Unlock()
			}
			_ = conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
		}
	}()
}

func (nm *NetworkManager) ClearDiscoveredRooms() {
	nm.roomsLock.Lock()
	nm.DiscoveredRooms = make(map[string]RoomInfo)
	nm.roomsLock.Unlock()
}

func (nm *NetworkManager) GetDiscoveredRooms() []RoomInfo {
	nm.roomsLock.RLock()
	defer nm.roomsLock.RUnlock()
	out := make([]RoomInfo, 0, len(nm.DiscoveredRooms))
	for key, r := range nm.DiscoveredRooms {
		if at, ok := nm.seenAt[key]; ok && time.Since(at) > 4*time.Second {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RoomName < out[j].RoomName })
	return out
}

func (nm *NetworkManager) SetRoomInfo(info RoomInfo) {
	nm.roomsLock.Lock()
	nm.RoomInfo = info
	nm.roomsLock.Unlock()
}
