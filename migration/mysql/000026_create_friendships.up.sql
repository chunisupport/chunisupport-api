CREATE TABLE friendship_statuses (
  id TINYINT UNSIGNED NOT NULL PRIMARY KEY,
  name VARCHAR(20) NOT NULL UNIQUE
);

INSERT INTO friendship_statuses (id, name) VALUES
  (1, 'pending'),
  (2, 'accepted'),
  (3, 'blocked')
ON DUPLICATE KEY UPDATE name = VALUES(name);

CREATE TABLE friendships (
  user_id INT UNSIGNED NOT NULL,
  friend_user_id INT UNSIGNED NOT NULL,
  status_id TINYINT UNSIGNED NOT NULL,
  requested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  accepted_at TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, friend_user_id),
  KEY idx_friendships_friend_user_status (friend_user_id, status_id, requested_at),
  KEY idx_friendships_user_status (user_id, status_id, accepted_at),
  CONSTRAINT fk_friendships_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_friendships_friend_user_id
    FOREIGN KEY (friend_user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_friendships_status_id
    FOREIGN KEY (status_id) REFERENCES friendship_statuses(id),
  CONSTRAINT chk_friendships_not_self CHECK (user_id <> friend_user_id),
  CONSTRAINT chk_friendships_accepted_at CHECK (
    (status_id = 2 AND accepted_at IS NOT NULL)
    OR (status_id <> 2 AND accepted_at IS NULL)
  )
);
