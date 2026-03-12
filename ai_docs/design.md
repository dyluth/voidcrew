# Design Document - VoidCrew

## 1. Overview

VoidCrew is a squad-management strategy game set on a derelict space hulk. The player issues persistent orders to a squad of 3-5 crew members. The game progresses in turns (Ticks), where each crew member attempts to fulfill their order until completed or interrupted.

## 2. Core Mechanics

### 2.1 Persistent Orders
- **Goal-Oriented**: Instead of direct movement, players assign a goal (e.g., "Scavenge at [10, 20]").
- **Auto-Pathing**: Crew members will move toward their target tile automatically each turn.
- **Continuous Action**: Once at the target, the crew member will perform the action (Scavenge, Repair, Guard) every turn until the player issues a new order or the task is finished.

### 2.2 Sequential Turn Resolution
- When the player advances the turn (Tick), each crew member's action is resolved one by one.
- **Interrupts**: If a crew member encounters a threat (denizen, hazard), their turn ends immediately, and their current order is paused/canceled. The player must decide how to react on the next turn.

### 2.3 Map and Fog of War
- **Map Structure**: A single large grid.
- **Visibility**: Crew members clear Fog of War (`?`) as they move.
- **Explored Areas**: Once explored, areas remain visible but dimmed, showing the last known state.

### 2.4 Resource Management
- **Oxygen/Rations**: Drains every Tick.
- **Scrap/Electronics**: Gained from the "Scavenge" order.

## 3. UI Panels

### 3.1 Map Panel (Left/Center)
- Displays the grid, crew locations (`@`), and their current targets (`X`).

### 3.2 Inspection/Order Panel (Right)
- **Status**: Shows crew health, skills, and the active persistent order.
- **Order Menu**: A CLI-style menu to change the selected crew member's goal.

### 3.3 Message Panel (Bottom)
- Sequential log: 
  1. "Hicks moved toward Reactor."
  2. "Ripley is scavenging... Found 2 Scrap."
  3. "WARNING: Hicks encountered a Lifeform! Order cancelled."

## 4. Input Controls
- `Tab`: Cycle through crew members.
- `Enter/Space`: Advance Turn (Process one Tick).
- `o`: Toggle Order Menu.
- `w/a/s/d`: Move target cursor (when selecting a destination).
- `q`: Quit.
