package lib

import (
    "log"
    "os"
)



func InitLog(path string) map[string]*log.Logger {
    file, err := os.OpenFile(path, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil{
        panic(err)
    }
    logs := make(map[string]*log.Logger)

    logs["info"] = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
    logs["warning"] = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
    logs["error"] = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
    return logs
}
