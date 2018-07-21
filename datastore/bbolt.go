package datastore

import (
	"log"
	"strconv"

	bolt "github.com/coreos/bbolt"
)

// Store represents the data store
type Store struct {
	db *bolt.DB
}

// New returns a new instance of the Store
func New(file string) (*Store, error) {
	s := new(Store)
	db, err := bolt.Open(file, 0600, nil)
	s.db = db
	return s, err
}

// Close releases the lock on the data store
func (ds *Store) Close() error {
	return ds.db.Close()
}

// Inc increments the score for a user
func (ds *Store) Inc(team, user string) (int, error) {
	return ds.update(1, team, user)
}

// Dec decrements the score for a user
func (ds *Store) Dec(team, user string) (int, error) {
	return ds.update(-1, team, user)
}

func (ds *Store) update(delta int, team, user string) (int, error) {
	var score int
	err := ds.db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucketIfNotExists([]byte(team))
		if err != nil {
			return err
		}

		bs := b.Get([]byte(user))
		if bs == nil {
			log.Println("no previous score for user:", user)
		} else {
			score, err = strconv.Atoi(string(bs))
			if err != nil {
				log.Println("unable to read previous score for user:", user)
				return err
			}
		}

		score += delta
		log.Println("new score for:", user, score)

		scoreStr := strconv.Itoa(score)

		return b.Put([]byte(user), []byte(scoreStr))
	})

	return score, err
}
