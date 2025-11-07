package dns

import (
	"bytes"
	"encoding/json"
	"net/http"
)



func RegisterInDNS(name, ip, apiAddr string) error {
	rec := map[string]interface{}{
		"name":name,
		"ip":ip,
		"ttl":360,
	}

	b,_ := json.Marshal(rec)
	resp, err := http.Post("http://"+apiAddr+"/add", "application/json", bytes.NewReader(b))

	if err != nil {
		return err
	}


	defer resp.Body.Close()
	return nil
}