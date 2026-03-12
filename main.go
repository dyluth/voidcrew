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
)

type Hazard struct {
	Type      HazardType
	X, Y      int
	Integrity int
	Timer     int
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
	OrderFixTarget
)

func (o OrderType) String() string {
	return [...]string{"None", "Explore (Auto)", "Move To Cursor", "Gather (Auto)", "Gather (Scrap)", "Gather (Elec)", "Gather (Food)", "Gather (Oxygen)", "Repair (Auto)", "Ventilate Area", "Fix Target"}[o]
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
}

func (c CrewMember) Level() int {
	return (c.XP / 100) + 1
}

type Model struct {
	Width, Height   int
	Map             [][]Tile
	Crew            []CrewMember
	Hazards         []Hazard
	SelectedCrew    int
	Messages        []string
	TickCount       int
	ShowMenu        bool
	MenuIndex       int
	CursorX, CursorY int
	
	RedAlert        bool
	AlertMsg        string
	WaitingOnIdle   bool

	Oxygen      float64
	Power       float64
	Rations     int
	Scrap       int
	Electronics int
}

func initialModel() Model {
	rand.Seed(time.Now().UnixNano())
	mWidth, mHeight := 50, 20
	gameMap := make([][]Tile, mHeight)
	for y := 0; y < mHeight; y++ {
		gameMap[y] = make([]Tile, mWidth)
		for x := 0; x < mWidth; x++ {
			char := "."
			resType := ResNone
			resCount := 0
			if x == 0 || x == mWidth-1 || y == 0 || y == mHeight-1 || (x == 25 && y > 2 && y < 17) {
				char = "#"
			} else if rand.Float32() < 0.04 {
				char = "%"; resType = ResScrap
				if rand.Float32() < 0.3 { resType = ResElectronics }
				resCount = rand.Intn(10) + 5
			} else if rand.Float32() < 0.015 {
				char = "V"; resType = ResRations; resCount = rand.Intn(10) + 5
			} else if rand.Float32() < 0.01 {
				char = "L"; resType = ResOxygen; resCount = rand.Intn(10) + 5
			} else if rand.Float32() < 0.008 {
				char = "G"; resType = ResPower; resCount = 1
			}
			gameMap[y][x] = Tile{Char: char, State: TileHidden, ResType: resType, ResCount: resCount}
		}
	}

	m := Model{
		Map:      gameMap,
		CursorX:  5,
		CursorY:  5,
		Crew: []CrewMember{
			{Name: "Hicks", Class: "Marine", Health: 150, MaxHP: 150, X: 5, Y: 5, Order: OrderNone, Status: "Idle"},
			{Name: "Ripley", Class: "Engineer", Health: 100, MaxHP: 100, X: 6, Y: 5, Order: OrderNone, Status: "Idle"},
			{Name: "Dallas", Class: "Scavenger", Health: 100, MaxHP: 100, X: 5, Y: 6, Order: OrderNone, Status: "Idle"},
			{Name: "Lambert", Class: "Medic", Health: 100, MaxHP: 100, X: 6, Y: 6, Order: OrderNone, Status: "Idle"},
		},
		Hazards:      []Hazard{},
		SelectedCrew: 0,
		Messages:     []string{"System: Hull sensors online. Fix Generators (G) for power."},
		Oxygen:      100.0,
		Power:       20.0,
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
		for x := range m.Map[y] {
			if m.Map[y][x].State == TileVisible { m.Map[y][x].State = TileExplored }
		}
	}
	for _, c := range m.Crew {
		radius := 6
		if c.Class == "Scavenger" { radius = 9 }
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

func (m *Model) isTileDangerous(x, y int, ignoreTargetX, ignoreTargetY int) bool {
	for _, h := range m.Hazards {
		if h.X == ignoreTargetX && h.Y == ignoreTargetY { continue }
		if h.X == x && h.Y == y { return true }
		if h.Type == HazardFire {
			dist := math.Abs(float64(h.X-x)) + math.Abs(float64(h.Y-y))
			if dist <= 1 { return true }
		}
	}
	return false
}

func (m *Model) getNextStep(startX, startY, targetX, targetY int, isRepairing bool) (int, int) {
	if startX == targetX && startY == targetY { return startX, startY }
	type Point struct{ x, y int }
	queue := []Point{{startX, startY}}
	cameFrom := make(map[Point]Point)
	cameFrom[Point{startX, startY}] = Point{-1, -1}
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
				ignX, ignY := -1, -1
				if isRepairing { ignX, ignY = targetX, targetY }
				if m.isTileDangerous(next.x, next.y, ignX, ignY) { continue }
				if _, seen := cameFrom[next]; !seen { cameFrom[next] = curr; queue = append(queue, next) }
			}
		}
	}
	return startX, startY
}

func (m *Model) findTargetForExploration(startX, startY int) (int, int, bool) {
	type Point struct{ x, y int }
	queue := []Point{{startX, startY}}
	visited := make(map[Point]bool)
	visited[Point{startX, startY}] = true
	for len(queue) > 0 {
		curr := queue[0]; queue = queue[1:]
		hasHiddenNeighbor := false
		for _, d := range []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nx, ny := curr.x+d.x, curr.y+d.y
			if ny >= 0 && ny < len(m.Map) && nx >= 0 && nx < len(m.Map[0]) {
				if m.Map[ny][nx].State == TileHidden { hasHiddenNeighbor = true; break }
			}
		}
		if hasHiddenNeighbor { return curr.x, curr.y, true }
		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Point{curr.x + d.x, curr.y + d.y}
			if next.y >= 0 && next.y < len(m.Map) && next.x >= 0 && next.x < len(m.Map[0]) && !visited[next] {
				visited[next] = true
				if m.Map[next.y][next.x].Char != "#" && !m.isTileDangerous(next.x, next.y, -1, -1) { 
					queue = append(queue, next) 
				}
			}
		}
	}
	return -1, -1, false
}

func (m *Model) findNearestGatherTarget(startX, startY int, filter ResourceType) (int, int, bool) {
	type Point struct{ x, y int }
	queue := []Point{{startX, startY}}
	visited := make(map[Point]bool)
	visited[Point{startX, startY}] = true
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
				if m.Map[next.y][next.x].Char != "#" && !m.isTileDangerous(next.x, next.y, -1, -1) { 
					queue = append(queue, next) 
				}
			}
		}
	}
	return -1, -1, false
}

func (m *Model) findNearestHazard(startX, startY int) (int, int, bool) {
	nearestDist := 9999.0
	targetX, targetY := -1, -1
	found := false
	for _, h := range m.Hazards {
		if h.Type == HazardGas { continue }
		dist := math.Abs(float64(h.X-startX)) + math.Abs(float64(h.Y-startY))
		if dist < nearestDist { nearestDist = dist; targetX, targetY = h.X, h.Y; found = true }
	}
	for y := range m.Map {
		for x := range m.Map[y] {
			t := m.Map[y][x]
			if t.ResType == ResPower && !t.IsActive {
				dist := math.Abs(float64(x-startX)) + math.Abs(float64(y-startY))
				if dist < nearestDist { nearestDist = dist; targetX, targetY = x, y; found = true }
			}
		}
	}
	return targetX, targetY, found
}

// --- Update Function ---

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.RedAlert || m.WaitingOnIdle {
			if msg.String() == "y" {
				m.RedAlert = false; m.WaitingOnIdle = false; m.AlertMsg = ""; return m, nil
			}
			return m, nil
		}

		if m.ShowMenu {
			switch msg.String() {
			case "esc", "o": m.ShowMenu = false
			case "up", "w": if m.MenuIndex > 0 { m.MenuIndex-- }
			case "down", "s": if m.MenuIndex < 10 { m.MenuIndex++ }
			case "enter":
				c := &m.Crew[m.SelectedCrew]
				// Map Menu Index to OrderType
				if m.MenuIndex == 0 {
					c.Order = OrderFixTarget
				} else if m.MenuIndex == 10 {
					c.Order = OrderNone
				} else {
					c.Order = OrderType(m.MenuIndex)
				}
				
				if c.Order == OrderMoveTo || c.Order == OrderFixTarget { 
					c.TargetX, c.TargetY = m.CursorX, m.CursorY 
				}
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

func (m *Model) triggerAlert(msg string) {
	m.RedAlert = true; m.AlertMsg = msg; m.addMessage(warningStyle.Render(msg))
}

func (m *Model) processTurn() {
	m.addMessage(fmt.Sprintf("--- TURN %d ---", m.TickCount))
	
	if rand.Float32() < 0.02 {
		ry, rx := rand.Intn(len(m.Map)), rand.Intn(len(m.Map[0]))
		if m.Map[ry][rx].State != TileHidden && m.Map[ry][rx].Char == "." {
			hType := HazardBreach; msg := "RED ALERT: HULL BREACH DETECTED!"
			r := rand.Float32()
			if r < 0.3 { hType = HazardFire; msg = "RED ALERT: ELECTRICAL FIRE!" } else if r < 0.5 { hType = HazardGas; msg = "RED ALERT: GAS LEAK!" }
			m.Hazards = append(m.Hazards, Hazard{Type: hType, X: rx, Y: ry, Integrity: 3, Timer: 15})
			m.triggerAlert(msg)
		}
	}
	newHazards := []Hazard{}
	for _, h := range m.Hazards {
		keep := true
		if h.Type == HazardFire && rand.Float32() < 0.10 {
			dirs := [][]int{{0,1},{0,-1},{1,0},{-1,0}}; d := dirs[rand.Intn(4)]
			nx, ny := h.X+d[0], h.Y+d[1]
			if ny >= 0 && ny < len(m.Map) && nx >= 0 && nx < len(m.Map[0]) && m.Map[ny][nx].Char == "." {
				newHazards = append(newHazards, Hazard{Type: HazardFire, X: nx, Y: ny, Integrity: 2})
			}
		} else if h.Type == HazardGas { h.Timer--; if h.Timer <= 0 { keep = false } }
		if keep { newHazards = append(newHazards, h) }
	}
	m.Hazards = newHazards

	breaches := 0
	for _, h := range m.Hazards { if h.Type == HazardBreach { breaches++ } }
	o2Drain := 0.0
	for _, c := range m.Crew { if c.Health > 0 { charO2 := 0.2; if c.Class == "Medic" { charO2 = 0.1 }; o2Drain += charO2 } }
	m.Oxygen = math.Max(0, m.Oxygen - (o2Drain + float64(breaches)*0.5))
	if m.Oxygen <= 0 { m.triggerAlert("CRITICAL: OXYGEN DEPLETED!") }
	for y := range m.Map {
		for x := range m.Map[y] { if m.Map[y][x].ResType == ResPower && m.Map[y][x].IsActive { m.Power += 2.0 } }
	}

	for i := range m.Crew {
		c := &m.Crew[i]
		if c.Health <= 0 { continue }
		eatInterval := 5; if c.Class == "Marine" { eatInterval = 3 }
		if m.TickCount % eatInterval == 0 {
			if m.Rations > 0 { m.Rations-- } else { c.Health -= 5; m.triggerAlert(fmt.Sprintf("%s STARVING!", c.Name)) }
		}
		if m.Oxygen <= 0 { c.Health -= 2; m.addMessage(warningStyle.Render(fmt.Sprintf("%s SUFFOCATING!", c.Name))) }
		for _, h := range m.Hazards {
			if (c.Order == OrderRepair || c.Order == OrderFixTarget) && c.TargetX == h.X && c.TargetY == h.Y { continue }
			dist := math.Abs(float64(h.X-c.X)) + math.Abs(float64(h.Y-c.Y))
			if h.Type == HazardFire && dist <= 1 { c.Health -= 10; m.triggerAlert(fmt.Sprintf("%s BURNED!", c.Name)) }
			if h.Type == HazardGas && dist == 0 { c.Health -= 5; m.triggerAlert(fmt.Sprintf("%s CHOKING!", c.Name)) }
		}
		if c.Health <= 0 { m.triggerAlert(fmt.Sprintf("%s HAS PERISHED.", c.Name)); continue }
		if c.Order == OrderNone { c.Status = "Idle"; continue }
		c.XP += 2

		if c.Order == OrderVentilate {
			if m.Map[c.Y][c.X].ResType == ResOxygen {
				if m.Power >= 10 {
					m.Power -= 10; filtered := []Hazard{}
					for _, h := range m.Hazards { if h.Type != HazardGas { filtered = append(filtered, h) } }
					m.Hazards = filtered; m.addMessage(fmt.Sprintf("%s cleared gas clouds.", c.Name))
					c.Order = OrderNone; c.Status = "Ventilated"
				} else { c.Status = "Power Required" }
			} else { c.Status = "Move to Console" }
			continue
		}

		switch c.Order {
		case OrderExplore:
			tx, ty, found := m.findTargetForExploration(c.X, c.Y)
			if found { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" }
		case OrderRepair:
			tx, ty, found := m.findNearestHazard(c.X, c.Y)
			if found { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" }
		case OrderGatherAuto, OrderGatherScrap, OrderGatherElec, OrderGatherFood, OrderGatherOxygen:
			tile := m.Map[c.Y][c.X]
			filter := ResNone
			if c.Order == OrderGatherScrap { filter = ResScrap } else if c.Order == OrderGatherElec { filter = ResElectronics } else if c.Order == OrderGatherFood { filter = ResRations } else if c.Order == OrderGatherOxygen { filter = ResOxygen }
			if c.Order == OrderGatherAuto { if m.Oxygen < 30 { filter = ResOxygen } else if m.Rations < 10 { filter = ResRations } }
			if tile.ResType == ResNone || tile.ResCount <= 0 || (filter != ResNone && tile.ResType != filter) || tile.ResType == ResPower {
				tx, ty, found := m.findNearestGatherTarget(c.X, c.Y, filter)
				if !found && c.Order == OrderGatherAuto { tx, ty, found = m.findNearestGatherTarget(c.X, c.Y, ResNone) }
				if found { c.TargetX, c.TargetY = tx, ty } else { c.Order = OrderNone; c.Status = "Idle" }
			} else { c.TargetX, c.TargetY = c.X, c.Y }
		}

		isRep := (c.Order == OrderRepair || c.Order == OrderFixTarget)
		if c.X == c.TargetX && c.Y == c.TargetY {
			tile := &m.Map[c.Y][c.X]
			if isRep {
				fIdx := -1
				for idx, h := range m.Hazards { if h.X == c.X && h.Y == c.Y { fIdx = idx; break } }
				if fIdx != -1 {
					h := &m.Hazards[fIdx]
					if h.Type == HazardBreach && m.Scrap < 5 && h.Integrity == 3 { c.Status = "No Scrap" } else {
						if h.Type == HazardBreach && h.Integrity == 3 { m.Scrap -= 5 }
						pwr := 1; if c.Class == "Engineer" { pwr = 2 }; h.Integrity -= pwr; c.Status = "Fixing..."
						if h.Integrity <= 0 { m.Hazards = append(m.Hazards[:fIdx], m.Hazards[fIdx+1:]...); m.addMessage(fmt.Sprintf("%s fixed hazard!", c.Name)); c.Order = OrderNone }
					}
				} else if tile.ResType == ResPower && !tile.IsActive {
					if m.Electronics >= 10 { m.Electronics -= 10; tile.IsActive = true; tile.Char = "G"; c.Order = OrderNone; c.Status = "Power Online" } else { c.Status = "No Parts" }
				}
			} else if tile.ResType != ResNone && tile.ResCount > 0 {
				tile.ResCount--
				switch tile.ResType {
				case ResScrap: y := rand.Intn(3)+1; if c.Class == "Scavenger" { y++ }; m.Scrap += y; c.Status = "Scrap+"
				case ResElectronics: y := rand.Intn(2)+1; if c.Class == "Scavenger" { y++ }; m.Electronics += y; c.Status = "Elec+"
				case ResRations: y := rand.Intn(5)+3; if c.Class == "Medic" { y+=2 }; m.Rations += y; c.Status = "Food+"
				case ResOxygen:
					if m.Power >= 2 { m.Power -= 2; m.Oxygen = math.Min(1000, m.Oxygen+5); c.Status = "O2+" } else { tile.ResCount++; c.Status = "No Power" }
				}
				if tile.ResCount <= 0 { tile.ResType = ResNone; tile.Char = "." }
			} else { c.Status = "Arrived"; if c.Order == OrderMoveTo { c.Order = OrderNone } }
			continue
		}
		nextX, nextY := m.getNextStep(c.X, c.Y, c.TargetX, c.TargetY, isRep)
		c.X, c.Y = nextX, nextY; c.Status = "Moving"
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
			char := tile.Char; style := lipgloss.NewStyle()
			switch tile.State {
			case TileHidden: char = "?"; style = style.Foreground(subtle)
			case TileExplored:
				style = style.Foreground(dimmed)
				if tile.ResType == ResScrap || tile.ResType == ResElectronics { style = style.Foreground(lipgloss.Color("#705020")) }
				if tile.ResType == ResRations { style = style.Foreground(lipgloss.Color("#507020")) }
				if tile.ResType == ResOxygen { style = style.Foreground(lipgloss.Color("#205070")) }
				if tile.ResType == ResPower { style = style.Foreground(lipgloss.Color("#702020")) }
			case TileVisible:
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
						break
					}
				}
			}
			if m.CursorX == x && m.CursorY == y { char = "+"; style = cursorStyle } else {
				for i, crew := range m.Crew {
					if crew.Health > 0 && crew.X == x && crew.Y == y {
						char = "@"; if i == m.SelectedCrew { style = crewSelectedStyle } else { style = selectedStyle }
						break
					}
				}
			}
			mapStr.WriteString(style.Render(char))
		}
		mapStr.WriteString("\n")
	}

	mapPanel := panelStyle.Width(m.Width * 2 / 3).Height(m.Height - 19).Render(mapStr.String())
	var rightStr strings.Builder
	if m.ShowMenu {
		rightStr.WriteString("--- ORDERS ---\n\n")
		ctxAct := "None"; tile := m.Map[m.CursorY][m.CursorX]
		isHaz := false; for _, h := range m.Hazards { if h.X == m.CursorX && h.Y == m.CursorY { isHaz = true; break } }
		if isHaz { ctxAct = "FIX HAZARD AT CURSOR" } else if tile.ResType == ResPower && !tile.IsActive { ctxAct = "REPAIR GENERATOR" } else if tile.ResType != ResNone { ctxAct = "GATHER AT CURSOR" }
		opts := []string{ctxAct, "Explore (Auto)", "Move To Cursor", "Gather (Auto)", "Gather (Scrap)", "Gather (Elec)", "Gather (Food)", "Gather (Oxygen)", "Repair (Auto)", "Ventilate Area", "None"}
		for i, opt := range opts {
			if i == m.MenuIndex { rightStr.WriteString(selectedStyle.Render("> "+opt) + "\n") } else { rightStr.WriteString("  " + opt + "\n") }
		}
	} else {
		rightStr.WriteString("--- SQUAD ---\n\n")
		for i, c := range m.Crew {
			if c.Health <= 0 { rightStr.WriteString(warningStyle.Render(c.Name + " (K.I.A.)") + "\n\n"); continue }
			st := lipgloss.NewStyle(); if i == m.SelectedCrew { st = crewSelectedStyle }
			hpCol := resourceStyle; if c.Health < (c.MaxHP * 0.3) { hpCol = warningStyle }
			rightStr.WriteString(st.Render(fmt.Sprintf("%s [L%d %s]", c.Name, c.Level(), c.Class)) + "\n")
			rightStr.WriteString(fmt.Sprintf(" HP: %s/%s\n Task: %s\n Stat: %s\n\n", hpCol.Render(fmt.Sprintf("%.0f", c.Health)), fmt.Sprintf("%.0f", c.MaxHP), c.Order.String(), c.Status))
		}
	}
	inspectPanel := panelStyle.Width(m.Width / 3).Height(m.Height - 19).Render(rightStr.String())
	
	msgStyle := panelStyle
	msgContent := strings.Join(m.Messages, "\n")
	if m.RedAlert || m.WaitingOnIdle {
		msgStyle = alertStyle
		msgContent += "\n\n" + warningStyle.Render(">>> "+m.AlertMsg+" [Press 'y' to confirm] <<<")
	}
	msgPanel := msgStyle.Width(m.Width - 4).Height(12).Render(msgContent)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, mapPanel, inspectPanel)
	return lipgloss.JoinVertical(lipgloss.Left, header, topRow, msgPanel)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil { os.Exit(1) }
}
