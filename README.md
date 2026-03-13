# VoidCrew

**VoidCrew** is a turn-based, ASCII-art survival strategy game written in Go. You command a salvage ship and its persistent crew, exploring dangerous space hulks to gather resources and survive the awakening horrors within.

## Gameplay Loop

1.  **The Hub (Strategy Layer)**:
    - **Starmap**: Scan for and select your next salvage target based on risk and reward.
    - **Barracks**: Manage your crew, level up their skills, and heal their wounds.
    - **Workshop**: Craft better equipment and upgrade your ship using salvaged scrap.
    - **Launch Bay**: Choose your squad and invest starting resources for the mission.

2.  **The Hulk (Tactical Layer)**:
    - **Explore**: Navigate rooms and corridors using a smart orders system.
    - **Gather**: Scavenge for Scrap, Electronics, and Food. Manage Oxygen and Power consoles.
    - **Combat**: Defend against evolving threats like Void Mites and (future) Security Drones.
    - **Evacuate**: Reach the airlock (`E`) to return to your ship with your haul.

## Crew Classes

- **Marine**: High HP, hazard resistance, expert combatant. Consumes more food.
- **Engineer**: Fast hazard repairs and base activation.
- **Scavenger**: Extended vision range and higher resource yields.
- **Medic**: Low personal oxygen use and expert squad healing.

## Tech Stack

- **Language**: Go (Golang)
- **UI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI framework)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Architecture**: Elm-style (Model-Update-View)
