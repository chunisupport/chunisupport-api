INSERT INTO full_chain_types (name, sort_order)
SELECT 'FULL CHAIN RAINBOW', 3
WHERE NOT EXISTS (
    SELECT 1 FROM full_chain_types WHERE name = 'FULL CHAIN RAINBOW'
);
