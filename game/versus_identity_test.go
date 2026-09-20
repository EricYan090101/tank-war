package game

import "testing"

func TestBothTeamsSeeTheirOwnSpawnAtBottom(t *testing.T) {
	g := versusTestGame(2, 3)
	for id := 0; id < 4; id++ {
		g.Versus = &VersusSession{LocalID: id}
		tank := g.Tanks[id]
		_, y := g.versusPosition(tank.X, tank.Y, 32)
		if y != 384 {
			t.Fatalf("player %d spawn screen y=%v", id, y)
		}
		for _, dir := range []Direction{DirUp, DirDown, DirLeft, DirRight} {
			worldDir := dir
			if g.versusRotated() {
				worldDir = oppositeDirection(dir)
			}
			dx, dy := worldDir.Delta()
			x0, y0 := g.versusPosition(128, 128, 32)
			x1, y1 := g.versusPosition(128+dx, 128+dy, 32)
			wantX, wantY := dir.Delta()
			if x1-x0 != wantX || y1-y0 != wantY {
				t.Fatal("screen controls reversed")
			}
		}
	}
}
func TestNamesCrossNetworkAndSnapshots(t *testing.T) {
	host, err := HostVersus("127.0.0.1:0", RoomInfo{TeamSize: 1, BestOf: 3}, "YellowAce")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := JoinVersus(host.listener.Addr().String(), "GreenAce")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	client.SendInput(VersusInput{})
	waitVersus(t, func() bool { return host.Names()[1] == "GreenAce" })
	g := versusTestGame(1, 3)
	g.Match.Names = host.Names()
	host.Publish(g.versusSnapshot())
	var snap *versusSnapshot
	waitVersus(t, func() bool { snap, _ = client.Receive(); return snap != nil })
	other := &Game{}
	other.applyVersusSnapshot(snap)
	if other.playerName(0) != "YellowAce" || other.playerName(1) != "GreenAce" {
		t.Fatal("lost player names")
	}
	if cleanPlayerName("abcdefghijklmnop!") != "abcdefghijkl" {
		t.Fatal("name length not bounded")
	}
}
