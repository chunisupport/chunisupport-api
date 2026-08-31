DELETE FROM chart_best_slot_stats_by_rating_band;
DELETE FROM chart_stats_by_rating_band;
DELETE FROM worldsend_chart_stats_by_rating_band;

DELETE FROM rating_bands
WHERE id = 29;

UPDATE rating_bands
SET label = '17.6+', max_exclusive = NULL
WHERE id = 28;
