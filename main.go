package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/takama/daemon"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var interceptHandler = new(InterceptHandler)
var interceptMutex sync.Mutex
var config PolicyConfig

func shouldBlockRequest() PolicyResult {

	currentDay := time.Now().Format("2006-01-02")
	if interceptHandler.todayRequest == "" || currentDay != interceptHandler.todayRequest {
		interceptHandler.todayRequest = currentDay

	}
	minutePolicy := checkPolicy(&interceptHandler.minute, 60, 5)
	minutePolicy.policyType = minute
	if minutePolicy.result {
		return minutePolicy
	}

	hourPolicy := checkPolicy(&interceptHandler.hour, 3600, 30)
	hourPolicy.policyType = hour
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
		switch policy.policyType {
		case minute:
			wecomNotify("每分钟下载上限告警", config.Wecom_hook_url)
		case hour:
			wecomNotify("每小时下载上限告警", config.Wecom_hook_url)
		}

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

func parseProxyJSON() {
	// read local file
	jsonFile, err := os.Open("proxy.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		logrus.Panic(err)
	}

	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			logrus.Panic(err)
		}
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		logrus.Panic(err)
	}
}

func installDaemon() {
	kind := daemon.SystemDaemon
	switch runtime.GOOS {
	case "darwin":
		kind = daemon.GlobalDaemon
	default:
		kind = daemon.SystemDaemon
	}
	service, err := daemon.New("limit_oss", "limit oss download", kind)
	if err != nil {
		panic(err)
	}
	status, err := service.Install()
	if err != nil {
		logrus.Info("daemon install:" + err.Error())
	}
	logrus.Info("daemon status:" + status)
}

func installLogrus() {
	logrus.SetReportCaller(true)
	// 创建日志文件
	f, err := os.Create("gin.log")
	if err != nil {
		logrus.Info("gin.log create:" + err.Error())
	}

	// 设置 Logrus 输出到文件
	logrus.SetOutput(f)
	// 设置 Gin 使用默认的日志中间件
	gin.DefaultWriter = f
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	installLogrus()
	installDaemon()
	r := gin.Default()
	r.Use(customRecoveryMiddleware())

	// parse proxy
	parseProxyJSON()

	resetPolicy(&interceptHandler.minute)
	resetPolicy(&interceptHandler.hour)

	for i := 0; i < len(config.Proxys); i++ {
		remote, err := url.Parse(config.Proxys[i].Remote)
		if err != nil {
			logrus.Panic(err)
		}

		r.Any(config.Proxys[i].RelativePath, func(c *gin.Context) {
			proxy(c, remote)
		})

	}

	err := r.Run(":9191")
	if err != nil {
		panic(err)
	}
}

func customRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if p := recover(); p != nil {
				if err, ok := p.(error); ok {
					// ignore panic abort handler for text/event-stream SSE
					if errors.Is(err, http.ErrAbortHandler) {
						logrus.Info("return abort:" + err.Error())
						return
					}
				}
				logrus.Info("abort with status")
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
