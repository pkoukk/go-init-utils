package giu

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func NewGinWithLogger(zl *zap.Logger) *gin.Engine {
	e := gin.New()
	e.Use(NewGinMiddlewareTrace(), NewGinMiddlewareJsonLogger(zl), NewGinMiddlewareRecovery(zl))
	return e
}

var GIN_TRACE_ID = "X-Trace-Id"

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

// bodyLogWriter is a wrapper around ResponseWriter that allows us to read the response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// NewGinMiddlewareJsonLogger returns a gin middleware for logging json request and response.
func NewGinMiddlewareJsonLogger(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// before request
		if filterFlags(c.ContentType()) == gin.MIMEJSON {
			data, _ := c.GetRawData()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(data))
			l.Info("[gin request]",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String(GIN_TRACE_ID, c.GetHeader(GIN_TRACE_ID)),
				zap.Any("body", json.RawMessage(data)))
		}

		bw := bodyLogWriter{body: bytes.NewBuffer([]byte{}), ResponseWriter: c.Writer}
		c.Writer = bw
		c.Next()

		// after request
		if filterFlags(c.Writer.Header().Get("Content-Type")) == gin.MIMEJSON {
			l.Info("[gin response]",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String(GIN_TRACE_ID, c.GetHeader(GIN_TRACE_ID)),
				zap.Any("body", json.RawMessage(bw.body.Bytes())))
		}
	}
}

// NewGinMiddlewareTrace returns a gin middleware for adding trace id to request header.
func NewGinMiddlewareTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(GIN_TRACE_ID)
		if traceID == "" {
			traceID = uuid.New().String()
			c.Header(GIN_TRACE_ID, traceID)
		}
		c.Next()
	}
}

type zapWriter struct {
	zl *zap.Logger
}

func (zw *zapWriter) Write(p []byte) (n int, err error) {
	zw.zl.Panic(string(p))
	return len(p), nil
}

func writerFromZapLogger(l *zap.Logger) *zapWriter {
	return &zapWriter{l}
}

// NewGinMiddlewareRecovery returns a gin middleware for recovery with zap logger.
func NewGinMiddlewareRecovery(zl *zap.Logger) gin.HandlerFunc {
	zl = zl.With(zap.String("module", "gin"), zap.String("type", "recovery"))
	return gin.RecoveryWithWriter(writerFromZapLogger(zl))
}
