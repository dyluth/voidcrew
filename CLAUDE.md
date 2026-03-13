# CLAUDE.md - VoidCrew AI Guide

This document provides specialized guidance for AI agents working on the VoidCrew project.

## Architecture

- **State Management**: The game uses a multi-view `Model`.
    - `ActiveView`: Enum (ViewHub, ViewMission, ViewStarmap, etc.).
    - `HubState`: Persistent data (Global Roster, Global Resources).
    - `MissionState`: Transient data (Tactical Map, Hazards, Denizens).
- **Persistence**: Save/Load squad and resources using JSON serialization.
- **Framework**: `github.com/charmbracelet/bubbletea`.
- **Styling**: `github.com/charmbracelet/lipgloss`.

## Design Philosophy

1.  **Risk/Reward Strategy**: The Meta-game is about choosing the right crew for the right job and knowing when to evacuate.
2.  **Turn-Based Depth**: The Tactical layer remains deterministic and sequential.
3.  **Visual Language**:
    - **Header**: Critical ship resources.
    - **Sidebar**: Detailed squad status and effects.
    - **Alerts**: Explicit confirmation required for bad events.

## Key Meta-Data

- **Airlock (`E`)**: Primary spawn and exit point.
- **Mite Tunnel (`&`)**: Spawner hazard. Must be sealed with Scrap.
- **Status Effects**: `STARVING`, `SUFFOCATING`, etc. recorded in `CrewMember.Effects`.

## Tactical Logic Notes

- **Firebreaks**: Firefighting crew members protect adjacent tiles from spread.
- **Repair Immunity**: Crew are immune to the hazard they are currently fixing.
- **Context Actions**: Menu index 0 is always the most relevant action for the cursor position.
