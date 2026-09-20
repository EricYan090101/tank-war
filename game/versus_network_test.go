package game

import (
	"testing"
	"time"
)

func waitVersus(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for network state")
}
func TestVersusNetworkLobbyInputAndSnapshots(t *testing.T) {
	for _, size := range []int{1, 2} {
		host, err := HostVersus("127.0.0.1:0", RoomInfo{RoomName: "Test", TeamSize: size, BestOf: 5})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(host.Close)
		if host.Begin() {
			t.Fatal("host started with missing players")
		}
		host.SelectTeam(0, 0)
		var clients []*VersusSession
		for id := 1; id < size*2; id++ {
			client, err := JoinVersus(host.listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(client.Close)
			clients = append(clients, client)
			if client.LocalID != id {
				t.Fatalf("wrong slot: got %d want %d", client.LocalID, id)
			}
			client.SendInput(VersusInput{Dir: Direction(id % 4), Moving: true, Fire: id%2 == 1, TeamChoice: id%2 + 1})
		}
		waitVersus(t, func() bool { return len(host.Inputs()) == size*2-1 })
		for id, input := range host.Inputs() {
			if input.Dir != Direction(id%4) || input.Fire != (id%2 == 1) {
				t.Fatal("player inputs mixed up")
			}
		}
		if !host.Begin() || !host.Info().InProgress {
			t.Fatal("full room could not start")
		}
		if extra, err := JoinVersus(host.listener.Addr().String()); err == nil {
			extra.Close()
			t.Fatal("started/full room admitted extra player")
		}
		game := versusTestGame(size, 5)
		game.LobbyMembers = host.Connected()
		snapshot := game.versusSnapshot()
		host.Publish(snapshot)
		for _, client := range clients {
			var received *versusSnapshot
			waitVersus(t, func() bool { received, _ = client.Receive(); return received != nil })
			if received.State != snapshot.State || len(received.Tanks) != size*2 || received.Tiles != snapshot.Tiles {
				t.Fatal("snapshot did not cross TCP intact")
			}
		}
		clients[len(clients)-1].Close()
		waitVersus(t, func() bool { return len(host.Connected()) == size*2-1 })
		host.UnlockLobby()
		if host.Begin() {
			t.Fatal("match restarted while missing player")
		}
		replacement, err := JoinVersus(host.listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(replacement.Close)
		if replacement.LocalID != size*2-1 {
			t.Fatal("departed slot was not reusable")
		}
		host.Close()
		waitVersus(t, func() bool { _, failure := replacement.Receive(); return failure != "" })
	}
}
func TestVersusInvalidConfig(t *testing.T) {
	for _, config := range []VersusConfig{{0, 3}, {3, 3}, {1, 4}, {2, 9}} {
		session, err := HostVersus("127.0.0.1:0", RoomInfo{TeamSize: config.TeamSize, BestOf: config.BestOf})
		if err == nil {
			session.Close()
			t.Fatal("invalid room was accepted")
		}
	}
}

func TestTeamSelectionCapacityAndLock(t *testing.T) {
	host, err := HostVersus("127.0.0.1:0", RoomInfo{TeamSize: 2, BestOf: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	var clients []*VersusSession
	for i := 0; i < 3; i++ {
		client, err := JoinVersus(host.listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		clients = append(clients, client)
	}
	if host.Begin() {
		t.Fatal("started before everyone chose a team")
	}
	if !host.SelectTeam(0, 1) {
		t.Fatal("host could not select top")
	}
	clients[0].SendInput(VersusInput{TeamChoice: 2})
	waitVersus(t, func() bool { return host.Teams()[1] == 1 })
	if host.SelectTeam(2, 1) {
		t.Fatal("third player joined full team")
	}
	if host.Teams()[2] != -1 {
		t.Fatal("failed selection changed team")
	}
	clients[1].SendInput(VersusInput{TeamChoice: 1})
	clients[2].SendInput(VersusInput{TeamChoice: 1})
	waitVersus(t, func() bool { return host.Teams() == [4]int{1, 1, 0, 0} })
	if !host.Begin() {
		t.Fatal("balanced teams could not start")
	}
	if host.SelectTeam(0, 0) {
		t.Fatal("team changed during match")
	}
}
