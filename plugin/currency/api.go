package currency

import "errors"

type Response struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Rates  map[string]float64 `json:"rates"`
}

var (
	ErrBadAmount    = errors.New("bad amount")
	ErrBadCurrency  = errors.New("bad currency")
	ErrSameCurrency = errors.New("is same currency")
)
