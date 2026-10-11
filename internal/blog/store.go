package blog

import (
	"database/sql"
	"encoding/json"
	"github.com/gofrs/flock"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	_ "modernc.org/sqlite"
)

type Store struct {
	mu     sync.Mutex
	db     *sql.DB
	state  State
	root   string
	active atomic.Pointer[Snapshot]
	lock   *flock.Flock
}

func OpenStore(root string) (*Store, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(root, "coordinator.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, fail("COORDINATOR_RUNNING", "该数据目录已有运行中的发布协调器")
	}
	success := false
	defer func() {
		if !success {
			_ = lock.Unlock()
		}
	}()
	for _, d := range []string{"assets", "releases", "previews", "backups", "restore"} {
		if err = os.MkdirAll(filepath.Join(root, d), 0700); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", filepath.Join(root, "jianxi.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA busy_timeout=5000; PRAGMA secure_delete=ON;
 CREATE TABLE IF NOT EXISTS state(id INTEGER PRIMARY KEY CHECK(id=1), data BLOB NOT NULL);
 CREATE VIRTUAL TABLE IF NOT EXISTS published_search USING fts5(release UNINDEXED, id UNINDEXED, title, body, tokenize='trigram');`)
	if err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db, root: root, lock: lock}
	var b []byte
	err = db.QueryRow("SELECT data FROM state WHERE id=1").Scan(&b)
	if err == sql.ErrNoRows {
		s.state = initialState()
		err = s.save(s.state, nil)
	} else if err == nil {
		err = json.Unmarshal(b, &s.state)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	if s.state.Active != "" {
		snap, err := s.readSnapshot(s.state.Active, s.state.ActiveHash)
		if err != nil {
			db.Close()
			return nil, err
		}
		s.active.Store(&snap)
	}
	success = true
	return s, nil
}
func (s *Store) Close() error {
	err := s.db.Close()
	unlockErr := s.lock.Unlock()
	if err != nil {
		return err
	}
	return unlockErr
}
func (s *Store) Read() State { s.mu.Lock(); defer s.mu.Unlock(); return clone(s.state) }

// Mutations are copy-on-write. Failed validation or persistence never leaks partial state.
func (s *Store) Update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.state)
	if err := fn(&next); err != nil {
		return err
	}
	next.Generation++
	if err := s.save(next, nil); err != nil {
		return err
	}
	s.state = next
	return nil
}
func (s *Store) save(st State, snap *Snapshot) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO state(id,data) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", b); err != nil {
		return err
	}
	if snap != nil {
		if _, err = tx.Exec("DELETE FROM published_search"); err != nil {
			return err
		}
		for id, r := range snap.Posts {
			if r.Content.Kind == "about" {
				continue
			}
			if _, err = tx.Exec("INSERT INTO published_search(release,id,title,body) VALUES(?,?,?,?)", snap.ID, id, r.Content.Title, searchText(r, *snap)); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
func (s *Store) readSnapshot(id, hash string) (Snapshot, error) {
	var snap Snapshot
	dir := filepath.Join(s.root, "releases", id)
	h, err := treeHash(dir)
	if err != nil {
		return snap, err
	}
	if h != hash {
		return snap, fail("ARTIFACT_CORRUPT", "当前发布制品校验失败；请从备份恢复")
	}
	b, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		return snap, err
	}
	err = json.Unmarshal(b, &snap)
	return snap, err
}
