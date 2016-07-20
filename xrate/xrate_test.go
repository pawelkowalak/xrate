package xrate

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/syndtr/goleveldb/leveldb"
)

type fakeLevelDB struct {
	Database
}

func (f *fakeLevelDB) Get(key []byte) (*FixerRates, error) {
	if string(key) == string(cacheKey("SEK")) {
		return &FixerRates{Base: "SEK"}, nil
	}
	return nil, leveldb.ErrNotFound
}

func (f *fakeLevelDB) Set(key, value []byte) error {
	return nil
}

func TestRatesValidations(t *testing.T) {
	cases := map[string]struct {
		amount, currency string
		rates            *Rates
		err              error
	}{
		"empty currency": {"200", "", nil, ErrEmptyCurr},
		"empty amount":   {"", "SEK", nil, ErrParseAmount},
		"cache hit":      {"200", "SEK", &Rates{Amount: Dec{decimal.New(200, 0)}, Currency: "SEK"}, nil},
	}

	s := NewService(&fakeLevelDB{})

	for k, tc := range cases {
		r, err := s.Rates(tc.amount, tc.currency)
		if err != tc.err {
			t.Errorf("%s: error = %v, expected %v", k, err, tc.err)
		}
		if r != nil && r.Amount.String() != tc.amount {
			t.Errorf("%s: amount = %v, expected %v", k, r.Amount.String(), tc.amount)
		}
		if r != nil && r.Currency != tc.currency {
			t.Errorf("%s: currency = %v, expected %v", k, r.Currency, tc.currency)
		}
	}
}
