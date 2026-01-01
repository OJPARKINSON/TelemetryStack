package verification

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

func RecordsStored() {
	u, err := url.Parse("http://localhost:9000")
	if err != nil {
		fmt.Println("error parsing url, ", err)
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
		fmt.Println("error to get stored records, ", err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error: failed ro read body, ", err)
	}

	log.Println(string(body))
}
