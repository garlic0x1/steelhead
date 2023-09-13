package steelhead

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
)

func ParseInput[T any](r *http.Request, to *T) error {
	if r.Method == "POST" || r.Method == "PUT" {
		decoder := json.NewDecoder(r.Body)
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
		Limit: lim,
		Offset: off,
	}, nil
}

func handleResponse(w http.ResponseWriter, status int, body any, err error) {
	// for shorthand returning errs
	if err != nil && status == 200 {
		status = 500
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err != nil {
		// show errors on dev, not on prod
		// if config.Cfg.Dev.TestMode {
			json.NewEncoder(w).Encode(HttpError{status, err.Error()})
		// } else {
		// 	json.NewEncoder(w).Encode(HttpError{status, "Something went wrong :("})
		// }
		return
	}

	// handle returning sqlx data
	page, ok := body.(Page); if ok {
		rows, ok := page.Data.(*sqlx.Rows); if ok {
			page.Data = DbToJson(rows)
			body = page
		}
	}

	rows, ok := body.(*sqlx.Rows); if ok {
		body = DbToJson(rows)
	}

	row, ok := body.(*sqlx.Row); if ok {
		var temp map[string]any
		row.MapScan(temp)
		body = temp
	}

	json.NewEncoder(w).Encode(body)
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
