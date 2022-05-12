package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

var Logger *zap.Logger

var SugaredLogger *zap.SugaredLogger

func IniLog(logPath string, logLevel int) {
	//文件输出配置
	var fileCfg = zapcore.EncoderConfig{
		MessageKey:   "msg",                       //结构化（json）输出：msg的key
		LevelKey:     "level",                     //结构化（json）输出：日志级别的key（INFO，WARN，ERROR等）
		TimeKey:      "ts",                        //结构化（json）输出：时间的key（INFO，WARN，ERROR等）
		CallerKey:    "file",                      //结构化（json）输出：打印日志的文件对应的Key
		EncodeLevel:  zapcore.CapitalLevelEncoder, //将日志级别转换成大写（INFO，WARN，ERROR等）
		EncodeCaller: zapcore.ShortCallerEncoder,  //采用短文件路径编码输出（test/main.go:14	）
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("15:04:05"))
		}, //输出的时间格式
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		}, //输出持续时间
	}

	// 终端输出配置
	var stdCfg = zapcore.EncoderConfig{
		TimeKey:        "ts",
		CallerKey:      "file",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	//自定义日志级别：自定义Info级别
	var infoLevel = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel && lvl >= zapcore.Level(logLevel)
	})

	//自定义日志级别：自定义Warn级别
	/*warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel && lvl >= zapcore.Level(logLevel)
	})*/

	// 实现多个输出
	var core = zapcore.NewTee(
		//将info及以下写入logPath，NewConsoleEncoder 是非结构化输出
		zapcore.NewCore(zapcore.NewJSONEncoder(fileCfg), zapcore.AddSync(NewLogWriter(logPath)), infoLevel),
		//warn及以下写入errPath
		//zapcore.NewCore(zapcore.NewJSONEncoder(config), zapcore.AddSync(warnWriter), warnLevel),
		//同时将日志输出到控制台，NewJSONEncoder 是结构化输出
		zapcore.NewCore(zapcore.NewJSONEncoder(stdCfg), zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), zapcore.Level(logLevel)),
	)

	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel))
	defer Logger.Sync()

	SugaredLogger = Logger.Sugar()
}

type LogWrite struct {
	path     string
	currName string
	file     *os.File
}

func NewLogWriter(path string) *LogWrite {
	if _, err := os.Stat(path); err != nil {
		_ = os.MkdirAll(path, 0777)
	}

	var log = &LogWrite{
		path: path,
	}

	log.Cut2new()

	return log
}

func (l *LogWrite) Cut2new() {
	var nowStr = time.Now().Local().Format("2006-01-02")
	if nowStr == l.currName {
		return
	}

	var fileName = fmt.Sprintf("%s/%s.log", l.path, nowStr)
	var f, _ = os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	_ = l.file.Close()
	l.file = f
	l.currName = nowStr
}

func (l *LogWrite) Write(b []byte) (int, error) {
	l.Cut2new()

	return l.file.Write(b)
}

// Recover 异常捕获处理
func Recover() {
	var err = recover()
	if err == nil {
		return
	}

	switch err.(type) {
	case runtime.Error:
		Logger.Error("runtime.err", zap.Any("err", err))
	default:
		Logger.Error("App Panic Stop", zap.Any("err", err))
	}

	Logger.Error("Recover", zap.Binary("err", debug.Stack()))
}
