package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func wecomNotify(content string, url string) {
	payload := []byte(`
    {
      "msgtype": "text",
      "text": {
          "content": "` + content + `"
      }
    }
    `)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("create request failed:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("request failed:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("response fail:", err)
		return
	}
	fmt.Println("response:", string(body))
}
