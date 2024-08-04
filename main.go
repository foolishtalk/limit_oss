package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type ProxyConfig struct {
	RelativePath string `json:"relativePath"`
	Remote       string `json:"remote"`
}

func proxy(c *gin.Context, remote *url.URL) {

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	proxy.Director = func(req *http.Request) {
		fmt.Println(req.URL)
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = remote.Path
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func parseProxyJSON() []ProxyConfig {
	// read local file
	jsonFile, err := os.Open("proxy.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully Opened proxy.json")
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			panic(err)
		}
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)

	var configs []ProxyConfig

	err = json.Unmarshal(byteValue, &configs)
	if err != nil {
		panic(err)
	}
	return configs
}

func main() {
	r := gin.Default()
	// parse proxy
	configs := parseProxyJSON()

	for i := 0; i < len(configs); i++ {
		remote, err := url.Parse(configs[i].Remote)
		if err != nil {
			panic(err)
		}

		r.Any(configs[i].RelativePath, func(c *gin.Context) {
			proxy(c, remote)
		})

	}

	err := r.Run(":9191")
	if err != nil {
		panic(err)
	}
}
