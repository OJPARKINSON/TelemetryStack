package verification

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type response struct {
	Query     string          `json:"query"`
	Columns   []column        `json:"columns"` // Should be array of column objects
	Timestamp int64           `json:"timestamp"`
	Dataset   [][]interface{} `json:"dataset"` // Mixed types in QuestDB responses
	Count     int             `json:"count"`
}

type column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func GetRecordCount() (int, error) {
	u, err := url.Parse("http://localhost:9000")
	if err != nil {
		return -1, fmt.Errorf("error parsing url, ", err)
	}

	u.Path += "exec"
	params := url.Values{}
	params.Add("query", `
		SELECT count(timestamp) FROM TelemetryTicks
	`)
	u.RawQuery = params.Encode()
	url := fmt.Sprintf("%v", u)

	res, err := http.Get(url)
	if err != nil {
		return -1, fmt.Errorf("error to get stored records, ", err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error: failed ro read body, ", err)
	}

	response := response{}

	if err := json.Unmarshal(body, &response); err != nil {
		return -1, fmt.Errorf("failed to unmarshal: %w, body: %s", err, string(body))
	}

	if len(response.Dataset) == 0 || len(response.Dataset[0]) == 0 {
		log.Printf("empty dataset in response")
	}

	actualCount := int(response.Dataset[0][0].(float64)) // JSON numbers are float64

	log.Printf("✓ Count validation passed: %d records", actualCount)

	return actualCount, nil
}
