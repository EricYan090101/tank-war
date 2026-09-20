package game

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"
)

type VersusInput struct {
	Name         string
	TeamChoice   int // 0 means no request, 1 bottom, 2 top.
	Dir          Direction
	Moving, Fire bool
}
type versusSnapshot struct {
	Coop       *coopSnapshot
	State      GameState
	Match      VersusMatch
	Connected  []int
	Teams      [4]int
	Tanks      []Tank
	Bullets    []Bullet
	Explosions []Explosion
	Tiles      [MapRows][MapCols]int
	Bricks     [MapRows][MapCols]uint8
	Frame      int
}
type versusWire struct {
	Kind     string
	ID       int
	Room     RoomInfo
	Input    VersusInput
	Snapshot *versusSnapshot
	Error    string
}
type versusPeer struct {
	conn net.Conn
	send chan versusWire
	done chan struct{}
	once sync.Once
}

func newVersusPeer(conn net.Conn) *versusPeer {
	return &versusPeer{conn: conn, send: make(chan versusWire, 1), done: make(chan struct{})}
}
func (p *versusPeer) close() { p.once.Do(func() { close(p.done); _ = p.conn.Close() }) }
func (p *versusPeer) queue(msg versusWire) {
	select {
	case <-p.done:
		return
	default:
	}
	select {
	case p.send <- msg:
	default:
		select {
		case <-p.send:
		default:
		}
		select {
		case p.send <- msg:
		default:
		}
	}
}
func (p *versusPeer) writeLoop() {
	defer p.close()
	encoder := json.NewEncoder(p.conn)
	for {
		select {
		case <-p.done:
			return
		case msg := <-p.send:
			_ = p.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if encoder.Encode(msg) != nil {
				return
			}
		}
	}
}

// Socket goroutines only mutate this locked transport. Game state stays on Update.
type VersusSession struct {
	Host      bool
	LocalID   int
	mu        sync.Mutex
	room      RoomInfo
	peers     map[int]*versusPeer
	names     [4]string
	localName string
	teams     map[int]int
	inputs    map[int]VersusInput
	inputAt   map[int]time.Time
	playing   bool
	closed    bool
	latest    *versusSnapshot
	failure   string
	listener  net.Listener
	client    *versusPeer
}

func HostVersus(address string, room RoomInfo, names ...string) (*VersusSession, error) {
	return hostSession(address, room, false, names...)
}
func HostCoop(address string, room RoomInfo) (*VersusSession, error) {
	room.TeamSize = 2
	room.BestOf = 3
	return hostSession(address, room, true)
}
func hostSession(address string, room RoomInfo, coop bool, names ...string) (*VersusSession, error) {
	config := VersusConfig{room.TeamSize, room.BestOf}
	if !config.Valid() {
		return nil, errors.New("invalid match settings")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	room.Mode = int(ModeVersus)
	if coop {
		room.Mode = int(ModeCoop)
	}
	room.MaxPlayers = config.Players()
	room.CurrentPlayers = 1
	s := &VersusSession{Host: true, room: room, listener: listener, peers: map[int]*versusPeer{}, inputs: map[int]VersusInput{}, inputAt: map[int]time.Time{}}
	if len(names) > 0 {
		s.names[0] = cleanPlayerName(names[0])
	}
	s.teams = map[int]int{0: 0}
	if room.TeamSize == 2 {
		s.teams[0] = -1
	}
	go s.acceptLoop()
	return s, nil
}
func (s *VersusSession) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		id := -1
		if !s.closed && !s.playing {
			for i := 1; i < s.room.MaxPlayers; i++ {
				if s.peers[i] == nil {
					id = i
					break
				}
			}
		}
		if id < 0 {
			s.mu.Unlock()
			_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
			_ = json.NewEncoder(conn).Encode(versusWire{Kind: "error", Error: "ROOM FULL OR MATCH STARTED"})
			_ = conn.Close()
			continue
		}
		p := newVersusPeer(conn)
		s.peers[id] = p
		s.teams[id] = id % 2
		if s.room.TeamSize == 2 {
			s.teams[id] = -1
		}
		room := s.room
		s.mu.Unlock()
		go s.servePeer(id, p, room)
	}
}
func wireScanner(conn net.Conn) *bufio.Scanner {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	return scanner
}
func (s *VersusSession) servePeer(id int, p *versusPeer, room RoomInfo) {
	defer func() {
		p.close()
		s.mu.Lock()
		delete(s.peers, id)
		delete(s.teams, id)
		s.names[id] = ""
		delete(s.inputs, id)
		delete(s.inputAt, id)
		s.mu.Unlock()
	}()
	_ = p.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if json.NewEncoder(p.conn).Encode(versusWire{Kind: "welcome", ID: id, Room: room}) != nil {
		return
	}
	go p.writeLoop()
	scanner := wireScanner(p.conn)
	for {
		_ = p.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if !scanner.Scan() {
			return
		}
		var msg versusWire
		if json.Unmarshal(scanner.Bytes(), &msg) != nil || msg.Kind != "input" || msg.Input.Dir < DirUp || msg.Input.Dir > DirRight {
			return
		}
		s.mu.Lock()
		if (!s.playing || s.names[id] == "") && msg.Input.Name != "" {
			s.names[id] = cleanPlayerName(msg.Input.Name)
		}
		if msg.Input.TeamChoice > 0 {
			s.selectTeamLocked(id, msg.Input.TeamChoice-1)
		}
		s.inputs[id] = msg.Input
		s.inputAt[id] = time.Now()
		s.mu.Unlock()
	}
}
func JoinVersus(address string, names ...string) (*VersusSession, error) {
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return nil, err
	}
	scanner := wireScanner(conn)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if !scanner.Scan() {
		_ = conn.Close()
		return nil, errors.New("room did not respond")
	}
	var msg versusWire
	if json.Unmarshal(scanner.Bytes(), &msg) != nil || msg.Kind != "welcome" || !(VersusConfig{msg.Room.TeamSize, msg.Room.BestOf}).Valid() {
		_ = conn.Close()
		return nil, errors.New("room full, started, or incompatible")
	}
	p := newVersusPeer(conn)
	s := &VersusSession{LocalID: msg.ID, room: msg.Room, client: p}
	if len(names) > 0 {
		s.localName = cleanPlayerName(names[0])
	}
	go p.writeLoop()
	go func() {
		defer func() {
			p.close()
			s.mu.Lock()
			if !s.closed {
				s.failure = "CONNECTION TO HOST LOST"
			}
			s.mu.Unlock()
		}()
		for {
			_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			if !scanner.Scan() {
				return
			}
			var msg versusWire
			if json.Unmarshal(scanner.Bytes(), &msg) != nil || msg.Kind != "state" || msg.Snapshot == nil {
				return
			}
			s.mu.Lock()
			s.latest = msg.Snapshot
			s.mu.Unlock()
		}
	}()
	return s, nil
}
func (s *VersusSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.client != nil {
		s.client.close()
	}
	for _, p := range s.peers {
		p.close()
	}
}
func (s *VersusSession) Info() RoomInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	info := s.room
	if s.Host {
		info.CurrentPlayers = 1 + len(s.peers)
	}
	info.InProgress = s.playing
	return info
}
func (s *VersusSession) Connected() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []int{0}
	for i := 1; i < s.room.MaxPlayers; i++ {
		if s.peers[i] != nil {
			out = append(out, i)
		}
	}
	return out
}
func (s *VersusSession) Begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.Host || s.playing {
		return false
	}
	if s.room.Mode == int(ModeCoop) {
		s.playing = true
		return true
	}
	if len(s.peers)+1 != s.room.MaxPlayers {
		return false
	}
	counts := [2]int{}
	for _, team := range s.teams {
		if team >= 0 && team < 2 {
			counts[team]++
		}
	}
	if counts != [2]int{s.room.TeamSize, s.room.TeamSize} {
		return false
	}
	s.playing = true
	return true
}
func (s *VersusSession) UnlockLobby() { s.mu.Lock(); s.playing = false; s.mu.Unlock() }
func (s *VersusSession) Inputs() map[int]VersusInput {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[int]VersusInput{}
	for id, input := range s.inputs {
		if time.Since(s.inputAt[id]) < 500*time.Millisecond {
			out[id] = input
		}
	}
	return out
}
func (s *VersusSession) SendInput(input VersusInput) {
	if s.client != nil {
		input.Name = s.localName
		s.client.queue(versusWire{Kind: "input", Input: input})
	}
}
func (s *VersusSession) Publish(snapshot *versusSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.peers {
		p.queue(versusWire{Kind: "state", Snapshot: snapshot})
	}
}
func (s *VersusSession) Receive() (*versusSnapshot, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.latest
	s.latest = nil
	return snap, s.failure
}

func (s *VersusSession) selectTeamLocked(id, team int) bool {
	if s.playing || s.closed || team < 0 || team > 1 {
		return false
	}
	if _, ok := s.teams[id]; !ok {
		return false
	}
	count := 0
	for other, t := range s.teams {
		if other != id && t == team {
			count++
		}
	}
	if count >= s.room.TeamSize {
		return false
	}
	s.teams[id] = team
	return true
}
func (s *VersusSession) SelectTeam(id, team int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selectTeamLocked(id, team)
}
func (s *VersusSession) Teams() [4]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := [4]int{-1, -1, -1, -1}
	for id, team := range s.teams {
		out[id] = team
	}
	return out
}

func cleanPlayerName(name string) string {
	out := ""
	for _, r := range name {
		if len(out) >= 12 {
			break
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out += string(r)
		}
	}
	return out
}
func (s *VersusSession) Names() [4]string { s.mu.Lock(); defer s.mu.Unlock(); return s.names }
