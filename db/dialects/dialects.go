package dialects


type DialectName string

const (
	Postgres DialectName = "pgx"

	TransactionWrappedPostgres DialectName = "txdb"

	PostgresWithoutLock DialectName = "postgresWithoutLock"
)
