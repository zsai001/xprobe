package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// 最大日志文件大小 (MB)
	maxSize = 10
	// 保留的旧日志文件数量
	maxBackups = 5
	// 保留的日志天数
	maxAge = 30
)

var (
	// 全局logger实例
	logger *zap.SugaredLogger
	// 确保初始化只执行一次
	once sync.Once
)

// getLogPath 根据不同操作系统返回日志文件路径
func getLogPath() string {
	var basePath string

	//switch runtime.GOOS {
	//case "windows":
	//	// Windows: %ProgramData%\XProbe\logs
	//	programData := os.Getenv("ProgramData")
	//	if programData == "" {
	//		programData = filepath.Join(os.Getenv("SystemDrive")+"\\", "ProgramData")
	//	}
	//	basePath = filepath.Join(programData, "XProbe", "logs")
	//case "darwin":
	//	// macOS: /Library/Logs/XProbe
	//	basePath = "/Library/Logs/XProbe"
	//default:
	//	// Linux: /var/log/xprobe
	//	basePath = "/var/log/xprobe"
	//}
	basePath = "./"

	// 确保日志目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
	}

	return filepath.Join(basePath, "xprobe.log")
}

// Init 初始化日志系统
func Init() {
	once.Do(func() {
		// 创建 lumberjack logger 实例
		logWriter := &lumberjack.Logger{
			Filename:   getLogPath(),
			MaxSize:    maxSize,    // MB
			MaxBackups: maxBackups, // 保留的旧文件个数
			MaxAge:     maxAge,     // 保留的天数
			Compress:   true,       // 压缩旧文件
			LocalTime:  true,       // 使用本地时间
		}

		// 创建多个输出
		writers := []zapcore.WriteSyncer{
			zapcore.AddSync(logWriter),
		}

		// 在开发环境下同时输出到控制台
		if os.Getenv("XPROBE_ENV") != "production" {
			writers = append(writers, zapcore.AddSync(os.Stdout))
		}

		// 定义日志格式
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     customTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// 创建核心
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(writers...),
			zap.NewAtomicLevelAt(zap.InfoLevel),
		)

		// 创建logger
		zapLogger := zap.New(core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zap.ErrorLevel),
		)

		// 转换为 SugaredLogger
		logger = zapLogger.Sugar()
	})
}

// customTimeEncoder 自定义时间编码器
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// Debug 输出调试级别日志
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Debugf 输出格式化的调试级别日志
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Info 输出信息级别日志
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Infof 输出格式化的信息级别日志
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Warn 输出警告级别日志
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Warnf 输出格式化的警告级别日志
func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

// Error 输出错误级别日志
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Errorf 输出格式化的错误级别日志
func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Fatalf 输出格式化的致命错误日志并退出程序
func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}

// WithFields 返回带有字段的日志记录器
func WithFields(fields map[string]interface{}) *zap.SugaredLogger {
	return logger.With(fieldsToArgs(fields)...)
}

// fieldsToArgs 将字段映射转换为参数列表
func fieldsToArgs(fields map[string]interface{}) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

// Sync 同步缓冲区到磁盘
func Sync() error {
	return logger.Sync()
}
