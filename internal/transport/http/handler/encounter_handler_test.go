package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
)

func TestCreateEncounterReturnsCreatedEncounter(t *testing.T) {
	handler := newTestEncounterHandler()

	body := `{"combatants":[{"id":"hero-1","name":"Hero","side":"party","current_hp":20,"max_hp":20,"armor_class":15,"initiative":12,"status":"active","effects":[]}]}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/session-1/encounter", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateEncounter(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response["session_id"] != "session-1" {
		t.Fatalf("expected session_id %q, got %v", "session-1", response["session_id"])
	}
}

func TestApplyDamageReturnsUpdatedEncounter(t *testing.T) {
	handler := newTestEncounterHandlerWithEncounter("session-1")

	request := httptest.NewRequest(http.MethodPost, "/sessions/session-1/encounter/damage", strings.NewReader(`{"target_id":"goblin-1","amount":5}`))
	recorder := httptest.NewRecorder()

	handler.ApplyDamage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Combatants []struct {
			ID        string `json:"id"`
			CurrentHP int    `json:"current_hp"`
		} `json:"combatants"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	for _, combatantItem := range response.Combatants {
		if combatantItem.ID == "goblin-1" && combatantItem.CurrentHP != 3 {
			t.Fatalf("expected goblin hp 3, got %d", combatantItem.CurrentHP)
		}
	}
}

func TestCanActReturnsBooleanResult(t *testing.T) {
	handler := newTestEncounterHandlerWithEncounter("session-1")

	request := httptest.NewRequest(http.MethodPost, "/sessions/session-1/encounter/can-act", strings.NewReader(`{"target_id":"hero-1"}`))
	recorder := httptest.NewRecorder()

	handler.CanAct(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		CanAct bool `json:"can_act"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if !response.CanAct {
		t.Fatal("expected hero to be able to act")
	}
}

func TestGetEncounterReturnsNotFoundWhenMissing(t *testing.T) {
	handler := newTestEncounterHandler()

	request := httptest.NewRequest(http.MethodGet, "/sessions/missing/encounter", nil)
	recorder := httptest.NewRecorder()

	handler.GetEncounter(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func newTestEncounterHandler() *EncounterHandler {
	repo := newFakeHTTPEncounterRepository()
	return NewEncounterHandler(service.NewEncounterService(repo))
}

func newTestEncounterHandlerWithEncounter(sessionID string) *EncounterHandler {
	repo := newFakeHTTPEncounterRepository()
	now := time.Date(2026, 4, 5, 16, 30, 0, 0, time.UTC)
	repo.bySessionID[sessionID] = combat.NewEncounter("encounter-1", sessionID, []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}, now)
	return NewEncounterHandler(service.NewEncounterService(repo))
}

type fakeHTTPEncounterRepository struct {
	bySessionID map[string]*combat.Encounter
}

func newFakeHTTPEncounterRepository() *fakeHTTPEncounterRepository {
	return &fakeHTTPEncounterRepository{
		bySessionID: make(map[string]*combat.Encounter),
	}
}

func (f *fakeHTTPEncounterRepository) Save(ctx context.Context, encounter *combat.Encounter) error {
	_ = ctx
	f.bySessionID[encounter.SessionID] = encounter
	return nil
}

func (f *fakeHTTPEncounterRepository) LoadBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	_ = ctx
	encounter, ok := f.bySessionID[sessionID]
	if !ok {
		return nil, repository.ErrEncounterNotFound
	}
	return encounter, nil
}
