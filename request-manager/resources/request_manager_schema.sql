-- When updating this file, create a new migration file in the
-- /migrations subdirectory to record the changes, and ensure
-- the migrations_test.go tests pass.

CREATE TABLE IF NOT EXISTS `requests` (
  `request_id`     BINARY(20)       NOT NULL,
  `type`           VARBINARY(75)    NOT NULL,
  `state`          TINYINT UNSIGNED NOT NULL DEFAULT 0,
  `user`           VARCHAR(100)         NULL DEFAULT NULL,
  `created_at`     TIMESTAMP(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `started_at`     TIMESTAMP(6)         NULL DEFAULT NULL,
  `finished_at`    TIMESTAMP(6)         NULL DEFAULT NULL,
  `total_jobs`     INT UNSIGNED     NOT NULL DEFAULT 0,
  `finished_jobs`  INT UNSIGNED     NOT NULL DEFAULT 0,
  `jr_url`         VARCHAR(2000)        NULL DEFAULT NULL,

  PRIMARY KEY (`request_id`),
  INDEX (`created_at`),         -- recently created
  INDEX (`finished_at`),        -- recently finished
  INDEX (`state`, `created_at`) -- currently running
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `request_archives` (
  `request_id`      BINARY(20) NOT NULL,
  `create_request`  BLOB       NOT NULL, -- proto.CreateRequest from caller
  `args`            BLOB       NOT NULL, -- finalized request args
  `job_chain`       LONGBLOB   NOT NULL, -- proto.JobChain

  PRIMARY KEY (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `job_log` (
  `request_id`    BINARY(20)       NOT NULL,
  `job_id`        BINARY(4)        NOT NULL,
  `name`          VARBINARY(100)   NOT NULL,
  `try`           SMALLINT         NOT NULL DEFAULT 0,
  `type`          VARBINARY(75)    NOT NULL,
  `state`         TINYINT UNSIGNED NOT NULL DEFAULT 0,
  `started_at`    BIGINT UNSIGNED  NOT NULL DEFAULT 0, -- Unix time (nanoseconds)
  `finished_at`   BIGINT UNSIGNED  NOT NULL DEFAULT 0, -- Unix time (nanoseconds)
  `error`         TEXT                 NULL DEFAULT NULL,
  `exit`          BIGINT               NULL DEFAULT NULL,
  `stdout`        LONGBLOB             NULL DEFAULT NULL,
  `stderr`        LONGBLOB             NULL DEFAULT NULL,

  PRIMARY KEY (`request_id`, `job_id`, `try`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `suspended_job_chains` (
  `request_id`          BINARY(20)    NOT NULL,
  `suspended_job_chain` LONGBLOB      NOT NULL,
  `rm_host`             VARCHAR(64)       NULL DEFAULT NULL,
  `updated_at`          TIMESTAMP(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `suspended_at`        TIMESTAMP(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

  PRIMARY KEY (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
