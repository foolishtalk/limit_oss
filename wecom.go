package main

import (
	"bytes"
	"github.com/sirupsen/logrus"
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
		logrus.Info("create request failed:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Info("request failed:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Panic(err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Info("response fail:", err)
		return
	}
	logrus.Info("response:", string(body))
}
