package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Init(isDev bool) error {
	// Создаём директорию для логов, если её нет
	err := os.MkdirAll("/app/logs", 0755)
	if err != nil {
		return err
	}

	// Открываем/создаем файл логов
	file, err := os.OpenFile("/app/logs/app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	// Настраиваем core для логирования в файл
	fileWriter := zapcore.AddSync(file)
	consoleWriter := zapcore.AddSync(os.Stdout)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), fileWriter, zapcore.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), consoleWriter, zapcore.DebugLevel),
	)

	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}

// Получить глобальный логгер
func L() *zap.Logger {
	if log == nil {
		panic("Logger не инициализирован. Нужен logger.Init() в main.go")
	}
	return log
}
