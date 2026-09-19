package tracer

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Scope identifies the subsystem responsible for an operational event.
// It is intentionally small and stable so the console remains easy to scan.
type Scope string

type FailureOrigin string

const (
	ScopeSystem   Scope = "SYSTEM"
	ScopeHTTP     Scope = "HTTP"
	ScopeAuth     Scope = "AUTH"
	ScopeDatabase Scope = "DATABASE"
	ScopeArtifact Scope = "ARTIFACT"
	ScopeQueue    Scope = "QUEUE"
	ScopeWorker   Scope = "WORKER"
	ScopeMatch    Scope = "MATCH"
	ScopeAgent    Scope = "AGENT"
	ScopeReplay   Scope = "REPLAY"
)

const (
	OriginAgent          FailureOrigin = "agent"
	OriginGame           FailureOrigin = "game"
	OriginPlatform       FailureOrigin = "platform"
	OriginInfrastructure FailureOrigin = "infrastructure"
)

// Config controls presentation only. Event meaning and fields stay identical
// between the human console and JSON production output.
type Config struct {
	Level  string
	Format string
	Color  string
}

type Field struct {
	key        string
	value      any
	detailOnly bool
}

func String(key, value string) Field  { return Field{key: key, value: value} }
func Int(key string, value int) Field { return Field{key: key, value: value} }
func Duration(key string, value time.Duration) Field {
	return Field{key: key, value: value}
}

func Origin(value FailureOrigin) Field { return String("origin", string(value)) }

// detail keeps diagnostic data in JSON output and in a DEBUG console, while
// leaving the default human console concise.
func detail(key string, value any) Field {
	return Field{key: key, value: value, detailOnly: true}
}

func Err(err error) Field {
	if err == nil {
		return detail("error", "")
	}
	return detail("error", redactSensitive(err.Error()))
}

var (
	emailPattern       = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	jwtPattern         = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)
	credentialPattern  = regexp.MustCompile(`(?i)(postgres(?:ql)?://[^:\s]+:)[^@\s]+@`)
	unixPathPattern    = regexp.MustCompile(`(?:\.{0,2}/)[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)+`)
	windowsPathPattern = regexp.MustCompile(`\b[A-Za-z]:\\(?:[^\\\s]+\\)*[^\\\s]+`)
)

func redactSensitive(value string) string {
	value = emailPattern.ReplaceAllString(value, "[redacted-email]")
	value = jwtPattern.ReplaceAllString(value, "[redacted-token]")
	value = credentialPattern.ReplaceAllString(value, `${1}[redacted]@`)
	value = unixPathPattern.ReplaceAllString(value, "[redacted-path]")
	value = windowsPathPattern.ReplaceAllString(value, "[redacted-path]")
	return value
}

type contextKey string

const (
	requestIDKey    contextKey = "request_id"
	actorIDKey      contextKey = "actor_id"
	jobIDKey        contextKey = "job_id"
	matchIDKey      contextKey = "match_id"
	attemptKey      contextKey = "attempt"
	requestTraceKey contextKey = "request_trace"
)

type requestTrace struct {
	scope   Scope
	event   string
	message string
	fields  []Field
	failed  bool
}

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(nonNilContext(ctx), requestIDKey, id)
}

func BeginRequest(ctx context.Context, id string) context.Context {
	ctx = withRequestID(ctx, id)
	return context.WithValue(ctx, requestTraceKey, &requestTrace{})
}

// FailRequest records the terminal cause without printing it immediately.
// The HTTP middleware emits one final line for the entire request.
func FailRequest(ctx context.Context, scope Scope, event, message string, fields ...Field) {
	if !hasField(fields, "origin") {
		origin := OriginPlatform
		switch scope {
		case ScopeDatabase, ScopeArtifact, ScopeQueue, ScopeWorker, ScopeReplay:
			origin = OriginInfrastructure
		}
		fields = append(fields, Origin(origin))
	}
	trace, ok := nonNilContext(ctx).Value(requestTraceKey).(*requestTrace)
	if !ok || trace == nil {
		ErrorEvent(ctx, scope, event, message, fields...)
		return
	}
	trace.scope = scope
	trace.event = event
	trace.message = message
	trace.fields = append([]Field(nil), fields...)
	trace.failed = true
}

func hasField(fields []Field, key string) bool {
	for _, field := range fields {
		if field.key == key {
			return true
		}
	}
	return false
}

func CompleteRequest(ctx context.Context, method, path string, status int, elapsed time.Duration) {
	fields := []Field{
		String("method", method),
		String("path", path),
		Int("status", status),
		Duration("elapsed", elapsed),
	}
	if trace, ok := nonNilContext(ctx).Value(requestTraceKey).(*requestTrace); ok && trace != nil && trace.failed {
		fields = append(fields, trace.fields...)
		ErrorEvent(ctx, trace.scope, trace.event, trace.message, fields...)
		return
	}
	if status >= 500 {
		fields = append(fields, Origin(OriginPlatform))
		ErrorEvent(ctx, ScopeHTTP, "http.request.failed", "La solicitud no pudo completarse", fields...)
		return
	}
	if status >= 400 || method == "GET" || method == "HEAD" || method == "OPTIONS" {
		DebugEvent(ctx, ScopeHTTP, "http.request.completed", "Solicitud completada", fields...)
		return
	}
	InfoEvent(ctx, ScopeHTTP, "http.request.completed", "Solicitud completada", fields...)
}

func WithActorID(ctx context.Context, id string) context.Context {
	return context.WithValue(nonNilContext(ctx), actorIDKey, id)
}

func WithJobID(ctx context.Context, id string) context.Context {
	return context.WithValue(nonNilContext(ctx), jobIDKey, id)
}

func WithMatchID(ctx context.Context, id string) context.Context {
	return context.WithValue(nonNilContext(ctx), matchIDKey, id)
}

func WithAttempt(ctx context.Context, attempt int) context.Context {
	return context.WithValue(nonNilContext(ctx), attemptKey, attempt)
}

type loggerState struct {
	logger *zap.Logger
	level  zapcore.Level
	format string
}

var global struct {
	sync.RWMutex
	state *loggerState
}

// Configure replaces the process logger. Invalid values fall back to safe
// defaults and are reported once after the logger is available.
func Configure(config Config) {
	state, warnings := buildState(config, zapcore.Lock(os.Stderr))

	global.Lock()
	previous := global.state
	global.state = state
	global.Unlock()

	if previous != nil {
		_ = previous.logger.Sync()
	}
	for _, warning := range warnings {
		logEvent(context.Background(), zapcore.WarnLevel, ScopeSystem, "logging.config.invalid", warning)
	}
}

func Sync() error {
	return currentState().logger.Sync()
}

func DebugEvent(ctx context.Context, scope Scope, event, message string, fields ...Field) {
	logEvent(ctx, zapcore.DebugLevel, scope, event, message, fields...)
}

func InfoEvent(ctx context.Context, scope Scope, event, message string, fields ...Field) {
	logEvent(ctx, zapcore.InfoLevel, scope, event, message, fields...)
}

func WarnEvent(ctx context.Context, scope Scope, event, message string, fields ...Field) {
	logEvent(ctx, zapcore.WarnLevel, scope, event, message, fields...)
}

func ErrorEvent(ctx context.Context, scope Scope, event, message string, fields ...Field) {
	logEvent(ctx, zapcore.ErrorLevel, scope, event, message, fields...)
}

func FatalEvent(ctx context.Context, scope Scope, event, message string, fields ...Field) {
	logEvent(ctx, zapcore.ErrorLevel, scope, event, message, fields...)
	_ = Sync()
	os.Exit(1)
}

func logEvent(ctx context.Context, level zapcore.Level, scope Scope, event, message string, fields ...Field) {
	state := currentState()
	if !state.logger.Core().Enabled(level) {
		return
	}

	allFields := append(contextFields(ctx), fields...)
	allFields = sanitizeFields(allFields)
	if state.format == "json" {
		zapFields := make([]zap.Field, 0, len(allFields)+2)
		zapFields = append(zapFields, zap.String("scope", strings.ToLower(string(scope))))
		zapFields = append(zapFields, zap.String("event", event))
		for _, field := range allFields {
			if field.key == "" {
				continue
			}
			zapFields = append(zapFields, zap.Any(field.key, field.value))
		}
		state.logger.Log(level, sanitizeMessage(message), zapFields...)
		return
	}

	visible := make([]Field, 0, len(allFields))
	for _, field := range allFields {
		if field.key == "" || (field.detailOnly && state.level > zapcore.DebugLevel) {
			continue
		}
		visible = append(visible, field)
	}
	formatted := fmt.Sprintf("%-10s %-38s", scope, sanitizeMessage(message))
	if suffix := renderFields(visible); suffix != "" {
		formatted += "  " + suffix
	}
	state.logger.Log(level, strings.TrimRight(formatted, " "))
}

var protectedFieldKeys = map[string]struct{}{
	"authorization": {},
	"code":          {},
	"email":         {},
	"password":      {},
	"payload":       {},
	"sql":           {},
	"token":         {},
	"username":      {},
}

func sanitizeFields(fields []Field) []Field {
	sanitized := make([]Field, 0, len(fields))
	for _, field := range fields {
		copy := field
		if _, protected := protectedFieldKeys[strings.ToLower(copy.key)]; protected {
			copy.value = "[redacted]"
			sanitized = append(sanitized, copy)
			continue
		}
		if value, ok := copy.value.(string); ok {
			if copy.key != "path" || !strings.HasPrefix(value, "/api/") {
				copy.value = redactSensitive(value)
			}
		}
		sanitized = append(sanitized, copy)
	}
	return sanitized
}

func currentState() *loggerState {
	global.RLock()
	state := global.state
	global.RUnlock()
	if state != nil {
		return state
	}

	global.Lock()
	defer global.Unlock()
	if global.state == nil {
		format := os.Getenv("LOG_FORMAT")
		if format == "" {
			mode := strings.ToLower(os.Getenv("MODE"))
			if mode == "gcp" || mode == "railway" {
				format = "json"
			} else {
				format = "console"
			}
		}
		global.state, _ = buildState(Config{
			Level:  os.Getenv("LOG_LEVEL"),
			Format: format,
			Color:  os.Getenv("LOG_COLOR"),
		}, zapcore.Lock(os.Stderr))
	}
	return global.state
}

func buildState(config Config, sink zapcore.WriteSyncer) (*loggerState, []string) {
	var warnings []string
	level, err := zapcore.ParseLevel(strings.ToLower(config.Level))
	if config.Level == "" {
		level = zapcore.InfoLevel
	} else if err != nil {
		level = zapcore.InfoLevel
		warnings = append(warnings, "LOG_LEVEL no reconocido; se usará info")
	}

	format := strings.ToLower(config.Format)
	if format == "" {
		format = "console"
	}
	if format != "console" && format != "json" {
		format = "console"
		warnings = append(warnings, "LOG_FORMAT no reconocido; se usará console")
	}

	colorMode := strings.ToLower(config.Color)
	if colorMode == "" {
		colorMode = "auto"
	}
	colorEnabled := false
	switch colorMode {
	case "always":
		colorEnabled = format == "console"
	case "never":
		colorEnabled = false
	case "auto":
		colorEnabled = format == "console" && isTerminal(os.Stderr) && os.Getenv("NO_COLOR") == ""
	default:
		warnings = append(warnings, "LOG_COLOR no reconocido; se usará auto")
		colorEnabled = format == "console" && isTerminal(os.Stderr) && os.Getenv("NO_COLOR") == ""
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey: "message",
		LevelKey:   "level",
		TimeKey:    "timestamp",
		LineEnding: zapcore.DefaultLineEnding,
		EncodeTime: zapcore.TimeEncoderOfLayout("15:04:05"),
	}
	var encoder zapcore.Encoder
	if format == "json" {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = consoleLevelEncoder(colorEnabled)
		if colorEnabled {
			encoderConfig.EncodeTime = func(value time.Time, encoder zapcore.PrimitiveArrayEncoder) {
				encoder.AppendString("\x1b[90m" + value.Format("15:04:05") + "\x1b[0m")
			}
		}
		encoderConfig.ConsoleSeparator = "  "
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, sink, level)
	return &loggerState{logger: zap.New(core), level: level, format: format}, warnings
}

func consoleLevelEncoder(color bool) zapcore.LevelEncoder {
	return func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
		label := fmt.Sprintf("%-5s", strings.ToUpper(level.String()))
		if !color {
			encoder.AppendString(label)
			return
		}
		code := "\x1b[90m"
		switch level {
		case zapcore.InfoLevel:
			code = "\x1b[34m"
		case zapcore.WarnLevel:
			code = "\x1b[33m"
		case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
			code = "\x1b[31m"
		}
		encoder.AppendString(code + label + "\x1b[0m")
	}
}

func contextFields(ctx context.Context) []Field {
	ctx = nonNilContext(ctx)
	fields := make([]Field, 0, 5)
	appendString := func(key contextKey) {
		if value, ok := ctx.Value(key).(string); ok && value != "" {
			fields = append(fields, String(string(key), value))
		}
	}
	appendString(requestIDKey)
	appendString(actorIDKey)
	appendString(jobIDKey)
	appendString(matchIDKey)
	if value, ok := ctx.Value(attemptKey).(int); ok && value > 0 {
		fields = append(fields, Int(string(attemptKey), value))
	}
	return fields
}

func renderFields(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}
	sort.SliceStable(fields, func(i, j int) bool {
		return fieldPriority(fields[i].key) < fieldPriority(fields[j].key)
	})
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.key+"="+renderValue(field.key, field.value))
	}
	return strings.Join(parts, " ")
}

func fieldPriority(key string) int {
	switch key {
	case "method":
		return 10
	case "path":
		return 20
	case "status":
		return 30
	case "elapsed":
		return 40
	case "actor_id", "job_id", "match_id", "attempt":
		return 80
	case "request_id":
		return 90
	default:
		return 50
	}
}

func renderValue(key string, value any) string {
	if duration, ok := value.(time.Duration); ok {
		return duration.Round(time.Millisecond).String()
	}
	text := sanitizeOneLine(fmt.Sprint(value))
	if strings.HasSuffix(key, "_id") {
		text = shortID(text)
	}
	if text == "" {
		return `""`
	}
	if strings.IndexFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || r == '=' || r == '"'
	}) >= 0 {
		return strconv.Quote(text)
	}
	return text
}

func shortID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func sanitizeMessage(message string) string {
	return sanitizeOneLine(redactSensitive(message))
}

func sanitizeOneLine(message string) string {
	message = strings.ReplaceAll(message, "\r", `\r`)
	message = strings.ReplaceAll(message, "\n", `\n`)
	message = strings.TrimSpace(message)
	if len(message) > 240 {
		message = message[:237] + "..."
	}
	return message
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

type OperationalMetrics struct {
	LeaseRenewals            atomic.Int64
	LeaseLosses              atomic.Int64
	SandboxSpawns            atomic.Int64
	SandboxDisqualifications atomic.Int64
	SandboxTimeouts          atomic.Int64
}

var GlobalMetrics OperationalMetrics

func RecordLeaseRenewal() {
	GlobalMetrics.LeaseRenewals.Add(1)
}

func RecordLeaseLoss() {
	GlobalMetrics.LeaseLosses.Add(1)
}

func RecordSandboxSpawn() {
	GlobalMetrics.SandboxSpawns.Add(1)
}

func RecordSandboxDisqualification() {
	GlobalMetrics.SandboxDisqualifications.Add(1)
}

func RecordSandboxTimeout() {
	GlobalMetrics.SandboxTimeouts.Add(1)
}

type MetricsSnapshot struct {
	LeaseRenewals            int64 `json:"lease_renewals"`
	LeaseLosses              int64 `json:"lease_losses"`
	SandboxSpawns            int64 `json:"sandbox_spawns"`
	SandboxDisqualifications int64 `json:"sandbox_disqualifications"`
	SandboxTimeouts          int64 `json:"sandbox_timeouts"`
}

func GetMetricsSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		LeaseRenewals:            GlobalMetrics.LeaseRenewals.Load(),
		LeaseLosses:              GlobalMetrics.LeaseLosses.Load(),
		SandboxSpawns:            GlobalMetrics.SandboxSpawns.Load(),
		SandboxDisqualifications: GlobalMetrics.SandboxDisqualifications.Load(),
		SandboxTimeouts:          GlobalMetrics.SandboxTimeouts.Load(),
	}
}
