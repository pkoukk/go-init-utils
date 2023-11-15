package giu

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type RestyParams struct {
	// Timeout is the amount of time to wait for a response.
	Timeout time.Duration
	// RetryTimes is the number of times to retry.
	RetryTimes int
	// DebugMode is the flag to enable/disable debug mode. It will print the request/response details.
	// It will print in debug level.
	DebugMode bool
	// StructLog is the flag to enable/disable simple request&response struct log. It's only work when resty is init with zap logger.
	// When it's enabled, it will set debug mode to true. Struct log will print in info level.
	StructLog bool
}

var _defaultRestyParams = &RestyParams{
	Timeout:    5 * time.Second,
	RetryTimes: 0,
	DebugMode:  false,
	StructLog:  false,
}

func NewResty(options *RestyParams) *resty.Client {
	client := resty.New()
	if options == nil {
		return client
	}
	if options.Timeout != 0 {
		client.SetTimeout(options.Timeout)
	}
	if options.RetryTimes != 0 {
		client.SetRetryCount(options.RetryTimes)
	}
	if options.DebugMode {
		client.SetDebug(true)
	}
	return client
}

func DefaultResty() *resty.Client {
	return NewResty(_defaultRestyParams)
}

func NewRestyWithLogger(options *RestyParams, logger *zap.Logger) *resty.Client {
	client := NewResty(options)
	client.SetLogger(logger.With(zap.String("module", "resty")).Sugar())
	if options.StructLog {
		client.SetDebug(true)
		client.OnRequestLog(func(rl *resty.RequestLog) error {
			logger.Info("[Resty Http Request]", restyLogToZapFields(rl.Header, rl.Body)...)
			return nil
		})
		client.OnResponseLog(func(rl *resty.ResponseLog) error {
			logger.Info("[Resty Http Response]", restyLogToZapFields(rl.Header, rl.Body)...)
			return nil
		})
	}
	return client
}

func restyLogToZapFields(headers http.Header, body string) []zap.Field {
	var fields []zap.Field
	for k, v := range headers {
		fields = append(fields, zap.Strings("HEADER: "+k, v))
	}
	fields = append(fields, zap.String("BODY", body))
	return fields
}
