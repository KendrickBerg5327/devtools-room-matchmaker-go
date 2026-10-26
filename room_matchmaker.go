package main

import (
	"fmt"
	"sort"
)

type Room struct {
	Name            string `json:"name"`
	MaxParticipants int    `json:"max_participants"`
	Participants    int    `json:"participants"`
}

type BuildEvent struct {
	BuildID string `json:"build_id"`
	Branch  string `json:"branch"`
}

type Match struct {
	Room       Room   `json:"room"`
	BuildID    string `json:"build_id"`
	Diagnostic string `json:"diagnostic"`
}

func MatchBuild(event BuildEvent, rooms []Room) (Match, error) {
	available := make([]Room, 0, len(rooms))
	for _, room := range rooms {
		if room.Participants < room.MaxParticipants {
			available = append(available, room)
		}
	}
	if len(available) == 0 {
		return Match{}, fmt.Errorf("no room capacity for build %s", event.BuildID)
	}
	sort.Slice(available, func(i, j int) bool {
		return available[i].Participants < available[j].Participants
	})
	chosen := available[0]
	return Match{Room: chosen, BuildID: event.BuildID, Diagnostic: "room selected with available capacity"}, nil
}
