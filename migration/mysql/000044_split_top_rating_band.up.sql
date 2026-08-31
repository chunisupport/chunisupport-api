DELETE FROM chart_best_slot_stats_by_rating_band;
DELETE FROM chart_stats_by_rating_band;
DELETE FROM worldsend_chart_stats_by_rating_band;

UPDATE rating_bands
SET label = '17.6', max_exclusive = 17.7
WHERE id = 28;

INSERT INTO rating_bands (id, label, min_inclusive, max_exclusive, sort_order)
VALUES (29, '17.7+', 17.7, NULL, 29);
