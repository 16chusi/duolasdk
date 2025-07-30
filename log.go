package duolasdk

//日志能力
import (
	"fmt"
	"github.com/labstack/gommon/log"
	"os"
	"path"
	"strings"
)

type AppLog struct {
	*log.Logger
}

type LoggerOption struct {
	Type     string // "file" or "console"
	FileName string // 日志文件路径 if type is "file"
	Level    string
	Prefix   string
	Flag     int
}

func NewLogger(o *LoggerOption) *AppLog {
	al := &AppLog{}

	al.Logger = log.New(o.Prefix)

	if o.Type == "file" {
		dir := path.Dir(o.FileName)
		os.MkdirAll(dir, os.ModePerm)
		out, err := os.OpenFile(o.FileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)

		if err != nil {
			log.Fatal("无法创建日志文件:", err)
		}

		al.Logger.SetOutput(out)

	}

	level := log.INFO

	switch strings.ToLower(o.Level) {
	case "debug":
		level = log.DEBUG
	case "info":
		level = log.INFO
	case "warn":
		level = log.WARN
	case "error":
		level = log.ERROR
	case "off":
		level = log.OFF
	}
	al.Logger.SetHeader(fmt.Sprintf("[%s]- %s", o.Prefix, "${time_rfc3339} ${level} ${short_file}:${line} "))

	al.Logger.SetLevel(level)

	return al
}

type WailsLog struct {
	log *AppLog
}

func (w *WailsLog) Print(message string) {
	w.log.Print(message)
}

func (w *WailsLog) Trace(message string) {
	w.log.Print(message)
}

func (w *WailsLog) Debug(message string) {
	w.log.Debug(message)
}

func (w *WailsLog) Info(message string) {
	w.log.Info(message)
}

func (w *WailsLog) Warning(message string) {
	w.log.Warn(message)
}

func (w *WailsLog) Error(message string) {
	w.log.Error(message)
}

func (w *WailsLog) Fatal(message string) {
	w.log.Fatal(message)
}

func NewWailsLog(log *AppLog) *WailsLog {
	return &WailsLog{log}
}
