// Package xrate delivers a service for exchange rates' conversion
// with intermediate caching in LevelDB.
package xrate

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// Database provides access to exchange rates data.
type Database interface {
	// Get retrieves FixerRates object from cache based on provided key.
	Get([]byte) (*FixerRates, error)
	// Set puts new object (slice of bytes) under given key.
	Set([]byte, []byte) error
}

// Service defines exchange rates service.
type Service struct {
	db Database
}

// NewService creates new xrate Service using given database.
func NewService(db Database) *Service {
	return &Service{db: db}
}

// Dec is a custom decimal.Decimal type to overwrite JSON marshaling. Original Decimal type
// is encoding values as strings and we want to use JSON Number type (without quotes).
// We're using 3rd party lib for decimal values as it is not safe to store money values
// in float types.
type Dec struct {
	decimal.Decimal
}

// MarshalJSON implements the json.Marshaler interface.
func (d Dec) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

// RatesMap is a string to Dec map custom type defined for XML marshalling.
type RatesMap map[string]Dec

// MarshalXML implements the xml.Marshaler interface.
func (rm RatesMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	for k, v := range rm {
		t := xml.StartElement{Name: xml.Name{Space: "", Local: k}}
		tokens = append(tokens, t, xml.CharData([]byte(v.String())), xml.EndElement{Name: t.Name})
	}
	tokens = append(tokens, xml.EndElement{Name: start.Name})

	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	return e.Flush()
}

// Rates is an exchange rate table returned to user.
type Rates struct {
	Amount    Dec      `json:"amount"`
	Currency  string   `json:"currency"`
	Converted RatesMap `json:"converted"`
}

// FixerRates represents response from fixer.io with exchange rates.
type FixerRates struct {
	Base  string         `json:"base"`
	Date  string         `json:"date"` // Not using time.Time as fixer.io is not responding with ISO 8601.
	Rates map[string]Dec `json:"rates"`
}

const defaultRoundPlaces = 2

// Convert FixerRates to Rates by multiplicating rates by amount and
// rounding final values to 2 decimal places.
func (fr *FixerRates) convToRates(amount decimal.Decimal, currency string) *Rates {
	r := Rates{Amount: Dec{amount}, Currency: currency}
	r.Converted = make(map[string]Dec)
	for k, v := range fr.Rates {
		r.Converted[k] = Dec{v.Mul(amount).Round(defaultRoundPlaces)}
	}
	return &r
}

// Validation errors for amount and currency.
var (
	ErrParseAmount = errors.New("Invalid amount value.")
	ErrEmptyCurr   = errors.New("Currency must not be empty.")
)

// Rates return exchange rates for given amount and currency. It tries to fetch
// from database first. If it failes, it uses fixer.io.
func (s *Service) Rates(amount, currency string) (*Rates, error) {
	if currency == "" {
		return nil, ErrEmptyCurr
	}
	am, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, ErrParseAmount
	}

	// Get from cache or fixer.io.
	key := cacheKey(currency)
	fr, err := s.db.Get(key)
	if err != nil {
		fr, err = s.fetchRates(currency, key)
	}

	return fr.convToRates(am, currency), nil
}

// Constructs cache key under which we store cached rates in LevelDB.
// Keys will rotate at midnight which could be improved as fixer.io
// is updating their tables around 4pm.
func cacheKey(currency string) []byte {
	key := bytes.Buffer{}
	key.WriteString(time.Now().Format("2006-01-02"))
	key.WriteString(currency)
	return key.Bytes()
}

// Fetch rates from fixer.io and store the results in LevelDB.
func (s *Service) fetchRates(currency string, key []byte) (*FixerRates, error) {
	resp, err := http.Get("http://api.fixer.io/latest?base=" + currency)
	if err != nil {
		return nil, fmt.Errorf("can't request from fixer.io: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body from fixer.io: %v", err)
	}
	fr := new(FixerRates)
	if err := json.Unmarshal(body, fr); err != nil {
		return nil, fmt.Errorf("can't parse response from fixer.io: %v", err)
	}
	s.db.Set(key, body)
	return fr, nil
}
