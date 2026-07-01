# Olympics API

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/gin-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white)
![WebSocket](https://img.shields.io/badge/websocket-010101?style=for-the-badge&logo=socketdotio&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

## Overview
A backend for a family "olympics": an opinionated, one-off yard-games tournament built with Go and Gin. It randomly forms teams of 2, runs a round-robin group stage across three games (Darts, Bocce, Cornhole), and seeds a single-elimination playoff bracket, with real-time updates over WebSockets.

## Features
- **Fixed, opinionated format** - Teams of 2 (odd player makes a team of 3), groups of 6, everyone advances
- **Team generation** - Participants are randomly shuffled into pairs
- **Equipment-aware scheduling** - The group round-robin is packed into rounds that respect the available stations (2 darts, 2 bocce, 2 cornhole) so the whole field plays at once and each team's games stay balanced
- **Record-based playoffs** - All teams advance; the bracket is seeded purely by wins, with equal records broken randomly
- **Lifecycle state machine** - setup → teams_generated → group_stage → playoffs → finished
- **Real-time updates** - In-process WebSocket hub broadcasts lifecycle changes, edits, and recorded scores (single replica, no NATS)
- **OIDC Authentication** - Reads are public; writes require the tournament creator or an olympics admin
- **PostgreSQL** - Data persistence with GORM; schema managed by versioned **golang-migrate** migrations applied at startup

## Dependencies
This application requires the following services to be deployed:
- **OIDC Provider** (Authentik) for authentication
- **PostgreSQL** database for data persistence

## Development
1. Copy `.env.example` to `.env` and fill in the values
2. Start required services (PostgreSQL, OIDC provider)
3. Run the API with `make run` (run `make help` to list all available tasks)
