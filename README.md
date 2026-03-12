# VoidCrew

**VoidCrew** is a turn-based, ASCII-art survival strategy game written in Go. You command a small squad of survivors struggling to maintain a foothold on a massive, derelict space hulk.

## Game Concept

- **Core Loop**: Explore the hulk's grid, scavenge for vital resources (Power, Oxygen, Rations, Scrap, Electronics), and repair damaged systems.
- **Progression**: What begins as a survival challenge against environmental hazards (hull breaches, failing life support) gradually shifts into a tactical combat struggle as you awaken the hulk's long-dormant denizens.
- **Squad Management**: You manage a small, persistent squad where each member has distinct skills. Keeping them alive is paramount, as new recruits are rare.
- **Turn-Based Strategy**: Every action costs time. Managing the "Tick" of resource consumption vs. the necessity of exploration is the heart of the game.

## Tech Stack

- **Language**: Go (Golang)
- **UI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI framework)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **State Management**: Elm-style architecture (Model-Update-View)

## UI Layout

The game uses a three-panel layout:
1. **Map Display**: An ASCII-based grid representing the space hulk, featuring Fog of War.
2. **Inspection Panel**: Detailed text descriptions of the currently selected crew member, object, or room.
3. **Message Log**: A scrollable history of system messages, event logs, and combat reports.
