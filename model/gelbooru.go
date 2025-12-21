package model

type GelbooruService interface {
	GetQuery(queryID int64) (string, error)
	SaveQuery(query string) (int64, error)
}
