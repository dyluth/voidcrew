# CLAUDE.md - VoidCrew AI Guide

This document provides specialized guidance for AI agents working on the VoidCrew project.

## Architecture

- **Framework**: `github.com/charmbracelet/bubbletea` (Tea model-update-view).
- **Styling**: `github.com/charmbracelet/lipgloss` for panel layout and colors.
- **State Management**: The `Model` is the central source of truth. The `Update` function must be kept clean, delegating complex logic to specialized sub-packages (e.g., `game`, `ui`, `map`).

## Key Symbols & Entities

- **@**: Crew Member (Persistent, unique).
- **#**: Unbroken Wall / Hull.
- **X**: Damaged Hull (Requires repair).
- **%**: Resource Cache (Scrap, Electronics).
- **.**: Explored Floor.
- **?**: Unknown Area (Fog of War).

## Design Philosophy

1. **Simplicity First**: The map is a single large grid initially. Focus on the core resource loop before adding multiple levels.
2. **Atmospheric TUI**: Use high-contrast ASCII and minimal but effective color styling (e.g., dim gray for fog, vibrant red for critical alerts).
3. **Turn-Based Integrity**: Each action must advance the game state precisely by one "Tick".
4. **Permanent Consequences**: Crew members are precious. Death should be impactful.

## Common Tasks

- **Adding a new command**: Register a keybinding in the `Update` function and add a corresponding menu item in the CLI-style menu.
- **Modifying UI layout**: Use Lip Gloss `JoinHorizontal` and `JoinVertical` to adjust the three-panel layout.
- **Resource Management**: All resources (Power, Oxygen, etc.) should be handled within a central `GameState` struct.
