package logger

import (
	"io"
	"log"
)

var (
	InfoLogger  *log.Logger
	WarnLogger  *log.Logger
	ErrorLogger *log.Logger
)

const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

func InitLogs(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) {
	//to be used for INFO-level logging: InfoLogger.Println("foo is now bar")
	InfoLogger = log.New(infoHandle, "INFO  - ", logPattern)
	//to be used for WARN-level logging: warnLogger.Println("foo is now bar")
	WarnLogger = log.New(warnHandle, "WARN  - ", logPattern)
	//to be used for ERROR-level logging: errorLogger.Println("foo is now bar")
	ErrorLogger = log.New(errorHandle, "ERROR - ", logPattern)
}
