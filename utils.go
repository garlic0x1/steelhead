package steelhead

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
)

// Build a standard net/http handler, middlewares still apply
func RawHandler(handler func(http.ResponseWriter, *http.Request)) Handler {
	return Handler(func(r *http.Request) (int, any, error) {
		return 200, func(w http.ResponseWriter) { handler(w, r) }, nil
	})
}

func DbToJson(rows *sqlx.Rows) []map[string]interface{} {
	count := 0
	var joined []map[string]interface{}
	for rows.Next() {
		count++
		tmp := make(map[string]interface{})
		rows.MapScan(tmp)
		for k, encoded := range tmp {
			switch encoded.(type) {
			case []byte:
				tmp[k] = string(encoded.([]byte))
				break
			case string:
				tmp[k] = string(encoded.(string))
			}
		}
		joined = append(joined, tmp)
	}
	return joined
}

func ParseInput[T any](r *http.Request, to *T) error {
	if r.Method == "POST" || r.Method == "PUT" {
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err := decoder.Decode(to)
		if err != nil {
			return fmt.Errorf("Failed to decode body")
		}
	} else {
		var ag = make(map[string]any)
		for key, val := range r.URL.Query() {
			if len(val) == 1 {
				ag[key] = val[0]
			} else {
				ag[key] = val
			}
		}
		js, err := json.Marshal(ag)
		if err != nil {
			return fmt.Errorf("Failed to unmarshal URL query")
		}
		err = json.Unmarshal(js, to)
		if err != nil {
			return fmt.Errorf("Failed to unmarshal URL query")
		}
	}
	return nil
}

func ParseQueryLike[T any](r *http.Request, to *T) error {
	var ag = make(map[string]any)
	for key, val := range r.URL.Query() {
		if len(val) == 1 {
			ag[key] = fmt.Sprintf("%%%s%%", val[0])
		} else {
			ag[key] = val
		}
	}
	js, err := json.Marshal(ag)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal URL query")
	}
	err = json.Unmarshal(js, to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal URL query")
	}
	return nil
}

func ExtractPaging(r *http.Request) (PageQuery, error) {
	lim, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return PageQuery{}, fmt.Errorf("Must provide limit and offset in query args")
	}

	off, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		return PageQuery{}, fmt.Errorf("Must provide limit and offset in query args")
	}

	return PageQuery{
		Limit:  lim,
		Offset: off,
	}, nil
}

// Return type magic, basically try to JSONify stuff

func handleResponse(w http.ResponseWriter, status int, body any, err error) {
	// for shorthand returning errs
	if err != nil && status == 200 {
		status = 500
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err != nil {
		// show errors on dev, not on prod
		if Debug {
			json.NewEncoder(w).Encode(HttpError{status, err.Error()})
		} else {
			json.NewEncoder(w).Encode(HttpError{status, "Something went wrong :("})
		}
		return
	}

	switch res := body.(type) {

	// return lambda for low level handlers
	case func(http.ResponseWriter):
		res(w)

	// return paginated JSON
	case Page:
		rows, _ := res.Data.(*sqlx.Rows)
		res.Data = DbToJson(rows)
		json.NewEncoder(w).Encode(res)

	// return sqlx types
	case *sqlx.Rows:
		json.NewEncoder(w).Encode(DbToJson(res))
	case *sqlx.Row:
		var temp map[string]any
		res.MapScan(temp)
		json.NewEncoder(w).Encode(temp)

	// default to JSON encode
	default:
		json.NewEncoder(w).Encode(res)
	}
}

// Helper functions for requests

// Get the body as a string
func DumpBody(r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	return string(body)
}

func DumpRequestInfo(r *http.Request) string {
	method := r.Method
	url := r.URL.String()
	protocol := r.Proto

	headers := ""
	for name, values := range r.Header {
		for _, value := range values {
			headers += fmt.Sprintf("%s: %s\r\n", name, value)
		}
	}

	body := DumpBody(r)

	return fmt.Sprintf(
		"%s %s %s\r\n%s\r\n%s",
		method, url, protocol,
		headers, body,
	)
}
