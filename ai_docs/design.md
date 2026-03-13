# Tactical Layer Design - VoidCrew Missions

## 1. Overview
The Tactical Layer is the core gameplay experience where the player commands a squad on a derelict hulk. It is turn-based and persistent until the mission ends.

## 2. Core Mechanics

### 2.1 Turn Resolution
- Actions are sequential. Interrupts (damage, alerts) pause the turn.
- **Confirmation**: Red Alerts and Idle warnings require `y` to confirm.

### 2.2 Navigation & Orders
- **Auto-Explore**: Finds nearest hidden tiles.
- **Gather (Auto/Specific)**: Intelligent resource gathering based on current ship needs (Oxygen > Food > Scrap).
- **Search & Destroy**: Prioritizes killing Denizens, then exploring.
- **Contextual Actions**: The menu dynamically offers the best action for the tile under the cursor (Fix, Repair, Gather, Move).

### 2.3 Environmental Hazards
- **Hull Breach (`!`)**: Drains Oxygen. Needs Scrap + Repair.
- **Electrical Fire (`*`)**: Spreads and damages. Needs Repair. Firefighters project a "Firebreak" safety zone.
- **Toxic Gas (`~`)**: Damage over time. Cleared by `Ventilate` at a Console (`L`).
- **Infested Tunnel (`&`)**: Spawns Mites. Needs Scrap + Repair.

### 2.4 Status & Survival
- **Oxygen**: Depletes per crew member + per breach. At 0, causes `SUFFOCATING` (HP damage).
- **Rations**: Consumed only to heal injuries (1 HP = 1 Ration). If 0, causes `STARVING` (HP damage).
- **HP**: If 0, character is K.I.A.

## 3. Map Representation
- **`@`**: Crew Member (Color coded by selection).
- **`#`**: Wall.
- **`.`**: Floor.
- **`E`**: Airlock (Start/Evacuation point).
- **`%`**: Material Cache (Scrap/Electronics).
- **`V`**: Hydroponics (Food).
- **`L`**: Life Support Console (Oxygen/Ventilation).
- **`G`**: Power Generator (Electronics needed to fix).
- **`m`**: Void Mite.
- **`?`**: Fog of War.
