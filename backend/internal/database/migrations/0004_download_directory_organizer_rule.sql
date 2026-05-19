ALTER TABLE download_directories ADD COLUMN organizer_rule_id TEXT REFERENCES organizer_rules(id);
