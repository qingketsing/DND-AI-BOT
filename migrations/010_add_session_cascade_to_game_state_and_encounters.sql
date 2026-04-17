DELETE FROM game_states
WHERE NOT EXISTS (
    SELECT 1 FROM sessions WHERE sessions.id = game_states.session_id
);

DELETE FROM encounters
WHERE NOT EXISTS (
    SELECT 1 FROM sessions WHERE sessions.id = encounters.session_id
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_game_states_session'
    ) THEN
        ALTER TABLE game_states
        ADD CONSTRAINT fk_game_states_session
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_encounters_session'
    ) THEN
        ALTER TABLE encounters
        ADD CONSTRAINT fk_encounters_session
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE;
    END IF;
END $$;
