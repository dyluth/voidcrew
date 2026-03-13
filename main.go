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

type ActiveView int

const (
	ViewHub ActiveView = iota
	ViewStarmap
	ViewBarracks
	ViewWorkshop
	ViewLaunchBay
	ViewMission
)

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
	Char     string
	State    TileState
	ResType  ResourceType
	ResCount int
	IsActive bool
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
	SpawnAcc  int
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
	Effects []string
}

func (c *CrewMember) HasEffect(eff string) bool {
	for _, e := range c.Effects {
		if e == eff {
			return true
		}
	}
	return false
}

func (c *CrewMember) AddEffect(eff string) {
	if !c.HasEffect(eff) {
		c.Effects = append(c.Effects, eff)
	}
}

func (c *CrewMember) RemoveEffect(eff string) {
	newEffs := []string{}
	for _, e := range c.Effects {
		if e != eff {
			newEffs = append(newEffs, e)
		}
	}
	c.Effects = newEffs
}

func (c CrewMember) Level() int {
	return (c.XP / 100) + 1
}

// --- Strategic Types ---

type MissionTarget struct {
	Name      string
	HulkType  string
	Oxygen    float64
	Richness  float64
	Risk      float64
	MiteClock int
}

type HubState struct {
	Roster      []CrewMember
	Scrap       int
	Electronics int
	Rations     int
	MenuIndex   int
	Targets     []MissionTarget
}

type MissionState struct {
	Map                  [][]Tile
	Crew                 []CrewMember
	Hazards              []Hazard
	Denizens             []Denizen
	SelectedCrew         int
	Messages             []string
	TickCount            int
	ShowMenu             bool
	MenuIndex            int
	CursorX, CursorY     int
	RedAlert             bool
	AlertMsg             string
	WaitingOnIdle        bool
	EvacX, EvacY         int
	LevelCleared         bool
	Oxygen               float64
	Power                float64
	ScrapCollected       int
	ElectronicsCollected int
	RationsCollected     int
}

func (ms *MissionState) addMessage(msg string) {
	ms.Messages = append(ms.Messages, msg)
	if len(ms.Messages) > 12 {
		ms.Messages = ms.Messages[1:]
	}
}

func (ms *MissionState) triggerAlert(msg string) {
	ms.RedAlert = true
	ms.AlertMsg = msg
	ms.addMessage(warningStyle.Render(msg))
}

type Model struct {
	Width, Height int
	ActiveView    ActiveView
	Hub           HubState
	Mission       *MissionState
}

// --- Level Generation ---

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
			if rx < r.x+r.w+2 && rx+rw+2 > r.x && ry < r.y+r.h+2 && ry+rh+2 > r.y {
				overlap = true
				break
			}
		}
		if !overlap {
			rooms = append(rooms, Rect{rx, ry, rw, rh})
			for y := ry; y < ry+rh; y++ {
				for x := rx; x < rx+rw; x++ {
					gameMap[y][x] = Tile{Char: ".", State: TileHidden}
				}
			}
		}
	}
	for i := 0; i < len(rooms)-1; i++ {
		r1, r2 := rooms[i], rooms[i+1]
		cx1, cy1 := r1.x+r1.w/2, r1.y+r1.h/2
		cx2, cy2 := r2.x+r2.w/2, r2.y+r2.h/2
		if rand.Float32() < 0.5 {
			for x := math.Min(float64(cx1), float64(cx2)); x <= math.Max(float64(cx1), float64(cx2)); x++ {
				gameMap[cy1][int(x)] = Tile{Char: ".", State: TileHidden}
			}
			for y := math.Min(float64(cy1), float64(cy2)); y <= math.Max(float64(cy1), float64(cy2)); y++ {
				gameMap[int(y)][cx2] = Tile{Char: ".", State: TileHidden}
			}
		} else {
			for y := math.Min(float64(cy1), float64(cy2)); y <= math.Max(float64(cy1), float64(cy2)); y++ {
				gameMap[int(y)][cx1] = Tile{Char: ".", State: TileHidden}
			}
			for x := math.Min(float64(cx1), float64(cx2)); x <= math.Max(float64(cx1), float64(cx2)); x++ {
				gameMap[cy2][int(x)] = Tile{Char: ".", State: TileHidden}
			}
		}
	}
	specialLocs := []ResourceType{ResPower, ResOxygen, ResRations}
	rand.Shuffle(len(rooms), func(i, j int) { rooms[i], rooms[j] = rooms[j], rooms[i] })
	for i, rType := range specialLocs {
		if i < len(rooms) {
			rx, ry := rooms[i].x+rand.Intn(rooms[i].w), rooms[i].y+rand.Intn(rooms[i].h)
			char := "G"
			if rType == ResOxygen {
				char = "L"
			} else if rType == ResRations {
				char = "V"
			}
			gameMap[ry][rx] = Tile{Char: char, State: TileHidden, ResType: rType, ResCount: rand.Intn(30) + 40}
		}
	}
	for y := 0; y < mHeight; y++ {
		for x := 0; x < mWidth; x++ {
			if gameMap[y][x].Char == "." && gameMap[y][x].ResType == ResNone {
				if rand.Float32() < 0.06 {
					resType := ResScrap
					if rand.Float32() < 0.25 {
						resType = ResElectronics
					}
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
				resType := ResScrap
				if rand.Float32() < 0.4 {
					resType = ResElectronics
				}
				gameMap[ry][rx] = Tile{Char: "%", State: TileHidden, ResType: resType, ResCount: rand.Intn(15) + 10}
			}
		}
	}
	startX, startY := rooms[0].x+1, rooms[0].y+1
	gameMap[startY][startX].Char = "E"
	return gameMap, startX, startY
}

func initialModel() Model {
	rand.Seed(time.Now().UnixNano())
	return Model{
		ActiveView: ViewHub,
		Hub: HubState{
			Roster: []CrewMember{
				{Name: "Hicks", Class: "Marine", Health: 150, MaxHP: 150, XP: 0, Effects: []string{}},
				{Name: "Ripley", Class: "Engineer", Health: 100, MaxHP: 100, XP: 0, Effects: []string{}},
				{Name: "Dallas", Class: "Scavenger", Health: 100, MaxHP: 100, XP: 0, Effects: []string{}},
				{Name: "Lambert", Class: "Medic", Health: 100, MaxHP: 100, XP: 0, Effects: []string{}},
			},
			Scrap: 50, Electronics: 30, Rations: 100, MenuIndex: 0,
		},
	}
}

func generateTargets() []MissionTarget {
	prefixes := []string{"Derelict", "Abandoned", "Ghost", "Void", "Shattered"}
	types := []string{"Freighter", "Science Lab", "Mining Rig", "Escort", "Tanker"}
	targets := []MissionTarget{}
	for i := 0; i < 3; i++ {
		targets = append(targets, MissionTarget{
			Name:      fmt.Sprintf("%s %s %d", prefixes[rand.Intn(len(prefixes))], types[rand.Intn(len(types))], rand.Intn(900)+100),
			HulkType:  types[rand.Intn(len(types))],
			Oxygen:    float64(rand.Intn(150) + 20),
			Richness:  0.5 + rand.Float64()*1.5,
			Risk:      0.5 + rand.Float64()*1.5,
			MiteClock: rand.Intn(40) + 10,
		})
	}
	return targets
}

func startMission(m *Model, target MissionTarget) {
	mWidth, mHeight := 60, 25
	gameMap, startX, startY := generateLevel(mWidth, mHeight)
	hazards := []Hazard{}
	for attempts := 0; attempts < 100; attempts++ {
		rx, ry := rand.Intn(mWidth), rand.Intn(mHeight)
		dist := math.Abs(float64(rx-startX)) + math.Abs(float64(ry-startY))
		if gameMap[ry][rx].Char == "." && gameMap[ry][rx].ResType == ResNone && dist > 15 {
			hazards = append(hazards, Hazard{Type: HazardInfested, X: rx, Y: ry, Integrity: 3})
			if len(hazards) >= int(4*target.Risk) {
				break
			}
		}
	}
	missionCrew := make([]CrewMember, len(m.Hub.Roster))
	for i, c := range m.Hub.Roster {
		missionCrew[i] = c
		missionCrew[i].X, missionCrew[i].Y = startX+(i%2), startY+(i/2)
	}
	m.Mission = &MissionState{
		Map:      gameMap,
		Crew:     missionCrew,
		Hazards:  hazards,
		Denizens: []Denizen{},
		Messages: []string{fmt.Sprintf("System: Boarding %s. Resources: %.1fx, Risk: %.1fx", target.Name, target.Richness, target.Risk)},
		CursorX:  startX,
		CursorY:  startY,
		EvacX:    startX,
		EvacY:    startY,
		Oxygen:   target.Oxygen,
		Power:    40.0,
	}
	m.Mission.updateVisibility()
	m.ActiveView = ViewMission
}

// --- Mission Logic (Delegated) ---

func (ms *MissionState) updateVisibility() {
	for y := range ms.Map {
		for x := range ms.Map[y] {
			if ms.Map[y][x].State == TileVisible {
				ms.Map[y][x].State = TileExplored
			}
		}
	}
	for _, c := range ms.Crew {
		radius := 6
		if c.Class == "Scavenger" {
			radius = 9
		}
		for y := c.Y - radius; y <= c.Y+radius; y++ {
			for x := c.X - radius; x <= c.X+radius; x++ {
				if y >= 0 && y < len(ms.Map) && x >= 0 && x < len(ms.Map[0]) {
					dist := math.Sqrt(float64((x-c.X)*(x-c.X) + (y-c.Y)*(y-c.Y)))
					if dist <= float64(radius) {
						ms.Map[y][x].State = TileVisible
					}
				}
			}
		}
	}
}

func (ms *MissionState) isTileDangerous(x, y int, ignX, ignY int, isRep bool) bool {
	for _, h := range ms.Hazards {
		if h.X == ignX && h.Y == ignY {
			continue
		}
		if h.X == x && h.Y == y {
			return true
		}
		if !isRep && h.Type == HazardFire {
			if math.Abs(float64(h.X-x))+math.Abs(float64(h.Y-y)) <= 1 {
				return true
			}
		}
	}
	return false
}

func (ms *MissionState) getNextStep(startX, startY, tx, ty int, isRep bool) (int, int) {
	if startX == tx && startY == ty {
		return startX, startY
	}
	type Pt struct{ x, y int }
	queue := []Pt{{startX, startY}}
	cameFrom := make(map[Pt]Pt)
	cameFrom[Pt{startX, startY}] = Pt{-1, -1}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if curr.x == tx && curr.y == ty {
			for cameFrom[curr].x != startX || cameFrom[curr].y != startY {
				curr = cameFrom[curr]
			}
			return curr.x, curr.y
		}
		dirs := []Pt{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Pt{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(ms.Map) && next.x >= 0 && next.x < len(ms.Map[0]) {
				if ms.Map[next.y][next.x].Char == "#" {
					continue
				}
				iX, iY := -1, -1
				if isRep {
					iX, iY = tx, ty
				}
				if ms.isTileDangerous(next.x, next.y, iX, iY, isRep) {
					continue
				}
				if _, seen := cameFrom[next]; !seen {
					cameFrom[next] = curr
					queue = append(queue, next)
				}
			}
		}
	}
	return startX, startY
}

func (ms *MissionState) findTargetForExploration(sx, sy int) (int, int, bool) {
	type Pt struct{ x, y int }
	queue := []Pt{{sx, sy}}
	visited := make(map[Pt]bool)
	visited[Pt{sx, sy}] = true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		hasH := false
		for _, d := range []Pt{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nx, ny := curr.x+d.x, curr.y+d.y
			if ny >= 0 && ny < len(ms.Map) && nx >= 0 && nx < len(ms.Map[0]) && ms.Map[ny][nx].State == TileHidden {
				hasH = true
				break
			}
		}
		if hasH {
			return curr.x, curr.y, true
		}
		dirs := []Pt{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Pt{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(ms.Map) && next.x >= 0 && next.x < len(ms.Map[0]) && !visited[next] {
				visited[next] = true
				if ms.Map[next.y][next.x].Char != "#" && !ms.isTileDangerous(next.x, next.y, -1, -1, false) {
					queue = append(queue, next)
				}
			}
		}
	}
	return -1, -1, false
}

func (ms *MissionState) findNearestGatherTarget(sx, sy int, f ResourceType) (int, int, bool) {
	type Pt struct{ x, y int }
	queue := []Pt{{sx, sy}}
	visited := make(map[Pt]bool)
	visited[Pt{sx, sy}] = true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		tile := ms.Map[curr.y][curr.x]
		if tile.ResType != ResNone && tile.ResType != ResPower && tile.ResCount > 0 {
			if f == ResNone || tile.ResType == f {
				return curr.x, curr.y, true
			}
		}
		dirs := []Pt{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Pt{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(ms.Map) && next.x >= 0 && next.x < len(ms.Map[0]) && !visited[next] {
				visited[next] = true
				if ms.Map[next.y][next.x].Char != "#" && !ms.isTileDangerous(next.x, next.y, -1, -1, false) {
					queue = append(queue, next)
				}
			}
		}
	}
	return -1, -1, false
}

func (ms *MissionState) findNearestHazard(sx, sy int) (int, int, bool) {
	nearest := 9999.0
	tx, ty := -1, -1
	found := false
	for _, h := range ms.Hazards {
		if h.Type == HazardGas {
			continue
		}
		d := math.Abs(float64(h.X-sx)) + math.Abs(float64(h.Y-sy))
		if d < nearest {
			nearest = d
			tx, ty = h.X, h.Y
			found = true
		}
	}
	for y := range ms.Map {
		for x := range ms.Map[y] {
			t := ms.Map[y][x]
			if t.ResType == ResPower && !t.IsActive {
				d := math.Abs(float64(x-sx)) + math.Abs(float64(y-sy))
				if d < nearest {
					nearest = d
					tx, ty = x, y
					found = true
				}
			}
		}
	}
	return tx, ty, found
}

func (ms *MissionState) findNearestDenizen(sx, sy int) (int, int, bool) {
	nearest := 9999.0
	tx, ty := -1, -1
	found := false
	for _, d := range ms.Denizens {
		if ms.Map[d.Y][d.X].State != TileHidden {
			dist := math.Abs(float64(d.X-sx)) + math.Abs(float64(d.Y-sy))
			if dist < nearest {
				nearest = dist
				tx, ty = d.X, d.Y
				found = true
			}
		}
	}
	return tx, ty, found
}

func (ms *MissionState) findNearestInjured(sx, sy int) (int, int, bool) {
	nearest := 9999.0
	tx, ty := -1, -1
	found := false
	for _, c := range ms.Crew {
		if c.Health > 0 && c.Health < c.MaxHP {
			dist := math.Abs(float64(c.X-sx)) + math.Abs(float64(c.Y-sy))
			if dist < nearest {
				nearest = dist
				tx, ty = c.X, c.Y
				found = true
			}
		}
	}
	return tx, ty, found
}

func (ms *MissionState) processTurn() {
	if ms.LevelCleared {
		return
	}
	ms.addMessage(fmt.Sprintf("--- TURN %d ---", ms.TickCount))
	if rand.Float32() < 0.02 {
		ry, rx := rand.Intn(len(ms.Map)), rand.Intn(len(ms.Map[0]))
		if ms.Map[ry][rx].State != TileHidden && ms.Map[ry][rx].Char == "." {
			hType := HazardBreach
			msg := "RED ALERT: HULL BREACH DETECTED!"
			r := rand.Float32()
			if r < 0.2 {
				hType = HazardFire
				msg = "RED ALERT: FIRE!"
			} else if r < 0.4 {
				hType = HazardGas
				msg = "RED ALERT: GAS LEAK!"
			} else if r < 0.6 {
				hType = HazardInfested
				msg = "CRITICAL ALERT: MITE TUNNEL DETECTED!"
			}
			ms.Hazards = append(ms.Hazards, Hazard{Type: hType, X: rx, Y: ry, Integrity: 3, Timer: 15})
			ms.triggerAlert(msg)
		}
	}
	newH := []Hazard{}
	for _, h := range ms.Hazards {
		keep := true
		if h.Type == HazardFire && rand.Float32() < 0.10 {
			dirs := [][]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
			d := dirs[rand.Intn(4)]
			nx, ny := h.X+d[0], h.Y+d[1]
			isP := false
			for _, c := range ms.Crew {
				if c.Health > 0 && (c.Order == OrderRepair || c.Order == OrderFixTarget) {
					if math.Abs(float64(c.X-nx))+math.Abs(float64(c.Y-ny)) <= 1 {
						isP = true
						break
					}
				}
			}
			if !isP && ny >= 0 && ny < len(ms.Map) && nx >= 0 && nx < len(ms.Map[0]) && ms.Map[ny][nx].Char == "." {
				exists := false
				for _, ex := range ms.Hazards {
					if ex.X == nx && ex.Y == ny {
						exists = true
						break
					}
				}
				if !exists {
					newH = append(newH, Hazard{Type: HazardFire, X: nx, Y: ny, Integrity: 2})
				}
			}
		} else if h.Type == HazardGas {
			h.Timer--
			if h.Timer <= 0 {
				keep = false
			}
		} else if h.Type == HazardInfested {
			h.SpawnAcc++
			if h.SpawnAcc >= 8 {
				h.SpawnAcc = 0
				ms.Denizens = append(ms.Denizens, Denizen{Type: TypeMite, Health: 10, X: h.X, Y: h.Y})
				ms.addMessage("System: Mites emerging!")
			}
		}
		if keep {
			newH = append(newH, h)
		}
	}
	ms.Hazards = newH
	for i := range ms.Denizens {
		d := &ms.Denizens[i]
		tx, ty, f := ms.findNearestGatherTarget(d.X, d.Y, ResNone)
		if f {
			nx, ny := ms.getNextStep(d.X, d.Y, tx, ty, false)
			d.X, d.Y = nx, ny
			if d.X == tx && d.Y == ty && ms.Map[d.Y][d.X].ResCount > 0 {
				ms.Map[d.Y][d.X].ResCount--
				if ms.Map[d.Y][d.X].ResCount <= 0 {
					ms.Map[d.Y][d.X].ResType = ResNone
					ms.Map[d.Y][d.X].Char = "."
				}
			}
		} else {
			dirs := [][]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
			dir := dirs[rand.Intn(4)]
			nx, ny := d.X+dir[0], d.Y+dir[1]
			if ny >= 0 && ny < len(ms.Map) && nx >= 0 && nx < len(ms.Map[0]) && ms.Map[ny][nx].Char == "." {
				d.X, d.Y = nx, ny
			}
		}
	}
	br := 0
	for _, h := range ms.Hazards {
		if h.Type == HazardBreach || h.Type == HazardInfested {
			br++
		}
	}
	o2D := 0.0
	for _, c := range ms.Crew {
		if c.Health > 0 {
			charO2 := 0.2
			if c.Class == "Medic" {
				charO2 = 0.1
			}
			o2D += charO2
		}
	}
	ms.Oxygen = math.Max(0, ms.Oxygen-(o2D+float64(br)*0.5))
	if ms.Oxygen <= 0 {
		ms.triggerAlert("CRITICAL: OXYGEN DEPLETED!")
	}
	for y := range ms.Map {
		for x := range ms.Map[y] {
			if ms.Map[y][x].ResType == ResPower && ms.Map[y][x].IsActive {
				ms.Power += 2.0
			}
		}
	}
	for i := range ms.Crew {
		c := &ms.Crew[i]
		if c.Health <= 0 {
			continue
		}
		if c.Health < c.MaxHP && ms.RationsCollected > 0 {
			ms.RationsCollected--
			c.Health = math.Min(c.MaxHP, c.Health+1)
			c.RemoveEffect("STARVING")
		} else if ms.RationsCollected <= 0 {
			if !c.HasEffect("STARVING") {
				ms.triggerAlert(fmt.Sprintf("%s is STARVING!", c.Name))
			}
			c.AddEffect("STARVING")
			if ms.TickCount%5 == 0 {
				c.Health -= 1
			}
		} else {
			c.RemoveEffect("STARVING")
		}
		if ms.Oxygen <= 0 {
			if !c.HasEffect("SUFFOCATING") {
				ms.triggerAlert(fmt.Sprintf("%s is SUFFOCATING!", c.Name))
			}
			c.AddEffect("SUFFOCATING")
			c.Health -= 2
		} else {
			c.RemoveEffect("SUFFOCATING")
		}
		isP := (c.Order == OrderRepair || c.Order == OrderFixTarget)
		if !isP {
			for _, h := range ms.Hazards {
				dist := math.Abs(float64(h.X-c.X)) + math.Abs(float64(h.Y-c.Y))
				if h.Type == HazardFire && dist <= 1 {
					c.Health -= 10
					ms.triggerAlert(fmt.Sprintf("%s BURNED!", c.Name))
				}
				if h.Type == HazardGas && dist == 0 {
					c.Health -= 5
					ms.triggerAlert(fmt.Sprintf("%s CHOKING!", c.Name))
				}
			}
		}
		newD := []Denizen{}
		for _, d := range ms.Denizens {
			if d.X == c.X && d.Y == c.Y {
				dmg := 20.0
				if c.Class == "Marine" {
					dmg = 100
				}
				d.Health -= dmg
				if d.Health > 0 {
					newD = append(newD, d)
					c.Health -= 5
					ms.triggerAlert(fmt.Sprintf("%s BIT BY MITE!", c.Name))
				} else {
					ms.addMessage(fmt.Sprintf("%s squashed a Mite.", c.Name))
					c.XP += 10
				}
			} else {
				newD = append(newD, d)
			}
		}
		ms.Denizens = newD
		if c.Health <= 0 {
			ms.triggerAlert(fmt.Sprintf("%s HAS PERISHED.", c.Name))
			continue
		}
		if c.Order == OrderNone {
			c.Status = "Idle"
			continue
		}
		c.XP += 2
		if c.Order == OrderVentilate {
			if ms.Map[c.Y][c.X].ResType == ResOxygen {
				if ms.Power >= 10 {
					ms.Power -= 10
					fH := []Hazard{}
					for _, h := range ms.Hazards {
						if h.Type != HazardGas {
							fH = append(fH, h)
						}
					}
					ms.Hazards = fH
					ms.addMessage(fmt.Sprintf("%s cleared gas clouds.", c.Name))
					c.Order = OrderNone
					c.Status = "Ventilated"
				} else {
					c.Status = "Power Required"
				}
			} else {
				c.Status = "Move to Console"
			}
			continue
		}
		switch c.Order {
		case OrderExplore:
			tx, ty, f := ms.findTargetForExploration(c.X, c.Y)
			if f {
				c.TargetX, c.TargetY = tx, ty
			} else {
				c.Order = OrderNone
				c.Status = "Idle"
			}
		case OrderSearchAndDestroy:
			tx, ty, f := ms.findNearestDenizen(c.X, c.Y)
			if !f {
				tx, ty, f = ms.findTargetForExploration(c.X, c.Y)
			}
			if f {
				c.TargetX, c.TargetY = tx, ty
			} else {
				c.Order = OrderNone
				c.Status = "Area Secure"
			}
		case OrderRepair:
			tx, ty, f := ms.findNearestHazard(c.X, c.Y)
			if f {
				c.TargetX, c.TargetY = tx, ty
			} else {
				c.Order = OrderNone
				c.Status = "Idle"
			}
		case OrderHeal:
			tx, ty, f := ms.findNearestInjured(c.X, c.Y)
			if f {
				c.TargetX, c.TargetY = tx, ty
			} else {
				c.Order = OrderNone
				c.Status = "Idle"
			}
		case OrderGatherAuto, OrderGatherScrap, OrderGatherElec, OrderGatherFood, OrderGatherOxygen:
			tile := ms.Map[c.Y][c.X]
			filter := ResNone
			if c.Order == OrderGatherScrap {
				filter = ResScrap
			} else if c.Order == OrderGatherElec {
				filter = ResElectronics
			} else if c.Order == OrderGatherFood {
				filter = ResRations
			} else if c.Order == OrderGatherOxygen {
				filter = ResOxygen
			}
			if c.Order == OrderGatherAuto {
				if ms.Oxygen < 30 {
					filter = ResOxygen
				} else if ms.RationsCollected < 10 {
					filter = ResRations
				}
			}
			if tile.ResType == ResNone || tile.ResCount <= 0 || (filter != ResNone && tile.ResType != filter) || tile.ResType == ResPower {
				tx, ty, f := ms.findNearestGatherTarget(c.X, c.Y, filter)
				if !f && c.Order == OrderGatherAuto {
					tx, ty, f = ms.findNearestGatherTarget(c.X, c.Y, ResNone)
				}
				if f {
					c.TargetX, c.TargetY = tx, ty
				} else {
					c.Order = OrderNone
					c.Status = "Idle"
				}
			} else {
				c.TargetX, c.TargetY = c.X, c.Y
			}
		}
		if c.X == c.TargetX && c.Y == c.TargetY {
			tile := &ms.Map[c.Y][c.X]
			if c.Order == OrderHeal {
				tIdx := -1
				for idx, t := range ms.Crew {
					if idx != i && t.X == c.X && t.Y == c.Y && t.Health > 0 && t.Health < t.MaxHP {
						tIdx = idx
						break
					}
				}
				if tIdx != -1 {
					if ms.RationsCollected >= 2 && ms.Power >= 5 {
						ms.RationsCollected -= 2
						ms.Power -= 5
						ms.Crew[tIdx].Health = math.Min(ms.Crew[tIdx].MaxHP, ms.Crew[tIdx].Health+20)
						c.Status = fmt.Sprintf("Healing %s", ms.Crew[tIdx].Name)
					} else {
						c.Status = "Supplies Required"
					}
				} else {
					c.Status = "No patient"
				}
			} else if isP {
				fIdx := -1
				for idx, h := range ms.Hazards {
					if h.Type != HazardGas && h.X == c.X && h.Y == c.Y {
						fIdx = idx
						break
					}
				}
				if fIdx != -1 {
					h := &ms.Hazards[fIdx]
					cost := 5
					if h.Type == HazardInfested {
						cost = 8
					}
					if (h.Type == HazardBreach || h.Type == HazardInfested) && ms.ScrapCollected < cost && h.Integrity == 3 {
						ms.addMessage(warningStyle.Render(fmt.Sprintf("%s: Needs %d Scrap!", c.Name, cost)))
						c.Order = OrderNone
						c.Status = "Idle (No Scrap)"
					} else {
						if (h.Type == HazardBreach || h.Type == HazardInfested) && h.Integrity == 3 {
							ms.ScrapCollected -= cost
						}
						pwr := 1
						if c.Class == "Engineer" {
							pwr = 2
						}
						h.Integrity -= pwr
						c.Status = "Fixing..."
						if h.Integrity <= 0 {
							hN := "hazard"
							if h.Type == HazardFire {
								hN = "fire"
							} else if h.Type == HazardBreach {
								hN = "breach"
							} else if h.Type == HazardInfested {
								hN = "tunnel"
							}
							ms.Hazards = append(ms.Hazards[:fIdx], ms.Hazards[fIdx+1:]...)
							ms.addMessage(fmt.Sprintf("%s resolved %s!", c.Name, hN))
							c.Order = OrderNone
						}
					}
				} else if tile.ResType == ResPower && !tile.IsActive {
					if ms.ElectronicsCollected >= 10 {
						ms.ElectronicsCollected -= 10
						tile.IsActive = true
						tile.Char = "G"
						c.Order = OrderNone
						c.Status = "Power Online"
					} else {
						ms.addMessage(warningStyle.Render(fmt.Sprintf("%s: Needs 10 Electronics!", c.Name)))
						c.Order = OrderNone
						c.Status = "Idle (No Parts)"
					}
				}
			} else if tile.ResType != ResNone && tile.ResCount > 0 {
				tile.ResCount--
				switch tile.ResType {
				case ResScrap:
					y := rand.Intn(3) + 1
					if c.Class == "Scavenger" {
						y++
					}
					ms.ScrapCollected += y
					c.Status = "Scrap+"
				case ResElectronics:
					y := rand.Intn(2) + 1
					if c.Class == "Scavenger" {
						y++
					}
					ms.ElectronicsCollected += y
					c.Status = "Elec+"
				case ResRations:
					y := rand.Intn(5) + 3
					if c.Class == "Medic" {
						y += 2
					}
					ms.RationsCollected += y
					c.Status = "Food+"
				case ResOxygen:
					if ms.Power >= 2 {
						ms.Power -= 2
						ms.Oxygen = math.Min(1000, ms.Oxygen+5)
						c.Status = "O2+"
					} else {
						tile.ResCount++
						c.Status = "No Power"
					}
				}
				if tile.ResCount <= 0 {
					tile.ResType = ResNone
					tile.Char = "."
				}
			} else {
				c.Status = "Arrived"
				if c.Order == OrderMoveTo {
					c.Order = OrderNone
				}
			}
			continue
		}
		nextX, nextY := ms.getNextStep(c.X, c.Y, c.TargetX, c.TargetY, isP)
		if nextX != c.X || nextY != c.Y {
			c.X, c.Y = nextX, nextY
			c.Status = "Moving"
		} else {
			c.Status = "Path Blocked"
		}
	}
}

// --- Main Model Delegation ---

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		switch m.ActiveView {
		case ViewHub:
			return updateHub(&m, msg)
		case ViewStarmap:
			return updateStarmap(&m, msg)
		case ViewBarracks:
			return updateBarracks(&m, msg)
		case ViewWorkshop:
			return updateWorkshop(&m, msg)
		case ViewMission:
			return updateMission(&m, msg)
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}
	return m, nil
}

func updateHub(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "w":
		if m.Hub.MenuIndex > 0 {
			m.Hub.MenuIndex--
		}
	case "down", "s":
		if m.Hub.MenuIndex < 3 {
			m.Hub.MenuIndex++
		}
	case "enter":
		switch m.Hub.MenuIndex {
		case 0:
			m.Hub.Targets = generateTargets()
			m.ActiveView = ViewStarmap
			m.Hub.MenuIndex = 0
		case 1:
			m.ActiveView = ViewBarracks
			m.Hub.MenuIndex = 0
		case 2:
			m.ActiveView = ViewWorkshop
			m.Hub.MenuIndex = 0
		case 3:
			m.Hub.Targets = generateTargets()
			m.ActiveView = ViewStarmap
			m.Hub.MenuIndex = 0
		}
	}
	return m, nil
}

func updateStarmap(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.ActiveView = ViewHub
		m.Hub.MenuIndex = 0
	case "up", "w":
		if m.Hub.MenuIndex > 0 {
			m.Hub.MenuIndex--
		}
	case "down", "s":
		if m.Hub.MenuIndex < len(m.Hub.Targets)-1 {
			m.Hub.MenuIndex++
		}
	case "enter":
		startMission(m, m.Hub.Targets[m.Hub.MenuIndex])
	}
	return m, nil
}

func updateBarracks(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "q" {
		m.ActiveView = ViewHub
		m.Hub.MenuIndex = 1
	}
	return m, nil
}

func updateWorkshop(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "q" {
		m.ActiveView = ViewHub
		m.Hub.MenuIndex = 2
	}
	return m, nil
}

func updateMission(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ms := m.Mission
	if ms.RedAlert || ms.WaitingOnIdle {
		if msg.String() == "y" {
			ms.RedAlert = false
			ms.WaitingOnIdle = false
			ms.AlertMsg = ""
			if ms.LevelCleared {
				m.Hub.Scrap += ms.ScrapCollected
				m.Hub.Electronics += ms.ElectronicsCollected
				m.Hub.Rations += ms.RationsCollected
				for _, c := range ms.Crew {
					for j, rc := range m.Hub.Roster {
						if c.Name == rc.Name {
							m.Hub.Roster[j] = c
						}
					}
				}
				m.ActiveView = ViewHub
				m.Mission = nil
				m.Hub.MenuIndex = 0
			}
			return m, nil
		}
		return m, nil
	}
	if ms.ShowMenu {
		switch msg.String() {
		case "esc", "o":
			ms.ShowMenu = false
		case "up", "w":
			if ms.MenuIndex > 0 {
				ms.MenuIndex--
			}
		case "down", "s":
			if ms.MenuIndex < 12 {
				ms.MenuIndex++
			}
		case "enter":
			c := &ms.Crew[ms.SelectedCrew]
			tile := ms.Map[ms.CursorY][ms.CursorX]
			var hazAtCursor *Hazard
			for i := range ms.Hazards {
				if ms.Hazards[i].X == ms.CursorX && ms.Hazards[i].Y == ms.CursorY {
					hazAtCursor = &ms.Hazards[i]
					break
				}
			}
			switch ms.MenuIndex {
			case 0:
				if ms.CursorX == ms.EvacX && ms.CursorY == ms.EvacY {
					ms.LevelCleared = true
					ms.AlertMsg = "EVACUATION READY! Return to Hub? (y/n)"
				} else if (hazAtCursor != nil && hazAtCursor.Type != HazardGas) || (tile.ResType == ResPower && !tile.IsActive) || (tile.ResType != ResNone && tile.ResType != ResPower) {
					c.Order = OrderFixTarget
				} else {
					c.Order = OrderMoveTo
				}
			case 1:
				c.Order = OrderExplore
			case 2:
				c.Order = OrderMoveTo
			case 3:
				c.Order = OrderGatherAuto
			case 4:
				c.Order = OrderGatherScrap
			case 5:
				c.Order = OrderGatherElec
			case 6:
				c.Order = OrderGatherFood
			case 7:
				c.Order = OrderGatherOxygen
			case 8:
				c.Order = OrderRepair
			case 9:
				c.Order = OrderVentilate
			case 10:
				c.Order = OrderSearchAndDestroy
			case 11:
				c.Order = OrderHeal
			case 12:
				c.Order = OrderNone
			}
			if c.Order == OrderMoveTo || c.Order == OrderFixTarget {
				c.TargetX, c.TargetY = ms.CursorX, ms.CursorY
			}
			ms.ShowMenu = false
		}
		return m, nil
	}
	switch msg.String() {
	case "tab":
		ms.SelectedCrew = (ms.SelectedCrew + 1) % len(ms.Crew)
	case "o":
		ms.ShowMenu = true
		ms.MenuIndex = 0
	case "w":
		if ms.CursorY > 0 {
			ms.CursorY--
		}
	case "s":
		if ms.CursorY < len(ms.Map)-1 {
			ms.CursorY++
		}
	case "a":
		if ms.CursorX > 0 {
			ms.CursorX--
		}
	case "d":
		if ms.CursorX < len(ms.Map[0])-1 {
			ms.CursorX++
		}
	case "enter", " ":
		hI := false
		for _, c := range ms.Crew {
			if c.Health > 0 && c.Order == OrderNone {
				hI = true
				break
			}
		}
		if hI {
			ms.WaitingOnIdle = true
			ms.AlertMsg = "IDLE CREW DETECTED! Advance anyway? (y/n)"
			return m, nil
		}
		ms.TickCount++
		ms.processTurn()
		ms.updateVisibility()
	}
	return m, nil
}

func (m Model) View() string {
	if m.Width == 0 {
		return "Initializing VoidCrew..."
	}
	switch m.ActiveView {
	case ViewHub:
		return viewHub(m)
	case ViewStarmap:
		return viewStarmap(m)
	case ViewBarracks:
		return viewBarracks(m)
	case ViewWorkshop:
		return viewWorkshop(m)
	case ViewMission:
		return viewMission(m)
	default:
		return "View Not Implemented"
	}
}

func viewHub(m Model) string {
	var s strings.Builder
	s.WriteString(headerStyle.Width(m.Width).Render(" VOIDCREW HUB - SALVAGE SHIP 'VAGABOND' "))
	s.WriteString("\n\n GLOBAL STOCK: SCRAP: " + fmt.Sprint(m.Hub.Scrap) + " | ELEC: " + fmt.Sprint(m.Hub.Electronics) + " | RATIONS: " + fmt.Sprint(m.Hub.Rations) + "\n\n")
	menuItems := []string{"Scan for Hulks (Start Mission)", "Barracks (Manage Crew)", "Workshop (Upgrades)", "Launch Bay (Deployment)"}
	for i, item := range menuItems {
		st := lipgloss.NewStyle()
		if i == m.Hub.MenuIndex {
			st = selectedStyle
		}
		s.WriteString(st.Render("> " + item) + "\n")
	}
	return panelStyle.Width(m.Width - 2).Height(m.Height - 4).Render(s.String())
}

func viewBarracks(m Model) string {
	var s strings.Builder
	s.WriteString(headerStyle.Width(m.Width).Render(" SHIP BARRACKS - CREW ROSTER "))
	s.WriteString("\n\n")
	for _, c := range m.Hub.Roster {
		hpCol := resourceStyle
		if c.Health < (c.MaxHP * 0.3) {
			hpCol = warningStyle
		}
		s.WriteString(fmt.Sprintf("> %s [%s]\n", c.Name, c.Class))
		s.WriteString(fmt.Sprintf("   HP: %s/%s | Level: %d | XP: %d\n\n", hpCol.Render(fmt.Sprintf("%.0f", c.Health)), fmt.Sprintf("%.0f", c.MaxHP), c.Level(), c.XP))
	}
	s.WriteString("\n [ESC] Return to Hub")
	return panelStyle.Width(m.Width - 2).Height(m.Height - 4).Render(s.String())
}

func viewWorkshop(m Model) string {
	var s strings.Builder
	s.WriteString(headerStyle.Width(m.Width).Render(" ENGINEERING WORKSHOP "))
	s.WriteString("\n\n [ CONSTRUCTION DECK OFFLINE ]\n\n Collect more Electronics to unlock ship upgrades.\n")
	s.WriteString("\n [ESC] Return to Hub")
	return panelStyle.Width(m.Width - 2).Height(m.Height - 4).Render(s.String())
}

func viewStarmap(m Model) string {
	var s strings.Builder
	s.WriteString(headerStyle.Width(m.Width).Render(" LONG-RANGE SENSORS: AVAILABLE TARGETS "))
	s.WriteString("\n\n")
	for i, t := range m.Hub.Targets {
		st := lipgloss.NewStyle()
		if i == m.Hub.MenuIndex {
			st = selectedStyle
		}
		s.WriteString(st.Render(fmt.Sprintf("> %s (%s)", t.Name, t.HulkType)) + "\n")
		s.WriteString(fmt.Sprintf("   O2 Level: %.0f | Resources: %.1fx | Risk: %.1fx\n\n", t.Oxygen, t.Richness, t.Risk))
	}
	s.WriteString("\n [ESC] Return to Hub | [ENTER] Board Hulk")
	return panelStyle.Width(m.Width - 2).Height(m.Height - 4).Render(s.String())
}

func viewMission(m Model) string {
	ms := m.Mission
	oxCol := resourceStyle
	if ms.Oxygen < 20 {
		oxCol = warningStyle
	}
	header := headerStyle.Width(m.Width).Render(fmt.Sprintf(" OXYGEN: %s | POWER: %s | RATIONS: %d | SCRAP: %d | ELEC: %d ",
		oxCol.Render(fmt.Sprintf("%.1f", ms.Oxygen)), resourceStyle.Render(fmt.Sprintf("%.0f", ms.Power)),
		ms.RationsCollected, ms.ScrapCollected, ms.ElectronicsCollected))
	var mapStr strings.Builder
	for y, row := range ms.Map {
		for x, tile := range row {
			var char string
			var style lipgloss.Style
			switch tile.State {
			case TileHidden:
				char = "?"
				style = lipgloss.NewStyle().Foreground(subtle)
			case TileExplored:
				char = tile.Char
				style = lipgloss.NewStyle().Foreground(dimmed)
				if tile.ResType == ResScrap || tile.ResType == ResElectronics {
					style = style.Foreground(lipgloss.Color("#705020"))
				}
				if tile.ResType == ResRations {
					style = style.Foreground(lipgloss.Color("#507020"))
				}
				if tile.ResType == ResOxygen {
					style = style.Foreground(lipgloss.Color("#205070"))
				}
				if tile.ResType == ResPower {
					style = style.Foreground(lipgloss.Color("#702020"))
				}
			case TileVisible:
				char = tile.Char
				style = lipgloss.NewStyle()
				if tile.ResType == ResScrap || tile.ResType == ResElectronics {
					style = cacheStyle
				}
				if tile.ResType == ResRations {
					style = hydroStyle
				}
				if tile.ResType == ResOxygen {
					style = consoleStyle
				}
				if tile.ResType == ResPower {
					style = genStyle
					if !tile.IsActive {
						style = style.Faint(true)
					}
				}
			}
			if tile.State != TileHidden {
				for _, h := range ms.Hazards {
					if h.X == x && h.Y == y {
						if h.Type == HazardBreach {
							char = "!"
							style = warningStyle
						}
						if h.Type == HazardFire {
							char = "*"
							style = warningStyle
						}
						if h.Type == HazardGas {
							char = "~"
							style = gasStyle
						}
						if h.Type == HazardInfested {
							char = "&"
							style = infestyle
						}
						break
					}
				}
				for _, d := range ms.Denizens {
					if d.X == x && d.Y == y {
						char = "m"
						style = miteStyle
						break
					}
				}
			}
			for i, crew := range ms.Crew {
				if crew.Health > 0 && crew.X == x && crew.Y == y {
					char = "@"
					if i == ms.SelectedCrew {
						style = crewSelectedStyle
					} else {
						style = selectedStyle
					}
					break
				}
			}
			renderedChar := style.Render(char)
			if ms.CursorX == x && ms.CursorY == y {
				renderedChar = lipgloss.NewStyle().Background(lipgloss.Color("#00FFFF")).Foreground(lipgloss.Color("#000000")).Bold(true).Render(char)
			}
			mapStr.WriteString(renderedChar)
		}
		mapStr.WriteString("\n")
	}
	mapPanel := panelStyle.Width(m.Width * 2 / 3).Height(m.Height - 19).Render(mapStr.String())
	var rightStr strings.Builder
	if ms.ShowMenu {
		rightStr.WriteString("--- ORDERS ---\n\n")
		var opts []string
		tile := ms.Map[ms.CursorY][ms.CursorX]
		var hAtC *Hazard
		for i := range ms.Hazards {
			if ms.Hazards[i].X == ms.CursorX && ms.Hazards[i].Y == ms.CursorY {
				hAtC = &ms.Hazards[i]
				break
			}
		}
		if ms.CursorX == ms.EvacX && ms.CursorY == ms.EvacY {
			opts = append(opts, "EVACUATE SHIP")
		} else if hAtC != nil {
			if hAtC.Type == HazardGas {
				opts = append(opts, "ENTER GAS CLOUD")
			} else {
				opts = append(opts, "FIX HAZARD AT CURSOR")
			}
		} else if tile.ResType == ResPower && !tile.IsActive {
			opts = append(opts, "REPAIR GENERATOR")
		} else if tile.ResType != ResNone && tile.ResType != ResPower {
			opts = append(opts, "GATHER AT CURSOR")
		} else {
			opts = append(opts, "MOVE TO CURSOR")
		}
		opts = append(opts, "Explore (Auto)", "Move To Cursor", "Gather (Auto)", "Gather (Scrap)", "Gather (Elec)", "Gather (Food)", "Gather (Oxygen)", "Repair (Auto)", "Ventilate Area", "Search & Destroy", "Heal Squad", "None")
		for i, opt := range opts {
			if i == ms.MenuIndex {
				rightStr.WriteString(selectedStyle.Render("> " + opt) + "\n")
			} else {
				rightStr.WriteString("  " + opt + "\n")
			}
		}
	} else {
		rightStr.WriteString("--- SQUAD ---\n\n")
		for i, c := range ms.Crew {
			if c.Health <= 0 {
				rightStr.WriteString(warningStyle.Render(c.Name + " (K.I.A.)") + "\n\n")
				continue
			}
			st := lipgloss.NewStyle()
			if i == ms.SelectedCrew {
				st = crewSelectedStyle
			}
			hpCol := resourceStyle
			if c.Health < (c.MaxHP * 0.3) {
				hpCol = warningStyle
			}
			effStr := ""
			if len(c.Effects) > 0 {
				effStr = " [" + strings.Join(c.Effects, ",") + "]"
			}
			rightStr.WriteString(st.Render(fmt.Sprintf("%s [L%d %s]", c.Name, c.Level(), c.Class)) + "\n")
			rightStr.WriteString(fmt.Sprintf(" HP: %s/%s%s\n Task: %s\n Stat: %s\n\n", hpCol.Render(fmt.Sprintf("%.0f", c.Health)), fmt.Sprintf("%.0f", c.MaxHP), warningStyle.Render(effStr), c.Order.String(), c.Status))
		}
	}
	inspectPanel := panelStyle.Width(m.Width / 3).Height(m.Height - 19).Render(rightStr.String())
	msgStyle := panelStyle
	msgContent := strings.Join(ms.Messages, "\n")
	if ms.RedAlert || ms.WaitingOnIdle {
		msgStyle = alertStyle
		msgContent += "\n\n" + warningStyle.Render(">>> "+ms.AlertMsg+" [Press 'y' to confirm] <<<")
	}
	msgPanel := msgStyle.Width(m.Width - 4).Height(12).Render(msgContent)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, mapPanel, inspectPanel)
	return lipgloss.JoinVertical(lipgloss.Left, header, topRow, msgPanel)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
