package notifier

import (
	"net/http"
	"time"

	"github.com/sms-forwarder/internal/config"
)

const defaultRequestTimeout = 10 * time.Second

func requestTimeout(cfg config.ChannelConfig) time.Duration {
	if cfg.RequestTimeoutSec <= 0 {
		return defaultRequestTimeout
	}
	return time.Duration(cfg.RequestTimeoutSec) * time.Second
}

func newHTTPClient(cfg config.ChannelConfig) *http.Client {
	return &http.Client{Timeout: requestTimeout(cfg)}
}
