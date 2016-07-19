package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/viru/xrate/xrate"

	"github.com/syndtr/goleveldb/leveldb"
)

// Custom HTTP handler that can return errors.
type appHandler func(http.ResponseWriter, *http.Request) error

// ServeHTTP calls original handler and handles returned error. If custom
// handlerError is thrown, use its message and HTTP status code. Otherwise
// log original error and return 503 to user without exposing real error.
func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		if herr, ok := err.(handlerError); ok {
			http.Error(w, herr.Error(), herr.code)
		} else {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}

	}
}

// Custom error type for our handlers.
type handlerError struct {
	code int
	err  string
}

// Error satisfies error interface.
func (h handlerError) Error() string {
	return h.err
}

// Exchange conversion handler uses xrate service and renders
// XML or JSON based on Accept header.
func convert(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "GET" {
		return handlerError{code: http.StatusMethodNotAllowed, err: "Method not allowed"}
	}

	rates, err := xrateSrv.Rates(
		strings.TrimSpace(r.URL.Query().Get("amount")),
		strings.TrimSpace(r.URL.Query().Get("currency")),
	)
	if err != nil {
		switch err {
		case xrate.ErrEmptyCurr:
			return handlerError{code: http.StatusBadRequest, err: err.Error()}
		case xrate.ErrParseAmount:
			return handlerError{code: http.StatusBadRequest, err: err.Error()}
		default:
			return fmt.Errorf("Error fetching exchange rates: %v", err)
		}
	}
	if r.Header.Get("Accept") == "application/xml" {
		buf, err := xml.Marshal(rates)
		if err != nil {
			return fmt.Errorf("Error marshaling XML: %v", err)
		}
		w.Header().Set("content-type", "application/xml")
		w.Write([]byte(xml.Header))
		w.Write(buf)
		return nil
	}
	buf, err := json.Marshal(rates)
	if err != nil {
		return fmt.Errorf("Error marshaling JSON: %v", err)
	}
	w.Header().Set("content-type", "application/json")
	w.Write(buf)
	return nil
}

var (
	bindAddr = flag.String("bind", ":8080", "HTTP bind address")
	dbPath   = flag.String("db-path", "/tmp/xrate.db", "LevelDB path")
)

var xrateSrv *xrate.Service

func main() {
	flag.Parse()

	db, err := leveldb.OpenFile(*dbPath, nil)
	if err != nil {
		log.Printf("Can't open LevelDB: %v", err)
		return
	}
	defer db.Close()

	xrateSrv = xrate.NewService(xrate.NewDatabase(db))

	http.Handle("/convert", appHandler(convert))
	log.Printf("Starting HTTP listener on %s", *bindAddr)
	if err := http.ListenAndServe(*bindAddr, nil); err != nil {
		log.Printf("Can't start HTTP listener: %v", err)
	}
}
