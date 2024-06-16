-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT NOT NULL PRIMARY KEY,
  createAt DATETIME NOT NULL,
  updateAt DATETIME NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('pending', 'in_progress', 'completed', 'failed'))
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE jobs
-- +goose StatementEnd
