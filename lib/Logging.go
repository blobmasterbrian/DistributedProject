package lib

import (
    "log"
    "os"
)

// Constants to access each logger from log map
const (
    INFO = iota
    WARNING
    ERROR
)

// Function to create log map with association between logger and constant
func InitLog(path string) map[int]*log.Logger {
    file, err := os.OpenFile(path, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)  // creates log file with proper permissions
    if err != nil{
        panic(err)
    }
    logs := make(map[int]*log.Logger)  // creates map

    logs[INFO] = log.New(file, "INFO:       ", log.Ldate|log.Ltime|log.Lshortfile)        // creates info logger association in map
    logs[WARNING] = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)  // creates warning logger association in map
    logs[ERROR] = log.New(file, "ERROR:     ", log.Ldate|log.Ltime|log.Lshortfile)      // creates error logger association in map
    return logs
}
