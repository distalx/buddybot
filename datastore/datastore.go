package datastore

// Scorer defines the methods required to implement the data store
type Scorer interface {
	Inc(team, user string) (int, error)
	Dec(team, user string) (int, error)
	Close() error
}
