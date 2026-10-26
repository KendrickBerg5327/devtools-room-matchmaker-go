package main

import "testing"

func TestMatchBuildChoosesRoomWithMostCapacity(t *testing.T) {
	rooms := []Room{
		{Name: "busy", MaxParticipants: 4, Participants: 4},
		{Name: "open", MaxParticipants: 8, Participants: 1},
	}
	got, err := MatchBuild(BuildEvent{BuildID: "build-17", Branch: "main"}, rooms)
	if err != nil {
		t.Fatal(err)
	}
	if got.Room.Name != "open" || got.BuildID != "build-17" {
		t.Fatalf("unexpected match: %+v", got)
	}
}
