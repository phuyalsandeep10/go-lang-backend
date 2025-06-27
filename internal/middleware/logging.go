package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"homeinsight-properties/pkg/logger"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
)

// statusCodeToDescription maps HTTP status codes to their standard descriptions
func statusCodeToDescription(status int) string {
	switch status {
	case http.StatusOK:
		return fmt.Sprintf("%d OK", status)
	case http.StatusCreated:
		return fmt.Sprintf("%d Created", status)
	case http.StatusAccepted:
		return fmt.Sprintf("%d Accepted", status)
	case http.StatusNoContent:
		return fmt.Sprintf("%d No Content", status)
	case http.StatusMovedPermanently:
		return fmt.Sprintf("%d Moved Permanently", status)
	case http.StatusFound:
		return fmt.Sprintf("%d Found", status)
	case http.StatusBadRequest:
		return fmt.Sprintf("%d Bad Request", status)
	case http.StatusUnauthorized:
		return fmt.Sprintf("%d Unauthorized", status)
	case http.StatusForbidden:
		return fmt.Sprintf("%d Forbidden", status)
	case http.StatusNotFound:
		return fmt.Sprintf("%d Not Found", status)
	case http.StatusInternalServerError:
		return fmt.Sprintf("%d Internal Server Error", status)
	case http.StatusServiceUnavailable:
		return fmt.Sprintf("%d Service Unavailable", status)
	default:
		return fmt.Sprintf("%d Unknown", status)
	}
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		// Process request
		c.Next()

		// Ordered log fields
		logFields := make(map[string]interface{})
		orderedKeys := []string{
			"path",
			"method",
			"status",
			"data_source",
			"cache_hit",
			"latency",
			"query",
			"property_id",
			"timestamp",
			"client_ip",
		}

		// Core log fields
		logFields["path"] = path
		logFields["method"] = method
		status := c.Writer.Status()
		logFields["status"] = statusCodeToDescription(status)
		latencyMs := time.Since(start).Milliseconds()
		logFields["latency"] = fmt.Sprintf("%d ms", latencyMs)
		logFields["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		logFields["client_ip"] = clientIP

		// Conditionally add route-specific fields
		if ds, exists := c.Get("data_source"); exists && ds != "" {
			logFields["data_source"] = ds
		}
		if ch, exists := c.Get("cache_hit"); exists {
			logFields["cache_hit"] = ch
		}
		if q, exists := c.Get("query"); exists && q != "" {
			logFields["query"] = q
		}
		if pid, exists := c.Get("property_id"); exists && pid != "" {
			logFields["property_id"] = pid
		}

		// Marshal JSON with indentation
		logJSON, err := json.MarshalIndent(logFields, "", "  ")
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to marshal log: %v", err)
			return
		}

		// Convert JSON to string
		logString := string(logJSON)

		// Colorize cache_hit value
		if ch, exists := logFields["cache_hit"]; exists {
			if ch.(bool) {
				logString = strings.Replace(logString, `"cache_hit": true`, `"cache_hit": `+color.GreenString("true"), 1)
			} else {
				logString = strings.Replace(logString, `"cache_hit": false`, `"cache_hit": `+color.RedString("false"), 1)
			}
		}

		// Colorize status value
		if statusVal, exists := logFields["status"]; exists {
			statusStr := fmt.Sprintf(`"status": "%s"`, statusVal)
			var coloredStatus string
			if status >= 200 && status < 300 {
				coloredStatus = fmt.Sprintf(`"status": %s`, color.GreenString(`"%s"`, statusVal))
			} else if status >= 300 && status < 400 {
				coloredStatus = fmt.Sprintf(`"status": %s`, color.YellowString(`"%s"`, statusVal))
			} else {
				coloredStatus = fmt.Sprintf(`"status": %s`, color.RedString(`"%s"`, statusVal))
			}
			logString = strings.Replace(logString, statusStr, coloredStatus, 1)
		}

		// Colorize latency value
		if latency, exists := logFields["latency"]; exists {
			latencyStr := fmt.Sprintf(`"latency": "%s"`, latency)
			var coloredLatency string
			if latencyMs <= 10 {
				coloredLatency = fmt.Sprintf(`"latency": %s`, color.GreenString(`"%d ms"`, latencyMs)) // Very Good
			} else if latencyMs <= 50 {
				coloredLatency = fmt.Sprintf(`"latency": %s`, color.YellowString(`"%d ms"`, latencyMs)) // Good
			} else {
				coloredLatency = fmt.Sprintf(`"latency": %s`, color.RedString(`"%d ms"`, latencyMs)) // Bad
			}
			logString = strings.Replace(logString, latencyStr, coloredLatency, 1)
		}

		// Reorder fields
		var orderedFields []string
		lines := strings.Split(logString, "\n")
		for _, key := range orderedKeys {
			for _, line := range lines {
				trimmedLine := strings.TrimSpace(line)
				if strings.HasPrefix(trimmedLine, `"`+key+`":`) {
					orderedFields = append(orderedFields, line)
				}
			}
		}
		// Reconstruct JSON with ordered fields
		if len(orderedFields) > 0 {
			logString = "{\n  " + strings.Join(orderedFields, ",\n  ") + "\n}"
		} else {
			logString = "{\n}"
		}

		logger.GlobalLogger.Println(logString)
	}
}
