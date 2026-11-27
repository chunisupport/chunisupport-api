-- 履歴と目標はFULL CHAINマスタへの外部キーを持たないため、利用済みマスタは明示的に残す。
DELETE fch
FROM full_chain_types AS fch
WHERE fch.name = 'FULL CHAIN RAINBOW'
  AND NOT EXISTS (SELECT 1 FROM player_records WHERE full_chain_id = fch.id)
  AND NOT EXISTS (SELECT 1 FROM player_worldsend_records WHERE full_chain_id = fch.id)
  AND NOT EXISTS (SELECT 1 FROM player_play_logs WHERE full_chain_id = fch.id)
  AND NOT EXISTS (SELECT 1 FROM player_record_histories WHERE full_chain_id = fch.id)
  AND NOT EXISTS (SELECT 1 FROM player_worldsend_record_histories WHERE full_chain_id = fch.id)
  AND NOT EXISTS (
      SELECT 1
      FROM goals AS g
      INNER JOIN achievement_types AS a ON a.id = g.achievement_type_id
      WHERE a.code = 'fullchain_count'
        AND g.achievement_params->>'$.lamp' = 'RAINBOW'
  );
