package main

import (
	"testing"
)

func TestCrewMemberEffects(t *testing.T) {
	c := CrewMember{Name: "Test"}
	
	if c.HasEffect("STARVING") {
		t.Errorf("Expected no effect, got STARVING")
	}
	
	c.AddEffect("STARVING")
	if !c.HasEffect("STARVING") {
		t.Errorf("Expected effect STARVING, got none")
	}
	
	c.AddEffect("STARVING") // Should not add twice
	if len(c.Effects) != 1 {
		t.Errorf("Expected 1 effect, got %d", len(c.Effects))
	}
	
	c.RemoveEffect("STARVING")
	if c.HasEffect("STARVING") {
		t.Errorf("Expected no effect after removal, got STARVING")
	}
}

func TestCrewMemberLevel(t *testing.T) {
	c := CrewMember{XP: 0}
	if c.Level() != 1 {
		t.Errorf("Expected level 1 for XP 0, got %d", c.Level())
	}
	
	c.XP = 100
	if c.Level() != 2 {
		t.Errorf("Expected level 2 for XP 100, got %d", c.Level())
	}
	
	c.XP = 250
	if c.Level() != 3 {
		t.Errorf("Expected level 3 for XP 250, got %d", c.Level())
	}
}

func TestGenerateTargets(t *testing.T) {
	targets := generateTargets()
	if len(targets) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(targets))
	}
	for _, target := range targets {
		if target.Name == "" {
			t.Errorf("Expected non-empty target name")
		}
		if target.Oxygen <= 0 {
			t.Errorf("Expected positive oxygen, got %f", target.Oxygen)
		}
	}
}

func TestGenerateLevel(t *testing.T) {
	mWidth, mHeight := 60, 25
	gameMap, startX, startY := generateLevel(mWidth, mHeight)
	
	if len(gameMap) != mHeight {
		t.Errorf("Expected map height %d, got %d", mHeight, len(gameMap))
	}
	if len(gameMap[0]) != mWidth {
		t.Errorf("Expected map width %d, got %d", mWidth, len(gameMap[0]))
	}
	
	if gameMap[startY][startX].Char != "E" {
		t.Errorf("Expected Airlock 'E' at start position [%d, %d], got '%s'", startX, startY, gameMap[startY][startX].Char)
	}
}

func TestMissionStateAddMessage(t *testing.T) {
	ms := &MissionState{}
	ms.addMessage("Msg 1")
	if len(ms.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(ms.Messages))
	}
	
	for i := 0; i < 20; i++ {
		ms.addMessage("More msgs")
	}
	
	if len(ms.Messages) > 12 {
		t.Errorf("Expected max 12 messages, got %d", len(ms.Messages))
	}
}

func TestMissionStateTriggerAlert(t *testing.T) {
	ms := &MissionState{}
	ms.triggerAlert("Critical Failure!")
	
	if !ms.RedAlert {
		t.Errorf("Expected RedAlert to be true")
	}
	if ms.AlertMsg != "Critical Failure!" {
		t.Errorf("Expected AlertMsg 'Critical Failure!', got '%s'", ms.AlertMsg)
	}
}

func TestIsTileDangerous(t *testing.T) {
	ms := &MissionState{
		Hazards: []Hazard{
			{Type: HazardFire, X: 10, Y: 10},
			{Type: HazardBreach, X: 5, Y: 5},
		},
	}
	
	// Test direct hit
	if !ms.isTileDangerous(10, 10, -1, -1, false) {
		t.Errorf("Expected tile [10, 10] to be dangerous (fire)")
	}
	
	// Test fire radius (adjacent)
	if !ms.isTileDangerous(10, 11, -1, -1, false) {
		t.Errorf("Expected tile [10, 11] to be dangerous (fire radius)")
	}
	
	// Test ignore target
	if ms.isTileDangerous(10, 10, 10, 10, true) {
		t.Errorf("Expected tile [10, 10] NOT to be dangerous when ignoring it")
	}
	
	// Test ignore fire radius when repairing
	if ms.isTileDangerous(10, 11, 10, 10, true) {
		t.Errorf("Expected tile [10, 11] NOT to be dangerous when repairing target at [10, 10]")
	}
	
	// Test safe tile
	if ms.isTileDangerous(0, 0, -1, -1, false) {
		t.Errorf("Expected tile [0, 0] to be safe")
	}
}

func TestGetNextStep(t *testing.T) {
	mWidth, mHeight := 20, 20
	gameMap := make([][]Tile, mHeight)
	for y := 0; y < mHeight; y++ {
		gameMap[y] = make([]Tile, mWidth)
		for x := 0; x < mWidth; x++ {
			gameMap[y][x] = Tile{Char: ".", State: TileVisible}
		}
	}
	
	ms := &MissionState{
		Map: gameMap,
	}
	
	// Test simple move
	// from (5,5) to (7,7)
	nx, ny := ms.getNextStep(5, 5, 7, 7, false)
	if nx == 5 && ny == 5 {
		t.Errorf("Expected crew to move from [5, 5]")
	}
	
	// Test wall avoidance
	// Move from (5,5) to (5,7). Path should go through (5,6)
	// We put a wall at x=5, y=6
	gameMap[6][5] = Tile{Char: "#", State: TileVisible}
	nx, ny = ms.getNextStep(5, 5, 5, 7, false)
	if nx == 5 && ny == 6 {
		t.Errorf("Expected crew to avoid wall at [x=5, y=6]")
	}
	
	// Test hazard avoidance
	// Move from (5,5) to (5,8).
	// Put fire at x=5, y=7. Its danger radius is 1, so (5,6) is also dangerous.
	ms.Hazards = []Hazard{{Type: HazardFire, X: 5, Y: 7}}
	nx, ny = ms.getNextStep(5, 5, 5, 8, false)
	if (nx == 5 && ny == 7) || (nx == 5 && ny == 6) {
		t.Errorf("Expected crew to avoid fire at [5, 7] and its radius, got [%d, %d]", nx, ny)
	}
}

