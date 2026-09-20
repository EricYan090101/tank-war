package game

import "testing"

func TestCoopLobbyCapacityStartAndSnapshots(t *testing.T) {
	for count := 1; count <= 4; count++ {
		host, err := HostCoop("127.0.0.1:0", RoomInfo{RoomName: "Coop"})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(host.Close)
		clients := []*VersusSession{}
		for i := 1; i < count; i++ {
			c, err := JoinVersus(host.listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(c.Close)
			clients = append(clients, c)
			c.SendInput(VersusInput{Dir: DirLeft, Moving: true, Fire: true})
		}
		waitVersus(t, func() bool { return len(host.Connected()) == count && len(host.Inputs()) == count-1 })
		if count == 4 {
			if c, err := JoinVersus(host.listener.Addr().String()); err == nil {
				c.Close()
				t.Fatal("fifth player admitted")
			}
		}
		if host.Info().MaxPlayers != 4 || !host.Begin() || !host.Info().InProgress {
			t.Fatal("cannot start with current player count")
		}
		if c, err := JoinVersus(host.listener.Addr().String()); err == nil {
			c.Close()
			t.Fatal("late join admitted")
		}
		g := &Game{Coop: host, Menu: NewMenuUI(), NetMgr: NewNetworkManager()}
		g.beginCoopBattle()
		g.LobbyMembers = host.Connected()
		g.coopInputs = host.Inputs()
		if len(g.Tanks) != count || g.State != StatePlaying {
			t.Fatal("players did not enter same match")
		}
		for _, tank := range g.Tanks {
			if g.Map.CheckTileCollision(tank.X, tank.Y, 32, 32) {
				t.Fatal("spawn blocked")
			}
		}
		g.updatePlayers()
		g.Score = 1234
		g.Tanks[0].ArmorShield = 2
		g.Enemies[0].ArmorShield = 1
		g.Bullets = append(g.Bullets, &Bullet{Emitter: g.Enemies[0]})
		if g.coopSnapshot().Bullets[len(g.Bullets)-1].Emitter != nil {
			t.Fatal("snapshot retained mutable enemy pointer")
		}
		g.Items = append(g.Items, &Item{})
		host.Publish(g.coopSnapshot())
		for _, c := range clients {
			var snap *versusSnapshot
			waitVersus(t, func() bool { snap, _ = c.Receive(); return snap != nil })
			other := &Game{}
			other.applyCoopSnapshot(snap)
			if other.Tanks[0].ArmorShield != 2 || other.Enemies[0].ArmorShield != 1 || other.Score != 1234 || len(other.Tanks) != count || len(other.Enemies) != len(g.Enemies) || len(other.Items) != 1 || other.Map.Tiles != g.Map.Tiles {
				t.Fatal("coop state mismatch")
			}
		}
	}
}

func TestCoopLobbyDepartureFreesSlot(t *testing.T) {
	host, err := HostCoop("127.0.0.1:0", RoomInfo{})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	c, err := JoinVersus(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	waitVersus(t, func() bool { return len(host.Connected()) == 2 })
	c.Close()
	waitVersus(t, func() bool { return len(host.Connected()) == 1 })
	replacement, err := JoinVersus(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer replacement.Close()
	if replacement.LocalID != 1 {
		t.Fatal("vacated slot not reused")
	}
	host.Close()
	waitVersus(t, func() bool { _, failure := replacement.Receive(); return failure != "" })
}
