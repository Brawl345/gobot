package currency

import "errors"

type Response struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Rates  map[string]float64 `json:"rates"`
}

var (
	BadAmountError    = errors.New("bad amount")
	BadCurrencyError  = errors.New("bad currency")
	SameCurrencyError = errors.New("is same currency")
)
