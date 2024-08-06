package main

type PolicyConfig struct {
	Wecom_hook_url string        `json:"wecom_hook_url"`
	Proxys         []ProxyConfig `json:"proxys"`
}

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
	result     bool
	expired    int64
	policyType PolicyType
}

type PolicyType uint32

const (
	none              = 0
	minute PolicyType = 1
	hour   PolicyType = 2
)
