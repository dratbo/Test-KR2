ALTER TABLE schematics ADD COLUMN IF NOT EXISTS hub_tier INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS recipe_unlock_tiers (
    recipe_class_name TEXT PRIMARY KEY,
    hub_tier INT NOT NULL
);

CREATE TABLE IF NOT EXISTS building_unlock_tiers (
    building_class_name TEXT PRIMARY KEY,
    hub_tier INT NOT NULL
);
