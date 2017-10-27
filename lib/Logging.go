package lib

import (
    "log"
    "os"
)


const (
    INFO = iota
    WARNING
    ERROR
)

func InitLog(path string) map[int]*log.Logger {
    file, err := os.OpenFile(path, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil{
        panic(err)
    }
    logs := make(map[int]*log.Logger)

    logs[INFO] = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
    logs[WARNING] = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
    logs[ERROR] = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
    return logs
}
