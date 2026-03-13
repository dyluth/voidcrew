package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Constants & Styles ---

var (
	subtle      = lipgloss.AdaptiveColor{Light: "#D9D9D9", Dark: "#383838"}
	highlight   = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	warning     = lipgloss.AdaptiveColor{Light: "#F26D5B", Dark: "#ED567A"}
	dimmed      = lipgloss.AdaptiveColor{Light: "#A0A0A0", Dark: "#505050"}
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#383838")).Bold(true).Padding(0, 1)

	panelStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(subtle).Padding(1)
	alertStyle = lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).BorderForeground(warning).Padding(1)

	selectedStyle     = lipgloss.NewStyle().Foreground(highlight).Bold(true)
	crewSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Underline(true)
	warningStyle      = lipgloss.NewStyle().Foreground(warning).Bold(true)
	resourceStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F")).Bold(true)
	cacheStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#E7BA5D")).Bold(true)
	consoleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#56B6C2")).Bold(true)
	hydroStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379")).Bold(true)
	gasStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD")).Bold(true)
	genStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75")).Bold(true)
	miteStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F00")).Bold(true)
	infestyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#D16286")).Bold(true).Blink(true)
)

// --- Types ---

type TileState int

const (
	TileHidden TileState = iota
	TileExplored
	TileVisible
)

type ResourceType int

const (
	ResNone ResourceType = iota
	ResScrap
	ResElectronics
	ResRations
	ResOxygen
	ResPower
)

type Tile struct {
	Char      string
	State     TileState
	ResType   ResourceType
	ResCount  int
	IsActive  bool
}

type HazardType int

const (
	HazardNone HazardType = iota
	HazardBreach
	HazardFire
	HazardGas
	HazardInfested
)

type Hazard struct {
	Type      HazardType
	X, Y      int
	Integrity int
	Timer     int
	SpawnAcc  int // Spawn accumulator for infested breaches
}

type DenizenType int

const (
	TypeMite DenizenType = iota
)

type Denizen struct {
	Type   DenizenType
	Health float64
	X, Y   int
}

// --- Model Definitions ---

type OrderType int

const (
	OrderNone OrderType = iota
	OrderExplore
	OrderMoveTo
	OrderGatherAuto
	OrderGatherScrap
	OrderGatherElec
	OrderGatherFood
	OrderGatherOxygen
	OrderRepair
	OrderVentilate
	OrderSearchAndDestroy
	OrderHeal
	OrderFixTarget
)

func (o OrderType) String() string {
	return [...]string{"None", "Explore (Auto)", "Move To Cursor", "Gather (Auto)", "Gather (Scrap)", "Gather (Elec)", "Gather (Food)", "Gather (Oxygen)", "Repair (Auto)", "Ventilate Area", "S&D (Auto)", "Heal Squad", "Fix Target"}[o]
}

type CrewMember struct {
	Name    string
	Class   string
	Health  float64
	MaxHP   float64
	XP      int
	X, Y    int
	TargetX int
	TargetY int
	Order   OrderType
	Status  string
	Effects []string // Persistent effects like STARVING, SUFFOCATING
}

func (c *CrewMember) HasEffect(eff string) bool {
	for _, e := range c.Effects { if e == eff { return true } }
	return false
}

func (c *CrewMember) AddEffect(eff string) {
	if !c.HasEffect(eff) { c.Effects = append(c.Effects, eff) }
}

func (c *CrewMember) RemoveEffect(eff string) {
	newEffs := []string{}
	for _, e := range c.Effects { if e != eff { newEffs = append(newEffs, e) } }
	c.Effects = newEffs
}

func (c CrewMember) Level() int {
	return (c.XP / 100) + 1
}

type Model struct {
	Width, Height   int
	Map             [][]Tile
	Crew            []CrewMember
	Hazards         []Hazard
	Denizens        []Denizen
	SelectedCrew    int
	Messages        []string
	TickCount       int
	ShowMenu        bool
	MenuIndex       int
	CursorX, CursorY int
	
	RedAlert        bool
	AlertMsg        string
	WaitingOnIdle   bool
	EvacX, EvacY    int // Airlock location
	LevelCleared    bool

	Oxygen      float64
	Power       float64
	Rations     int
	Scrap       int
	Electronics int
}

func generateLevel(mWidth, mHeight int) ([][]Tile, int, int) {
	gameMap := make([][]Tile, mHeight)
	for y := 0; y < mHeight; y++ {
		gameMap[y] = make([]Tile, mWidth)
		for x := 0; x < mWidth; x++ {
			gameMap[y][x] = Tile{Char: "#", State: TileHidden}
		}
	}

	type Rect struct{ x, y, w, h int }
	rooms := []Rect{}

	for i := 0; i < 25; i++ {
		rw, rh := rand.Intn(6)+5, rand.Intn(4)+5
		rx, ry := rand.Intn(mWidth-rw-4)+2, rand.Intn(mHeight-rh-4)+2
		overlap := false
		for _, r := range rooms {
			if rx < r.x+r.w+2 && rx+rw+2 > r.x && ry < r.y+r.h+2 && ry+rh+2 > r.y { overlap = true; break }
		}
		if !overlap {
			rooms = append(rooms, Rect{rx, ry, rw, rh})
			for y := ry; y < ry+rh; y++ {
				for x := rx; x < rx+rw; x++ { gameMap[y][x] = Tile{Char: ".", State: TileHidden} }
			}
		}
	}

	for i := 0; i < len(rooms)-1; i++ {
		r1, r2 := rooms[i], rooms[i+1]
		cx1, cy1 := r1.x+r1.w/2, r1.y+r1.h/2
		cx2, cy2 := r2.x+r2.w/2, r2.y+r2.h/2
		if rand.Float32() < 0.5 {
			for x := math.Min(float64(cx1), float64(cx2)); x <= math.Max(float64(cx1), float64(cx2)); x++ { gameMap[cy1][int(x)] = Tile{Char: ".", State: TileHidden} }
			for y := math.Min(float64(cy1), float64(cy2)); y <= math.Max(float64(cy1), float64(cy2)); y++ { gameMap[int(y)][cx2] = Tile{Char: ".", State: TileHidden} }
		} else {
			for y := math.Min(float64(cy1), float64(cy2)); y <= math.Max(float64(cy1), float64(cy2)); y++ { gameMap[int(y)][cx1] = Tile{Char: ".", State: TileHidden} }
			for x := math.Min(float64(cx1), float64(cx2)); x <= math.Max(float64(cx1), float64(cx2)); x++ { gameMap[cy2][int(x)] = Tile{Char: ".", State: TileHidden} }
		}
	}

	specialLocs := []ResourceType{ResPower, ResOxygen, ResRations}
	rand.Shuffle(len(rooms), func(i, j int) { rooms[i], rooms[j] = rooms[j], rooms[i] })
	for i, rType := range specialLocs {
		if i < len(rooms) {
			rx, ry := rooms[i].x+rand.Intn(rooms[i].w), rooms[i].y+rand.Intn(rooms[i].h)
			char := "G"; if rType == ResOxygen { char = "L" } else if rType == ResRations { char = "V" }
			gameMap[ry][rx] = Tile{Char: char, State: TileHidden, ResType: rType, ResCount: rand.Intn(30) + 40}
		}
	}

	for y := 0; y < mHeight; y++ {
		for x := 0; x < mWidth; x++ {
			if gameMap[y][x].Char == "." && gameMap[y][x].ResType == ResNone {
				if rand.Float32() < 0.06 {
					resType := ResScrap; if rand.Float32() < 0.25 { resType = ResElectronics }
					gameMap[y][x] = Tile{Char: "%", State: TileHidden, ResType: resType, ResCount: rand.Intn(6) + 3}
				}
			}
		}
	}

	for i := 1; i < len(rooms); i++ {
		numCaches := rand.Intn(3) + 1
		for j := 0; j < numCaches; j++ {
			rx, ry := rooms[i].x+rand.Intn(rooms[i].w), rooms[i].y+rand.Intn(rooms[i].h)
			if gameMap[ry][rx].Char == "." && gameMap[ry][rx].ResType == ResNone {
				resType := ResScrap; if rand.Float32() < 0.4 { resType = ResElectronics }
				gameMap[ry][rx] = Tile{Char: "%", State: TileHidden, ResType: resType, ResCount: rand.Intn(15) + 10}
			}
		}
	}

	startX, startY := rooms[0].x+1, rooms[0].y+1
	gameMap[startY][startX].Char = "E" // Airlock
	return gameMap, startX, startY
}

func initialModel() Model {
	rand.Seed(time.Now().UnixNano())
	mWidth, mHeight := 60, 25
	gameMap, startX, startY := generateLevel(mWidth, mHeight)

	hazards := []Hazard{}
	tunnelsCount := 0
	for attempts := 0; attempts < 100; attempts++ {
		rx, ry := rand.Intn(mWidth), rand.Intn(mHeight)
		dist := math.Abs(float64(rx-startX)) + math.Abs(float64(ry-startY))
		if gameMap[ry][rx].Char == "." && gameMap[ry][rx].ResType == ResNone && dist > 15 {
			hazards = append(hazards, Hazard{Type: HazardInfested, X: rx, Y: ry, Integrity: 3})
			tunnelsCount++
			if tunnelsCount >= 4 { break }
		}
	}

	m := Model{
		Map:      gameMap,
		CursorX:  startX,
		CursorY:  startY,
		EvacX:    startX,
		EvacY:    startY,
		Crew: []CrewMember{
			{Name: "Hicks", Class: "Marine", Health: 150, MaxHP: 150, X: startX, Y: startY, Order: OrderNone, Status: "Idle", Effects: []string{}},
			{Name: "Ripley", Class: "Engineer", Health: 100, MaxHP: 100, X: startX + 1, Y: startY, Order: OrderNone, Status: "Idle", Effects: []string{}},
			{Name: "Dallas", Class: "Scavenger", Health: 100, MaxHP: 100, X: startX, Y: startY + 1, Order: OrderNone, Status: "Idle", Effects: []string{}},
			{Name: "Lambert", Class: "Medic", Health: 100, MaxHP: 100, X: startX + 1, Y: startY + 1, Order: OrderNone, Status: "Idle", Effects: []string{}},
		},
		Hazards:      hazards,
		Denizens:     []Denizen{},
		SelectedCrew: 0,
		Messages:     []string{"System: Squad deployed. Reach Airlock (E) to Evacuate."},
		Oxygen:      100.0,
		Power:       40.0,
		Rations:     60,
		Scrap:       10,
		Electronics: 15,
	}
	m.updateVisibility()
	return m
}

// --- Logic Functions ---

func (m *Model) updateVisibility() {
	for y := range m.Map {
		for x := range m.Map[y] { if m.Map[y][x].State == TileVisible { m.Map[y][x].State = TileExplored } }
	}
	for _, c := range m.Crew {
		radius := 6; if c.Class == "Scavenger" { radius = 9 }
		for y := c.Y - radius; y <= c.Y+radius; y++ {
			for x := c.X - radius; x <= c.X+radius; x++ {
				if y >= 0 && y < len(m.Map) && x >= 0 && x < len(m.Map[0]) {
					dist := math.Sqrt(float64((x-c.X)*(x-c.X) + (y-c.Y)*(y-c.Y)))
					if dist <= float64(radius) { m.Map[y][x].State = TileVisible }
				}
			}
		}
	}
}

func (m *Model) isTileDangerous(x, y int, ignoreTargetX, ignoreTargetY int, isRepairing bool) bool {
	for _, h := range m.Hazards {
		if h.X == ignoreTargetX && h.Y == ignoreTargetY { continue }
		if h.X == x && h.Y == y { return true }
		if !isRepairing && h.Type == HazardFire {
			if math.Abs(float64(h.X-x)) + math.Abs(float64(h.Y-y)) <= 1 { return true }
		}
	}
	return false
}

func (m *Model) getNextStep(startX, startY, targetX, targetY int, isRepairing bool) (int, int) {
	if startX == targetX && startY == targetY { return startX, startY }
	type Point struct{ x, y int }
	queue := []Point{{startX, startY}}
	cameFrom := make(map[Point]Point); cameFrom[Point{startX, startY}] = Point{-1, -1}
	for len(queue) > 0 {
		curr := queue[0]; queue = queue[1:]
		if curr.x == targetX && curr.y == targetY {
			for cameFrom[curr].x != startX || cameFrom[curr].y != startY { curr = cameFrom[curr] }
			return curr.x, curr.y
		}
		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Point{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(m.Map) && next.x >= 0 && next.x < len(m.Map[0]) {
				if m.Map[next.y][next.x].Char == "#" { continue }
				ignX, ignY := -1, -1; if isRepairing { ignX, ignY = targetX, targetY }
				if m.isTileDangerous(next.x, next.y, ignX, ignY, isRepairing) { continue }
				if _, seen := cameFrom[next]; !seen { cameFrom[next] = curr; queue = append(queue, next) }
			}
		}
	}
	return startX, startY
}

func (m *Model) findTargetForExploration(startX, startY int) (int, int, bool) {
	type Point struct{ x, y int }
	queue := []Point{{startX, startY}}
	visited := make(map[Point]bool); visited[Point{startX, startY}] = true
	for len(queue) > 0 {
		curr := queue[0]; queue = queue[1:]
		hasHiddenNeighbor := false
		for _, d := range []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nx, ny := curr.x+d.x, curr.y+d.y
			if ny >= 0 && ny < len(m.Map) && nx >= 0 && nx < len(m.Map[0]) && m.Map[ny][nx].State == TileHidden { hasHiddenNeighbor = true; break }
		}
		if hasHiddenNeighbor { return curr.x, curr.y, true }
		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Point{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(m.Map) && next.x >= 0 && next.x < len(m.Map[0]) && !visited[next] {
				visited[next] = true
				if m.Map[next.y][next.x].Char != "#" && !m.isTileDangerous(next.x, next.y, -1, -1, false) { queue = append(queue, next) }
			}
		}
	}
	return -1, -1, false
}

func (m *Model) findNearestGatherTarget(startX, startY int, filter ResourceType) (int, int, bool) {
	type Point struct{ x, y int }
	queue := []Point{{startX, startY}}
	visited := make(map[Point]bool); visited[Point{startX, startY}] = true
	for len(queue) > 0 {
		curr := queue[0]; queue = queue[1:]
		tile := m.Map[curr.y][curr.x]
		if tile.ResType != ResNone && tile.ResType != ResPower && tile.ResCount > 0 {
			if filter == ResNone || tile.ResType == filter { return curr.x, curr.y, true }
		}
		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Point{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(m.Map) && next.x >= 0 && next.x < len(m.Map[0]) && !visited[next] {
				visited[next] = true
				if m.Map[next.y][next.x].Char != "#" && !m.isTileDangerous(next.x, next.y, -1, -1, false) { queue = append(queue, next) }
			}
		}
	}
	return -1, -1, false
}

func (m *Model) findNearestHazard(startX, startY int) (int, int, bool) {
	nearestDist := 9999.0; tx, ty := -1, -1; found := false
	for _, h := range m.Hazards {
		if h.Type == HazardGas { continue }
		dist := math.Abs(float64(h.X-startX)) + math.Abs(float64(h.Y-startY))
		if dist < nearestDist { nearestDist = dist; tx, ty = h.X, h.Y; found = true }
	}
	for y := range m.Map {
		for x := range m.Map[y] {
			t := m.Map[y][x]; if t.ResType == ResPower && !t.IsActive {
				dist := math.Abs(float64(x-startX)) + math.Abs(float64(y-startY))
				if dist < nearestDist { nearestDist = dist; tx, ty = x, y; found = true }
			}
		}
	}
	return tx, ty, found
}

func (m *Model) findNearestDenizen(startX, startY int) (int, int, bool) {
	nearestDist := 9999.0; tx, ty := -1, -1; found := false
	for _, d := range m.Denizens {
		if m.Map[d.Y][d.X].State != TileHidden {
			dist := math.Abs(float64(d.X-startX)) + math.Abs(float64(d.Y-startY))
			if dist < nearestDist { nearestDist = dist; tx, ty = d.X, d.Y; found = true }
		}
	}
	return tx, ty, found
}

func (m *Model) findNearestInjured(startX, startY int) (int, int, bool) {
	nearestDist := 9999.0; tx, ty := -1, -1; found := false
	for _, c := range m.Crew {
		if c.Health > 0 && c.Health < c.MaxHP {
			dist := math.Abs(float64(c.X-startX)) + math.Abs(float64(c.Y-startY))
			if dist < nearestDist { nearestDist = dist; tx, ty = c.X, c.Y; found = true }
		}
	}
	return tx, ty, found
}

// --- Update Function ---

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.RedAlert || m.WaitingOnIdle {
			if msg.String() == "y" { m.RedAlert = false; m.WaitingOnIdle = false; m.AlertMsg = ""; return m, nil }
			return m, nil
		}
		if m.ShowMenu {
			switch msg.String() {
			case "esc", "o": m.ShowMenu = false
			case "up", "w": if m.MenuIndex > 0 { m.MenuIndex-- }
			case "down", "s": if m.MenuIndex < 12 { m.MenuIndex++ }
			case "enter":
				c := &m.Crew[m.SelectedCrew]
				tile := m.Map[m.CursorY][m.CursorX]
				var hazAtCursor *Hazard
				for i := range m.Hazards { if m.Hazards[i].X == m.CursorX && m.Hazards[i].Y == m.CursorY { hazAtCursor = &m.Hazards[i]; break } }
				switch m.MenuIndex {
				case 0:
					if m.CursorX == m.EvacX && m.CursorY == m.EvacY { m.LevelCleared = true; m.addMessage("EVACUATING... SUCCESS!")
					} else if (hazAtCursor != nil && hazAtCursor.Type != HazardGas) || (tile.ResType == ResPower && !tile.IsActive) || (tile.ResType != ResNone && tile.ResType != ResPower) { c.Order = OrderFixTarget
					} else { c.Order = OrderMoveTo }
				case 1: c.Order = OrderExplore
				case 2: c.Order = OrderMoveTo
				case 3: c.Order = OrderGatherAuto
				case 4: c.Order = OrderGatherScrap
				case 5: c.Order = OrderGatherElec
				case 6: c.Order = OrderGatherFood
				case 7: c.Order = OrderGatherOxygen
				case 8: c.Order = OrderRepair
				case 9: c.Order = OrderVentilate
				case 10: c.Order = OrderSearchAndDestroy
				case 11: c.Order = OrderHeal
				case 12: c.Order = OrderNone
				}
				if c.Order == OrderMoveTo || c.Order == OrderFixTarget { c.TargetX, c.TargetY = m.CursorX, m.CursorY }
				m.ShowMenu = false
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c": return m, tea.Quit
		case "tab": m.SelectedCrew = (m.SelectedCrew + 1) % len(m.Crew)
		case "o": m.ShowMenu = true; m.MenuIndex = 0
		case "w": if m.CursorY > 0 { m.CursorY-- }
		case "s": if m.CursorY < len(m.Map)-1 { m.CursorY++ }
		case "a": if m.CursorX > 0 { m.CursorX-- }
		case "d": if m.CursorX < len(m.Map[0])-1 { m.CursorX++ }
		case "enter", " ":
			hasIdle := false
			for _, c := range m.Crew { if c.Health > 0 && c.Order == OrderNone { hasIdle = true; break } }
			if hasIdle { m.WaitingOnIdle = true; m.AlertMsg = "IDLE CREW DETECTED! Advance anyway? (y/n)"; return m, nil }
			m.TickCount++; m.processTurn(); m.updateVisibility()
		}
	case tea.WindowSizeMsg: m.Width = msg.Width; m.Height = msg.Height
	}
	return m, nil
}

func (m *Model) triggerAlert(msg string) { m.RedAlert = true; m.AlertMsg = msg; m.addMessage(warningStyle.Render(msg)) }

func (m *Model) processTurn() {
	if m.LevelCleared { return }
	m.addMessage(fmt.Sprintf("--- TURN %d ---", m.TickCount))
	if rand.Float32() < 0.02 {
		ry, rx := rand.Intn(len(m.Map)), rand.Intn(len(m.Map[0]))
		if m.Map[ry][rx].State != TileHidden && m.Map[ry][rx].Char == "." {
			hType := HazardBreach; msg := "RED ALERT: HULL BREACH DETECTED!"
			r := rand.Float32(); if r < 0.2 { hType = HazardFire; msg = "RED ALERT: FIRE!" } else if r < 0.4 { hType = HazardGas; msg = "RED ALERT: GAS LEAK!" } else if r < 0.6 { hType = HazardInfested; msg = "CRITICAL ALERT: MITE TUNNEL DETECTED!" }
			m.Hazards = append(m.Hazards, Hazard{Type: hType, X: rx, Y: ry, Integrity: 3, Timer: 15}); m.triggerAlert(msg)
		}
	}
	newHazards := []Hazard{}
	for _, h := range m.Hazards {
		keep := true
		if h.Type == HazardFire && rand.Float32() < 0.10 {
			dirs := [][]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}; d := dirs[rand.Intn(4)]; nx, ny := h.X+d[0], h.Y+d[1]
			isP := false; for _, c := range m.Crew { if c.Health > 0 && (c.Order == OrderRepair || c.Order == OrderFixTarget) { if math.Abs(float64(c.X-nx)) + math.Abs(float64(c.Y-ny)) <= 1 { isP = true; break } } }
			if !isP && ny >= 0 && ny < len(m.Map) && nx >= 0 && nx < len(m.Map[0]) && m.Map[ny][nx].Char == "." {
				exists := false; for _, ex := range m.Hazards { if ex.X == nx && ex.Y == ny { exists = true; break } }
				if !exists { newHazards = append(newHazards, Hazard{Type: HazardFire, X: nx, Y: ny, Integrity: 2}) }
			}
		} else if h.Type == HazardGas { h.Timer--; if h.Timer <= 0 { keep = false }
		} else if h.Type == HazardInfested { h.SpawnAcc++; if h.SpawnAcc >= 8 { h.SpawnAcc = 0; m.Denizens = append(m.Denizens, Denizen{Type: TypeMite, Health: 10, X: h.X, Y: h.Y}); m.addMessage("System: Mites emerging!") } }
		if keep { newHazards = append(newHazards, h) }
	}
	m.Hazards = newHazards
	for i := range m.Denizens {
		d := &m.Denizens[i]; tx, ty, f := m.findNearestGatherTarget(d.X, d.Y, ResNone)
		if f { nx, ny := m.getNextStep(d.X, d.Y, tx, ty, false); d.X, d.Y = nx, ny; if d.X == tx && d.Y == ty && m.Map[d.Y][d.X].ResCount > 0 { m.Map[d.Y][d.X].ResCount--; if m.Map[d.Y][d.X].ResCount <= 0 { m.Map[d.Y][d.X].ResType = ResNone; m.Map[d.Y][d.X].Char = "." } }
		} else { dirs := [][]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}; dir := dirs[rand.Intn(4)]; nx, ny := d.X+dir[0], d.Y+dir[1]; if ny >= 0 && ny < len(m.Map) && nx >= 0 && nx < len(m.Map[0]) && m.Map[ny][nx].Char == "." { d.X, d.Y = nx, ny } }
	}
	breaches := 0; for _, h := range m.Hazards { if h.Type == HazardBreach || h.Type == HazardInfested { breaches++ } }
	o2D := 0.0; for _, c := range m.Crew { if c.Health > 0 { charO2 := 0.2; if c.Class == "Medic" { charO2 = 0.1 }; o2D += charO2 } }
	m.Oxygen = math.Max(0, m.Oxygen-(o2D+float64(breaches)*0.5))
	if m.Oxygen <= 0 { m.triggerAlert("CRITICAL: OXYGEN DEPLETED!") }
	for y := range m.Map { for x := range m.Map[y] { if m.Map[y][x].ResType == ResPower && m.Map[y][x].IsActive { m.Power += 2.0 } } }
	for i := range m.Crew {
		c := &m.Crew[i]; if c.Health <= 0 { continue }
		if c.Health < c.MaxHP && m.Rations > 0 {
			m.Rations--
			c.Health = math.Min(c.MaxHP, c.Health+1)
			c.RemoveEffect("STARVING")
		} else if m.Rations <= 0 {
			if !c.HasEffect("STARVING") { m.triggerAlert(fmt.Sprintf("%s is STARVING!", c.Name)) }
			c.AddEffect("STARVING")
			if m.TickCount%5 == 0 { c.Health -= 1 }
		} else {
			c.RemoveEffect("STARVING")
		}
		if m.Oxygen <= 0 { if !c.HasEffect("SUFFOCATING") { m.triggerAlert(fmt.Sprintf("%s is SUFFOCATING!", c.Name)) }; c.AddEffect("SUFFOCATING"); c.Health -= 2
		} else { c.RemoveEffect("SUFFOCATING") }
		isP := (c.Order == OrderRepair || c.Order == OrderFixTarget)
		if !isP {
			for _, h := range m.Hazards {
				dist := math.Abs(float64(h.X-c.X)) + math.Abs(float64(h.Y-c.Y))
				if h.Type == HazardFire && dist <= 1 { c.Health -= 10; m.triggerAlert(fmt.Sprintf("%s BURNED!", c.Name)) }
				if h.Type == HazardGas && dist == 0 { c.Health -= 5; m.triggerAlert(fmt.Sprintf("%s CHOKING!", c.Name)) }
			}
		}
		newD := []Denizen{}
		for _, d := range m.Denizens {
			if d.X == c.X && d.Y == c.Y {
				dmg := 20.0; if c.Class == "Marine" { dmg = 100 }; d.Health -= dmg
				if d.Health > 0 { newD = append(newD, d); c.Health -= 5; m.triggerAlert(fmt.Sprintf("%s BIT BY MITE!", c.Name)) } else { m.addMessage(fmt.Sprintf("%s squashed a Mite.", c.Name)); c.XP += 10 }
			} else { newD = append(newD, d) }
		}
		m.Denizens = newD
		if c.Health <= 0 { m.triggerAlert(fmt.Sprintf("%s HAS PERISHED.", c.Name)); continue }
		if c.Order == OrderNone { c.Status = "Idle"; continue }
		c.XP += 2
		if c.Order == OrderVentilate {
			if m.Map[c.Y][c.X].ResType == ResOxygen {
				if m.Power >= 10 { m.Power -= 10; fH := []Hazard{}; for _, h := range m.Hazards { if h.Type != HazardGas { fH = append(fH, h) } }; m.Hazards = fH; m.addMessage(fmt.Sprintf("%s cleared gas clouds.", c.Name)); c.Order = OrderNone; c.Status = "Ventilated"
				} else { c.Status = "Power Required" }
			} else { c.Status = "Move to Console" }
			continue
		}
		switch c.Order {
		case OrderExplore: tx, ty, f := m.findTargetForExploration(c.X, c.Y); if f { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" }
		case OrderSearchAndDestroy: tx, ty, f := m.findNearestDenizen(c.X, c.Y); if !f { tx, ty, f = m.findTargetForExploration(c.X, c.Y) }; if f { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Area Secure" }
		case OrderRepair: tx, ty, f := m.findNearestHazard(c.X, c.Y); if f { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" }
		case OrderHeal: tx, ty, f := m.findNearestInjured(c.X, c.Y); if f { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" }
		case OrderGatherAuto, OrderGatherScrap, OrderGatherElec, OrderGatherFood, OrderGatherOxygen:
			tile := m.Map[c.Y][c.X]; filter := ResNone; if c.Order == OrderGatherScrap { filter = ResScrap } else if c.Order == OrderGatherElec { filter = ResElectronics } else if c.Order == OrderGatherFood { filter = ResRations } else if c.Order == OrderGatherOxygen { filter = ResOxygen }
			if c.Order == OrderGatherAuto { if m.Oxygen < 30 { filter = ResOxygen } else if m.Rations < 10 { filter = ResRations } }
			if tile.ResType == ResNone || tile.ResCount <= 0 || (filter != ResNone && tile.ResType != filter) || tile.ResType == ResPower { tx, ty, f := m.findNearestGatherTarget(c.X, c.Y, filter); if !f && c.Order == OrderGatherAuto { tx, ty, f = m.findNearestGatherTarget(c.X, c.Y, ResNone) }; if f { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" } } else { c.TargetX, c.TargetY = c.X, c.Y }
		}
		if c.X == c.TargetX && c.Y == c.TargetY {
			tile := &m.Map[c.Y][c.X]
			if c.Order == OrderHeal {
				tIdx := -1; for idx, t := range m.Crew { if idx != i && t.X == c.X && t.Y == c.Y && t.Health > 0 && t.Health < t.MaxHP { tIdx = idx; break } }
				if tIdx != -1 { if m.Rations >= 2 && m.Power >= 5 { m.Rations -= 2; m.Power -= 5; m.Crew[tIdx].Health = math.Min(m.Crew[tIdx].MaxHP, m.Crew[tIdx].Health+20); c.Status = fmt.Sprintf("Healing %s", m.Crew[tIdx].Name) } else { c.Status = "Supplies Required" } } else { c.Status = "No patient" }
			} else if isP {
				fIdx := -1; for idx, h := range m.Hazards { if h.Type != HazardGas && h.X == c.X && h.Y == c.Y { fIdx = idx; break } }
				if fIdx != -1 {
					h := &m.Hazards[fIdx]; cost := 5; if h.Type == HazardInfested { cost = 8 }
					if (h.Type == HazardBreach || h.Type == HazardInfested) && m.Scrap < cost && h.Integrity == 3 { m.addMessage(warningStyle.Render(fmt.Sprintf("%s: Needs %d Scrap!", c.Name, cost))); c.Order = OrderNone; c.Status = "Idle (No Scrap)" } else {
						if (h.Type == HazardBreach || h.Type == HazardInfested) && h.Integrity == 3 { m.Scrap -= cost }; pwr := 1; if c.Class == "Engineer" { pwr = 2 }; h.Integrity -= pwr; c.Status = "Fixing..."
						if h.Integrity <= 0 { hN := "hazard"; if h.Type == HazardFire { hN = "fire" } else if h.Type == HazardBreach { hN = "breach" } else if h.Type == HazardInfested { hN = "tunnel" }; m.Hazards = append(m.Hazards[:fIdx], m.Hazards[fIdx+1:]...); m.addMessage(fmt.Sprintf("%s resolved %s!", c.Name, hN)); c.Order = OrderNone }
					}
				} else if tile.ResType == ResPower && !tile.IsActive { if m.Electronics >= 10 { m.Electronics -= 10; tile.IsActive = true; tile.Char = "G"; c.Order = OrderNone; c.Status = "Power Online" } else { m.addMessage(warningStyle.Render(fmt.Sprintf("%s: Needs 10 Electronics!", c.Name))); c.Order = OrderNone; c.Status = "Idle (No Parts)" } }
			} else if tile.ResType != ResNone && tile.ResCount > 0 {
				tile.ResCount--; switch tile.ResType {
				case ResScrap: y := rand.Intn(3) + 1; if c.Class == "Scavenger" { y++ }; m.Scrap += y; c.Status = "Scrap+"
				case ResElectronics: y := rand.Intn(2) + 1; if c.Class == "Scavenger" { y++ }; m.Electronics += y; c.Status = "Elec+"
				case ResRations: y := rand.Intn(5) + 3; if c.Class == "Medic" { y += 2 }; m.Rations += y; c.Status = "Food+"
				case ResOxygen: if m.Power >= 2 { m.Power -= 2; m.Oxygen = math.Min(1000, m.Oxygen+5); c.Status = "O2+" } else { tile.ResCount++; c.Status = "No Power" }
				}
				if tile.ResCount <= 0 { tile.ResType = ResNone; tile.Char = "." }
			} else { c.Status = "Arrived"; if c.Order == OrderMoveTo { c.Order = OrderNone } }
			continue
		}
		nextX, nextY := m.getNextStep(c.X, c.Y, c.TargetX, c.TargetY, isP); if nextX != c.X || nextY != c.Y { c.X, c.Y = nextX, nextY; c.Status = "Moving" } else { c.Status = "Path Blocked" }
	}
}

func (m *Model) addMessage(msg string) {
	m.Messages = append(m.Messages, msg)
	if len(m.Messages) > 12 { m.Messages = m.Messages[1:] }
}

func (m Model) View() string {
	if m.Width == 0 { return "Initializing..." }
	oxCol := resourceStyle; if m.Oxygen < 20 { oxCol = warningStyle }
	header := headerStyle.Width(m.Width).Render(fmt.Sprintf(" OXYGEN: %s | POWER: %s | RATIONS: %s | SCRAP: %s | ELECTRONICS: %s ",
		oxCol.Render(fmt.Sprintf("%.1f", m.Oxygen)), resourceStyle.Render(fmt.Sprintf("%.0f", m.Power)),
		resourceStyle.Render(fmt.Sprintf("%d", m.Rations)), resourceStyle.Render(fmt.Sprintf("%d", m.Scrap)), resourceStyle.Render(fmt.Sprintf("%d", m.Electronics))))

	var mapStr strings.Builder
	for y, row := range m.Map {
		for x, tile := range row {
			var char string; var style lipgloss.Style
			switch tile.State {
			case TileHidden: char = "?"; style = lipgloss.NewStyle().Foreground(subtle)
			case TileExplored:
				char = tile.Char; style = lipgloss.NewStyle().Foreground(dimmed)
				if tile.ResType == ResScrap || tile.ResType == ResElectronics { style = style.Foreground(lipgloss.Color("#705020")) }
				if tile.ResType == ResRations { style = style.Foreground(lipgloss.Color("#507020")) }
				if tile.ResType == ResOxygen { style = style.Foreground(lipgloss.Color("#205070")) }
				if tile.ResType == ResPower { style = style.Foreground(lipgloss.Color("#702020")) }
			case TileVisible:
				char = tile.Char; style = lipgloss.NewStyle()
				if tile.ResType == ResScrap || tile.ResType == ResElectronics { style = cacheStyle }
				if tile.ResType == ResRations { style = hydroStyle }
				if tile.ResType == ResOxygen { style = consoleStyle }
				if tile.ResType == ResPower { style = genStyle; if !tile.IsActive { style = style.Faint(true) } }
			}
			if tile.State != TileHidden {
				for _, h := range m.Hazards {
					if h.X == x && h.Y == y {
						if h.Type == HazardBreach { char = "!"; style = warningStyle }
						if h.Type == HazardFire { char = "*"; style = warningStyle }
						if h.Type == HazardGas { char = "~"; style = gasStyle }
						if h.Type == HazardInfested { char = "&"; style = infestyle }
						break
					}
				}
				for _, d := range m.Denizens { if d.X == x && d.Y == y { char = "m"; style = miteStyle; break } }
			}
			for i, crew := range m.Crew {
				if crew.Health > 0 && crew.X == x && crew.Y == y {
					char = "@"; if i == m.SelectedCrew { style = crewSelectedStyle } else { style = selectedStyle }; break
				}
			}
			renderedChar := style.Render(char)
			if m.CursorX == x && m.CursorY == y { renderedChar = lipgloss.NewStyle().Background(lipgloss.Color("#00FFFF")).Foreground(lipgloss.Color("#000000")).Bold(true).Render(char) }
			mapStr.WriteString(renderedChar)
		}
		mapStr.WriteString("\n")
	}

	mapPanel := panelStyle.Width(m.Width * 2 / 3).Height(m.Height - 19).Render(mapStr.String())
	var rightStr strings.Builder
	if m.ShowMenu {
		rightStr.WriteString("--- ORDERS ---\n\n")
		var opts []string; tile := m.Map[m.CursorY][m.CursorX]
		var hAtC *Hazard; for i := range m.Hazards { if m.Hazards[i].X == m.CursorX && m.Hazards[i].Y == m.CursorY { hAtC = &m.Hazards[i]; break } }
		if m.CursorX == m.EvacX && m.CursorY == m.EvacY { opts = append(opts, "EVACUATE SHIP")
		} else if hAtC != nil { if hAtC.Type == HazardGas { opts = append(opts, "ENTER GAS CLOUD") } else { opts = append(opts, "FIX HAZARD AT CURSOR") }
		} else if tile.ResType == ResPower && !tile.IsActive { opts = append(opts, "REPAIR GENERATOR")
		} else if tile.ResType != ResNone && tile.ResType != ResPower { opts = append(opts, "GATHER AT CURSOR")
		} else { opts = append(opts, "MOVE TO CURSOR") }
		opts = append(opts, "Explore (Auto)", "Move To Cursor", "Gather (Auto)", "Gather (Scrap)", "Gather (Elec)", "Gather (Food)", "Gather (Oxygen)", "Repair (Auto)", "Ventilate Area", "Search & Destroy", "Heal Squad", "None")
		for i, opt := range opts { if i == m.MenuIndex { rightStr.WriteString(selectedStyle.Render("> "+opt) + "\n") } else { rightStr.WriteString("  " + opt + "\n") } }
	} else {
		rightStr.WriteString("--- SQUAD ---\n\n")
		for i, c := range m.Crew {
			if c.Health <= 0 { rightStr.WriteString(warningStyle.Render(c.Name + " (K.I.A.)") + "\n\n"); continue }
			st := lipgloss.NewStyle(); if i == m.SelectedCrew { st = crewSelectedStyle }
			hpCol := resourceStyle; if c.Health < (c.MaxHP * 0.3) { hpCol = warningStyle }
			effStr := ""; if len(c.Effects) > 0 { effStr = " [" + strings.Join(c.Effects, ",") + "]" }
			rightStr.WriteString(st.Render(fmt.Sprintf("%s [L%d %s]", c.Name, c.Level(), c.Class)) + "\n")
			rightStr.WriteString(fmt.Sprintf(" HP: %s/%s%s\n Task: %s\n Stat: %s\n\n", hpCol.Render(fmt.Sprintf("%.0f", c.Health)), fmt.Sprintf("%.0f", c.MaxHP), warningStyle.Render(effStr), c.Order.String(), c.Status))
		}
	}
	inspectPanel := panelStyle.Width(m.Width / 3).Height(m.Height - 19).Render(rightStr.String())
	msgStyle := panelStyle; msgContent := strings.Join(m.Messages, "\n")
	if m.RedAlert || m.WaitingOnIdle { msgStyle = alertStyle; msgContent += "\n\n" + warningStyle.Render(">>> "+m.AlertMsg+" [Press 'y' to confirm] <<<") }
	msgPanel := msgStyle.Width(m.Width - 4).Height(12).Render(msgContent)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, mapPanel, inspectPanel)
	return lipgloss.JoinVertical(lipgloss.Left, header, topRow, msgPanel)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil { os.Exit(1) }
}
