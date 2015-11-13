package utility

import(
    "fmt"
    "log"
    "os"
    "runtime"
    "strconv"
    "strings"
    )

type DebugTrace struct {
    trace *log.Logger
    writeTrace bool
}

func DebugTraceFactory (enable string, header string) *DebugTrace {
    dt := new(DebugTrace)
    if enable != "" {
        dt.writeTrace = true
        traceHandle := os.Stdout
        dt.trace = log.New(traceHandle,
            header,
            log.Ldate|log.Ltime)
    }
    return dt
}

func (self *DebugTrace) Debug(prefix string, a ...interface{}) {
    if self.logging() {
        pc := make([]uintptr, 10)
        runtime.Callers(2,pc)
        funcName := runtime.FuncForPC(pc[0]).Name()
        dotIndex := strings.LastIndex(funcName, ".")
        shortFuncName := ""
        if dotIndex != -1 {
            shortFuncName = funcName[dotIndex+1:]
        } else {
            shortFuncName = funcName
        }
        _, file, line, _ := runtime.Caller(1)
        lineStr := strconv.Itoa(line)
        if len(lineStr) < 4 {
            lineStr = lineStr+strings.Repeat(" ",4-len(lineStr))
        }
        slashIndex := strings.LastIndex(file, "/")
        shortFile := ""
        if slashIndex != -1 {
            shortFile = file[slashIndex+1:]
        } else {
            shortFile = file
        }
        params := fmt.Sprintf("(%v)",a...)
        self.trace.Printf("%s:%v %v %v%v\n",shortFile,lineStr,prefix,shortFuncName,params)
    }
    return
}

func (self *DebugTrace) logging() bool {
    return self.writeTrace
}
