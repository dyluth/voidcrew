# Strategic Layer Design - VoidCrew Hub

## 1. Overview
The "Hub" represents the player's own salvage ship. It serves as the meta-game layer where players manage their crew, upgrade equipment, and select their next mission.

## 2. Hub Sections

### 2.1 The Starmap (Mission Selection)
- **Function**: Scans the sector for derelict hulks.
- **Display**: Presents 2-3 procedurally generated missions at a time.
- **Mission Parameters**:
    - `Hulk Type`: (e.g., Freighter, Science Vessel, Fighter). Affects room styles.
    - `Initial Oxygen`: How much air remains in the hulk (0 to 200).
    - `Resource Richness`: Multiplier for Scrap, Electronics, and Food caches.
    - `Hazard Risk`: Probability of Fire, Breaches, and Gas spawning.
    - `Enemy Profiles`: A list of enemy types and their independent "Awakening Clocks".

### 2.2 The Barracks (Crew Management)
- **Function**: View the full roster of surviving crew.
- **Progression**: Spend XP gained during missions to unlock new skills or improve stats (HP, Carry Capacity).
- **Specialization**: Branching paths for classes (e.g., Marine -> Tank or Scout).
- **Healing**: View persistent injuries and status effects from previous tours.

### 2.3 The Workshop (Crafting & Upgrades)
- **Function**: Spend Scrap and Electronics gathered from missions.
- **Personal Gear**: Build better weapons (higher damage), protective suits (hazard resistance), or specialized tools.
- **Ship Upgrades**: Improve the Hub's capabilities (e.g., "Advanced Scanners" to see more mission parameters).

### 2.4 The Launch Bay (Deployment)
- **Function**: Prepare for the next mission.
- **Squad Selection**: Choose which members from the Barracks to send (up to 4).
- **Resource Investment**: Decide how much Hub Scrap, Food, and Oxygen to "bring along" to give the squad a head start on a low-resource hulk.

## 3. Mission Parameterization (The Awakening Clock)
Every mission has an independent clock for each enemy type:
- **Phase 1: Quiet**: Spawn rate = 0%.
- **Phase 2: Skittering**: Low spawn rate (e.g., 1% per turn).
- **Phase 3: Swarm**: High/Exponential spawn rate.
- *Example*: A ship could have Mites enter "Skittering" at Turn 10, while Drones remain "Quiet" until Turn 50, then immediately jump to "Swarm".

## 4. Failure & Persistence
- **Permadeath**: If a crew member dies in the Tactical Layer, they are removed from the roster permanently.
- **Mission Failure**: If the whole squad is lost, the mission ends. You keep your Hub resources but must recruit new (rookie) crew members using Scrap.
- **Endless Mode**: The game continues until you run out of crew and resources, or (future) reach a specific sector goal.
