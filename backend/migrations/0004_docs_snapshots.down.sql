DROP TABLE IF EXISTS doc_snapshots;
DROP TABLE IF EXISTS doc_updates;
DROP TRIGGER IF EXISTS doc_states_tsv_upd ON doc_states;
DROP FUNCTION IF EXISTS doc_states_tsv_trigger();
DROP TABLE IF EXISTS doc_states;
