package db

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"

	"PhoenixOracle/build/static"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/nodeversion"
	"PhoenixOracle/lib/timerbackup"
	"github.com/coreos/go-semver/semver"

	"PhoenixOracle/db/config"
	"PhoenixOracle/db/migrate"
	"PhoenixOracle/db/models"
	"PhoenixOracle/db/orm"
	"PhoenixOracle/lib/gracefulpanic"
	"PhoenixOracle/lib/postgres"
	"PhoenixOracle/util"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gorm.io/gorm"
)

const (
	AutoMigrate = "auto_migrate"
)

type Store struct {
	*orm.ORM
	Config         config.GeneralConfig
	Clock          utils.AfterNower
	AdvisoryLocker postgres.AdvisoryLocker
	closeOnce      *sync.Once
}

func NewStore(config config.GeneralConfig, advisoryLock postgres.AdvisoryLocker, shutdownSignal gracefulpanic.Signal) (*Store, error) {
	return newStore(config, advisoryLock, shutdownSignal)
}

func NewInsecureStore(config config.GeneralConfig, advisoryLocker postgres.AdvisoryLocker, shutdownSignal gracefulpanic.Signal) (*Store, error) {
	return newStore(config, advisoryLocker, shutdownSignal)
}

func newStore(
	config config.GeneralConfig,
	advisoryLocker postgres.AdvisoryLocker,
	shutdownSignal gracefulpanic.Signal,
) (*Store, error) {
	if err := utils.EnsureDirAndMaxPerms(config.RootDir(), os.FileMode(0700)); err != nil {
		return nil, errors.Wrap(err, "error while creating project root dir")
	}

	orm, err := initializeORM(config, shutdownSignal)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize ORM")
	}

	store := &Store{
		Clock:          utils.Clock{},
		AdvisoryLocker: advisoryLocker,
		Config:         config,
		ORM:            orm,
		closeOnce:      &sync.Once{},
	}
	return store, nil
}

func (s *Store) Start() error {
	return nil
}

func (s *Store) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.ORM.Close()
		err = multierr.Append(err, s.AdvisoryLocker.Close())
	})
	return err
}

func (s *Store) Ready() error {
	return nil
}

func (s *Store) Healthy() error {
	return nil
}

func (s *Store) Unscoped() *Store {
	cpy := *s
	cpy.ORM = s.ORM.Unscoped()
	return &cpy
}

func (s *Store) AuthorizedUserWithSession(sessionID string) (models.User, error) {
	return s.ORM.AuthorizedUserWithSession(
		sessionID, s.Config.SessionTimeout().Duration())
}

func CheckSquashUpgrade(db *gorm.DB) error {
	if static.Version == "unset" {
		return nil
	}
	squashVersionMinus1 := semver.New("0.9.10")
	currentVersion, err := semver.NewVersion(static.Version)
	if err != nil {
		return errors.Wrapf(err, "expected VERSION to be valid semver (for example 1.42.3). Got: %s", static.Version)
	}
	lastV1Migration := "1611847145"
	if squashVersionMinus1.LessThan(*currentVersion) {
		if !db.Migrator().HasTable("migrations") {
			return nil
		}
		q := db.Exec("SELECT * FROM migrations WHERE id = ?", lastV1Migration)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return errors.Errorf("Need to upgrade to phoenix version %v first before upgrading to version %v in order to run migrations", squashVersionMinus1, currentVersion)
		}
	}
	return nil
}

func initializeORM(cfg config.GeneralConfig, shutdownSignal gracefulpanic.Signal) (*orm.ORM, error) {
	dbURL := cfg.DatabaseURL()
	dbOrm, err := orm.NewORM(dbURL.String(), cfg.DatabaseTimeout(), shutdownSignal, cfg.GetDatabaseDialectConfiguredOrDefault(), cfg.GetAdvisoryLockIDConfiguredOrDefault(), cfg.GlobalLockRetryInterval().Duration(), cfg.ORMMaxOpenConns(), cfg.ORMMaxIdleConns())
	if err != nil {
		return nil, errors.Wrap(err, "initializeORM#NewORM")
	}

	verORM := nodeversion.NewORM(postgres.WrapDbWithSqlx(
		postgres.MustSQLDB(dbOrm.DB)),
	)

	if cfg.DatabaseBackupMode() != config.DatabaseBackupModeNone {
		var version *nodeversion.NodeVersion
		var versionString string

		version, err = verORM.FindLatestNodeVersion()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Default.Debugf("Failed to find any node version in the DB: %w", err)
			} else if strings.Contains(err.Error(), "relation \"node_versions\" does not exist") {
				logger.Default.Debugf("Failed to find any node version in the DB, the node_versions table does not exist yet: %w", err)
			} else {
				return nil, errors.Wrap(err, "initializeORM#FindLatestNodeVersion")
			}
		}

		if version != nil {
			versionString = version.Version
		}

		databaseBackup := timerbackup.NewDatabaseBackup(cfg, logger.Default)
		databaseBackup.RunBackupGracefully(versionString)
	}
	if err = CheckSquashUpgrade(dbOrm.DB); err != nil {
		panic(err)
	}
	if cfg.MigrateDatabase() {
		dbOrm.SetLogging(cfg.LogSQLStatements() || cfg.LogSQLMigrations())

		err = dbOrm.RawDBWithAdvisoryLock(func(db *gorm.DB) error {
			return migrate.Migrate(postgres.UnwrapGormDB(db).DB)
		})
		if err != nil {
			return nil, errors.Wrap(err, "initializeORM#Migrate")
		}
	}

	nodeVersion := static.Version
	if nodeVersion == "unset" {
		nodeVersion = fmt.Sprintf("random_%d", rand.Uint32())
	}
	version := nodeversion.NewNodeVersion(nodeVersion)
	err = verORM.UpsertNodeVersion(version)
	if err != nil {
		return nil, errors.Wrap(err, "initializeORM#UpsertNodeVersion")
	}

	dbOrm.SetLogging(cfg.LogSQLStatements())
	return dbOrm, nil
}
