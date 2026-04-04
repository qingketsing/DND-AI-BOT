package composite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
	goredis "github.com/redis/go-redis/v9"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestIntegrationSessionRepositoryPersistsToPGAndBackfillsRedis(t *testing.T) {
	ctx := context.Background()
	deps := openIntegrationDependencies(t)
	defer deps.close()
	resetIntegrationState(t, ctx, deps.db, deps.redis)

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-int-1", model.ChannelWeb, now)
	session.AppendSystemMessage("welcome", now.Add(time.Minute))
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Alice"}, "hello", now.Add(2*time.Minute))
	session.AppendAgentMessage(model.User{ID: "agent-1", Name: "DM Agent"}, "reply", now.Add(3*time.Minute))

	repo := NewCompositeSessionRepository(
		postgresstore.NewPGSessionStore(deps.db),
		rediscache.NewRedisSessionCache(deps.redis),
		newIntegrationCachePolicy(),
	)

	if err := repo.Save(ctx, session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	pgStore := postgresstore.NewPGSessionStore(deps.db)
	persisted, err := pgStore.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("expected PG load to succeed, got %v", err)
	}
	if len(persisted.History) != 3 {
		t.Fatalf("expected 3 history records in PG, got %d", len(persisted.History))
	}

	if exists := redisKeyExists(t, ctx, deps.redis, "session:"+session.ID); exists {
		t.Fatal("expected cache key to be absent immediately after save")
	}

	loaded, err := repo.Load(ctx, session.ID)
	if err != nil {
		t.Fatalf("expected composite load to succeed, got %v", err)
	}
	if loaded.ID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, loaded.ID)
	}
	if !redisKeyExists(t, ctx, deps.redis, "session:"+session.ID) {
		t.Fatal("expected redis cache to be backfilled after load")
	}
}

func TestIntegrationSessionRepositoryUpdatesPGAndRefreshesRedisAfterReload(t *testing.T) {
	ctx := context.Background()
	deps := openIntegrationDependencies(t)
	defer deps.close()
	resetIntegrationState(t, ctx, deps.db, deps.redis)

	now := time.Date(2026, 4, 4, 13, 0, 0, 0, time.UTC)
	session := model.NewSession("session-int-update-1", model.ChannelBot, now)
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Alice"}, "first", now.Add(time.Minute))

	repo := NewCompositeSessionRepository(
		postgresstore.NewPGSessionStore(deps.db),
		rediscache.NewRedisSessionCache(deps.redis),
		newIntegrationCachePolicy(),
	)

	if err := repo.Save(ctx, session); err != nil {
		t.Fatalf("expected initial save to succeed, got %v", err)
	}
	if _, err := repo.Load(ctx, session.ID); err != nil {
		t.Fatalf("expected initial load to succeed, got %v", err)
	}

	session.AppendAgentMessage(model.User{ID: "agent-1", Name: "DM Agent"}, "second", now.Add(2*time.Minute))
	if err := repo.Save(ctx, session); err != nil {
		t.Fatalf("expected update save to succeed, got %v", err)
	}

	if exists := redisKeyExists(t, ctx, deps.redis, "session:"+session.ID); exists {
		t.Fatal("expected cache key to be removed after update save")
	}

	pgStore := postgresstore.NewPGSessionStore(deps.db)
	persisted, err := pgStore.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("expected PG load to succeed, got %v", err)
	}
	if len(persisted.History) != 2 {
		t.Fatalf("expected 2 history records in PG after update, got %d", len(persisted.History))
	}
	if persisted.History[1].Message.Content != "second" {
		t.Fatalf("expected latest PG message to be %q, got %q", "second", persisted.History[1].Message.Content)
	}

	reloaded, err := repo.Load(ctx, session.ID)
	if err != nil {
		t.Fatalf("expected reload to succeed, got %v", err)
	}
	if len(reloaded.History) != 2 {
		t.Fatalf("expected 2 history records after reload, got %d", len(reloaded.History))
	}
	if reloaded.History[1].Message.Content != "second" {
		t.Fatalf("expected latest cached message to be %q, got %q", "second", reloaded.History[1].Message.Content)
	}
	if !redisKeyExists(t, ctx, deps.redis, "session:"+session.ID) {
		t.Fatal("expected redis cache to be backfilled after reload")
	}
}

func TestIntegrationGameStateRepositoryPersistsToPGAndBackfillsRedis(t *testing.T) {
	ctx := context.Background()
	deps := openIntegrationDependencies(t)
	defer deps.close()
	resetIntegrationState(t, ctx, deps.db, deps.redis)

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	player := state.PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
		Inventory: []state.InventoryItem{
			{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 2},
		},
		Quests: []state.QuestProgress{
			{ID: "quest-1", Title: "Find Key", Status: state.QuestStatusActive},
		},
	}
	gameState := state.NewGameState("state-int-1", "session-int-2", player, now)
	gameState.SetCurrentScene("tavern", now.Add(time.Minute))

	repo := NewCompositeGameStateRepository(
		postgresstore.NewPGGameStateStore(deps.db),
		rediscache.NewRedisGameStateCache(deps.redis),
		newIntegrationCachePolicy(),
	)

	if err := repo.Save(ctx, gameState); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	pgStore := postgresstore.NewPGGameStateStore(deps.db)
	persisted, err := pgStore.GetGameStateBySessionID(ctx, gameState.SessionID)
	if err != nil {
		t.Fatalf("expected PG load to succeed, got %v", err)
	}
	if persisted.CurrentScene != "tavern" {
		t.Fatalf("expected current scene tavern, got %q", persisted.CurrentScene)
	}

	if exists := redisKeyExists(t, ctx, deps.redis, "game_state:"+gameState.SessionID); exists {
		t.Fatal("expected cache key to be absent immediately after save")
	}

	loaded, err := repo.LoadBySessionID(ctx, gameState.SessionID)
	if err != nil {
		t.Fatalf("expected composite load to succeed, got %v", err)
	}
	if loaded.ID != gameState.ID {
		t.Fatalf("expected game state id %q, got %q", gameState.ID, loaded.ID)
	}
	if !redisKeyExists(t, ctx, deps.redis, "game_state:"+gameState.SessionID) {
		t.Fatal("expected redis cache to be backfilled after load")
	}
}

func TestIntegrationGameStateRepositoryUpdatesPGAndRefreshesRedisAfterReload(t *testing.T) {
	ctx := context.Background()
	deps := openIntegrationDependencies(t)
	defer deps.close()
	resetIntegrationState(t, ctx, deps.db, deps.redis)

	now := time.Date(2026, 4, 4, 13, 0, 0, 0, time.UTC)
	player := state.PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
	}
	gameState := state.NewGameState("state-int-update-1", "session-int-update-2", player, now)
	gameState.SetCurrentScene("tavern", now.Add(time.Minute))

	repo := NewCompositeGameStateRepository(
		postgresstore.NewPGGameStateStore(deps.db),
		rediscache.NewRedisGameStateCache(deps.redis),
		newIntegrationCachePolicy(),
	)

	if err := repo.Save(ctx, gameState); err != nil {
		t.Fatalf("expected initial save to succeed, got %v", err)
	}
	if _, err := repo.LoadBySessionID(ctx, gameState.SessionID); err != nil {
		t.Fatalf("expected initial load to succeed, got %v", err)
	}

	gameState.AddGold(25, now.Add(2*time.Minute))
	gameState.AddItem(state.InventoryItem{ID: "inv-2", ItemID: "rope", Name: "Rope", Quantity: 1}, now.Add(3*time.Minute))
	gameState.SetCurrentScene("forest", now.Add(4*time.Minute))
	if err := repo.Save(ctx, gameState); err != nil {
		t.Fatalf("expected update save to succeed, got %v", err)
	}

	if exists := redisKeyExists(t, ctx, deps.redis, "game_state:"+gameState.SessionID); exists {
		t.Fatal("expected cache key to be removed after update save")
	}

	pgStore := postgresstore.NewPGGameStateStore(deps.db)
	persisted, err := pgStore.GetGameStateBySessionID(ctx, gameState.SessionID)
	if err != nil {
		t.Fatalf("expected PG load to succeed, got %v", err)
	}
	if persisted.CurrentScene != "forest" {
		t.Fatalf("expected current scene %q, got %q", "forest", persisted.CurrentScene)
	}
	if persisted.Player.Gold != 35 {
		t.Fatalf("expected gold 35, got %d", persisted.Player.Gold)
	}
	if len(persisted.Player.Inventory) != 1 || persisted.Player.Inventory[0].ItemID != "rope" {
		t.Fatalf("expected updated inventory to contain rope, got %+v", persisted.Player.Inventory)
	}

	reloaded, err := repo.LoadBySessionID(ctx, gameState.SessionID)
	if err != nil {
		t.Fatalf("expected reload to succeed, got %v", err)
	}
	if reloaded.CurrentScene != "forest" || reloaded.Player.Gold != 35 {
		t.Fatalf("expected cached state to reflect updates, got scene=%q gold=%d", reloaded.CurrentScene, reloaded.Player.Gold)
	}
	if !redisKeyExists(t, ctx, deps.redis, "game_state:"+gameState.SessionID) {
		t.Fatal("expected redis cache to be backfilled after reload")
	}
}

func TestIntegrationEncounterRepositoryPersistsToPGAndBackfillsRedis(t *testing.T) {
	ctx := context.Background()
	deps := openIntegrationDependencies(t)
	defer deps.close()
	resetIntegrationState(t, ctx, deps.db, deps.redis)

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	combatants := []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}
	encounter := combat.NewEncounter("encounter-int-1", "session-int-3", combatants, now)
	target, err := encounter.FindCombatant("hero-1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	target.AddEffect(combat.StatusEffect{ID: "effect-1", Type: combat.EffectStunned, Source: "spell", Duration: 1})

	repo := NewCompositeEncounterRepository(
		postgresstore.NewPGEncounterStore(deps.db),
		rediscache.NewRedisEncounterCache(deps.redis),
		newIntegrationCachePolicy(),
	)

	if err := repo.Save(ctx, encounter); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	pgStore := postgresstore.NewPGEncounterStore(deps.db)
	persisted, err := pgStore.GetEncounterBySessionID(ctx, encounter.SessionID)
	if err != nil {
		t.Fatalf("expected PG load to succeed, got %v", err)
	}
	if len(persisted.Combatants) != 2 {
		t.Fatalf("expected 2 combatants, got %d", len(persisted.Combatants))
	}

	if exists := redisKeyExists(t, ctx, deps.redis, "encounter:"+encounter.SessionID); exists {
		t.Fatal("expected cache key to be absent immediately after save")
	}

	loaded, err := repo.LoadBySessionID(ctx, encounter.SessionID)
	if err != nil {
		t.Fatalf("expected composite load to succeed, got %v", err)
	}
	if loaded.ID != encounter.ID {
		t.Fatalf("expected encounter id %q, got %q", encounter.ID, loaded.ID)
	}
	if !redisKeyExists(t, ctx, deps.redis, "encounter:"+encounter.SessionID) {
		t.Fatal("expected redis cache to be backfilled after load")
	}
}

func TestIntegrationEncounterRepositoryUpdatesPGAndRefreshesRedisAfterReload(t *testing.T) {
	ctx := context.Background()
	deps := openIntegrationDependencies(t)
	defer deps.close()
	resetIntegrationState(t, ctx, deps.db, deps.redis)

	now := time.Date(2026, 4, 4, 13, 0, 0, 0, time.UTC)
	combatants := []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}
	encounter := combat.NewEncounter("encounter-int-update-1", "session-int-update-3", combatants, now)

	repo := NewCompositeEncounterRepository(
		postgresstore.NewPGEncounterStore(deps.db),
		rediscache.NewRedisEncounterCache(deps.redis),
		newIntegrationCachePolicy(),
	)

	if err := repo.Save(ctx, encounter); err != nil {
		t.Fatalf("expected initial save to succeed, got %v", err)
	}
	if _, err := repo.LoadBySessionID(ctx, encounter.SessionID); err != nil {
		t.Fatalf("expected initial load to succeed, got %v", err)
	}

	if err := encounter.ApplyDamage("goblin-1", 5, now.Add(time.Minute)); err != nil {
		t.Fatalf("expected apply damage to succeed, got %v", err)
	}
	encounter.AdvanceTurn(now.Add(2 * time.Minute))
	target, err := encounter.FindCombatant("hero-1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	target.AddEffect(combat.StatusEffect{ID: "effect-2", Type: combat.EffectStunned, Source: "trap", Duration: 2})

	if err := repo.Save(ctx, encounter); err != nil {
		t.Fatalf("expected update save to succeed, got %v", err)
	}

	if exists := redisKeyExists(t, ctx, deps.redis, "encounter:"+encounter.SessionID); exists {
		t.Fatal("expected cache key to be removed after update save")
	}

	pgStore := postgresstore.NewPGEncounterStore(deps.db)
	persisted, err := pgStore.GetEncounterBySessionID(ctx, encounter.SessionID)
	if err != nil {
		t.Fatalf("expected PG load to succeed, got %v", err)
	}
	if persisted.TurnIndex != 1 {
		t.Fatalf("expected turn index 1 after update, got %d", persisted.TurnIndex)
	}
	goblin, err := persisted.FindCombatant("goblin-1")
	if err != nil {
		t.Fatalf("expected goblin to exist, got %v", err)
	}
	if goblin.CurrentHP != 3 {
		t.Fatalf("expected goblin HP 3 after update, got %d", goblin.CurrentHP)
	}
	hero, err := persisted.FindCombatant("hero-1")
	if err != nil {
		t.Fatalf("expected hero to exist, got %v", err)
	}
	if !hero.HasEffect(combat.EffectStunned) {
		t.Fatal("expected hero to have stunned effect after update")
	}

	reloaded, err := repo.LoadBySessionID(ctx, encounter.SessionID)
	if err != nil {
		t.Fatalf("expected reload to succeed, got %v", err)
	}
	if reloaded.TurnIndex != 1 {
		t.Fatalf("expected cached encounter turn index 1, got %d", reloaded.TurnIndex)
	}
	if !redisKeyExists(t, ctx, deps.redis, "encounter:"+encounter.SessionID) {
		t.Fatal("expected redis cache to be backfilled after reload")
	}
}

type integrationDependencies struct {
	db    *sql.DB
	redis *goredis.Client
}

func openIntegrationDependencies(t *testing.T) integrationDependencies {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("integration test disabled; set INTEGRATION_TEST=1 to enable")
	}

	postgresDSN := os.Getenv("POSTGRES_TEST_DSN")
	if postgresDSN == "" {
		postgresDSN = "postgres://dnd:dndpass@127.0.0.1:5432/dndbot?sslmode=disable"
	}
	redisAddr := os.Getenv("REDIS_TEST_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	db, err := sql.Open("pgx", postgresDSN)
	if err != nil {
		t.Fatalf("expected postgres open to succeed, got %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("expected postgres ping to succeed, got %v", err)
	}

	client := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("expected redis ping to succeed, got %v", err)
	}

	applyMigrations(t, context.Background(), db)

	return integrationDependencies{db: db, redis: client}
}

func (d integrationDependencies) close() {
	_ = d.redis.Close()
	_ = d.db.Close()
}

func applyMigrations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	files := []string{
		"001_create_sessions.sql",
		"002_create_session_messages.sql",
		"003_create_game_states.sql",
		"004_create_encounters.sql",
	}

	for _, file := range files {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", file))
		if err != nil {
			t.Fatalf("expected migration %s to be readable, got %v", file, err)
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			t.Fatalf("expected migration %s to apply, got %v", file, err)
		}
	}
}

func resetIntegrationState(t *testing.T, ctx context.Context, db *sql.DB, client *goredis.Client) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		TRUNCATE TABLE
			session_messages,
			sessions,
			game_states,
			encounters
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("expected database cleanup to succeed, got %v", err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("expected redis cleanup to succeed, got %v", err)
	}
}

func redisKeyExists(t *testing.T, ctx context.Context, client *goredis.Client, key string) bool {
	t.Helper()

	count, err := client.Exists(ctx, key).Result()
	if err != nil {
		t.Fatalf("expected redis exists to succeed, got %v", err)
	}
	return count == 1
}

func newIntegrationCachePolicy() CachePolicy {
	return CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
		TTLJitter:   5 * time.Second,
	}
}
