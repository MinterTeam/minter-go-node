package utils

import (
	"github.com/syndtr/goleveldb/leveldb/opt"
	db "github.com/tendermint/tm-db"
	"os"
	"path/filepath"
)

type Storage struct {
	minterHome   string
	minterConfig string
	eventDB      db.DB
	stateDB      db.DB
	snapshotDB   db.DB
}

func (s *Storage) SetMinterConfig(minterConfig string) {
	s.minterConfig = minterConfig
}

func (s *Storage) SetMinterHome(minterHome string) {
	s.minterHome = minterHome
}

func (s *Storage) EventDB() db.DB {
	return s.eventDB
}

func (s *Storage) StateDB() db.DB {
	return s.stateDB
}

func NewStorage(home string, config string) *Storage {
	return &Storage{eventDB: db.NewMemDB(), stateDB: db.NewMemDB(), snapshotDB: db.NewMemDB(), minterConfig: config, minterHome: home}
}

func (s *Storage) InitSnapshotLevelDB(name string, opts *opt.Options) (db.DB, error) {
	levelDB, err := db.NewGoLevelDBWithOpts(name, s.GetMinterHome(), opts)
	if err != nil {
		return nil, err
	}
	s.snapshotDB = levelDB
	return s.snapshotDB, nil
}

func (s *Storage) InitEventLevelDB(name string, opts *opt.Options) (db.DB, error) {
	levelDB, err := db.NewGoLevelDBWithOpts(name, s.GetMinterHome(), opts)
	if err != nil {
		return nil, err
	}
	s.eventDB = levelDB
	return s.eventDB, nil
}

func (s *Storage) InitStateLevelDB(name string, opts *opt.Options) (db.DB, error) {
	levelDB, err := db.NewGoLevelDBWithOpts(name, s.GetMinterHome(), opts)
	if err != nil {
		return nil, err
	}
	s.stateDB = levelDB
	return s.stateDB, nil
}

func (s *Storage) GetMinterHome() string {
	if s.minterHome != "" {
		return s.minterHome
	}

	s.minterHome = os.Getenv("MINTERHOME")

	if s.minterHome != "" {
		return s.minterHome
	}

	s.minterHome = os.ExpandEnv(filepath.Join("$HOME", ".minter"))
	return s.minterHome
}

func (s *Storage) GetMinterConfigPath() string {
	if s.minterConfig != "" {
		return s.minterConfig
	}

	s.minterConfig = s.GetMinterHome() + "/config/config.toml"
	return s.minterConfig
}
