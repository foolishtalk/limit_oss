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
	"strconv"
	"sync"
	"time"
)

type ProxyConfig struct {
	RelativePath string `json:"relativePath"`
	Remote       string `json:"remote"`
}

type InterceptHandler struct {
	todayRequest string
	minute       InterceptPolicy
	hour         InterceptPolicy
}

type InterceptPolicy struct {
	lastTime int64
	count    int64
}

type PolicyResult struct {
	result  bool
	expired int64
}

var interceptHandler = new(InterceptHandler)
var interceptMutex sync.Mutex

func shouldBlockRequest() PolicyResult {

	currentDay := time.Now().Format("2006-01-02")
	if interceptHandler.todayRequest == "" || currentDay != interceptHandler.todayRequest {
		interceptHandler.todayRequest = currentDay

	}
	minutePolicy := checkPolicy(&interceptHandler.minute, 60, 5)
	if minutePolicy.result {
		return minutePolicy
	}

	hourPolicy := checkPolicy(&interceptHandler.hour, 3600, 30)
	if hourPolicy.result {
		return hourPolicy
	}
	return minutePolicy
}

func resetPolicy(policy *InterceptPolicy) {
	interceptMutex.Lock()
	policy.lastTime = time.Now().Unix()
	policy.count = 0
	interceptMutex.Unlock()
}

func checkPolicy(policy *InterceptPolicy, second int64, maxCount int64) PolicyResult {
	expired := second - (time.Now().Unix() - policy.lastTime)
	if expired > 0 {
		if policy.count < maxCount {
			interceptMutex.Lock()
			policy.count++
			interceptMutex.Unlock()
		} else {
			return PolicyResult{result: true, expired: expired}
		}
	} else {
		resetPolicy(policy)
	}
	return PolicyResult{result: false, expired: 0}
}

func proxy(c *gin.Context, remote *url.URL) {
	// get current time
	policy := shouldBlockRequest()
	if policy.result {
		c.JSON(200, gin.H{
			"code":   0,
			"cn_msg": "目前下载的人数太多，请稍等" + strconv.FormatInt(policy.expired, 10) + "秒后再试",
			"en_msg": "Currently, there are too many people downloading. Please wait for" + strconv.FormatInt(policy.expired, 10) + " seconds and try again.",
		})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	proxy.Director = func(req *http.Request) {
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

	resetPolicy(&interceptHandler.minute)
	resetPolicy(&interceptHandler.hour)

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
