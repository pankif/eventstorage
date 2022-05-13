package main

import (
	"encoding/hex"
	"fmt"
	"github.com/pankif/binarylog"
	"os"
)

func main() {
	fmt.Println(os.TempDir())
	binlog, _ := binarylog.New("./", os.Stderr)
	binlog.SetAutoFlushCount(1)
	binlog.SetLogFileMaxSize(binarylog.KB)
	defer func() {
		_ = binlog.CloseLogFile()
	}()

	// binlog.Log([]byte([]byte{0, 0, 0}))
	binlog.Log([]byte("its binlog row "))
	return

	data, err := binlog.Read(0, 99, 0)
	fmt.Println(err)
	decoded, _ := binlog.Decode(data)
	fmt.Println(string(decoded))
}

func interest() {
	g, _ := hex.DecodeString("1") // 67 in HEX is 'g' char, 6 or 7 (or some wrong symbol) decode from hex return zero length result
	fmt.Println(string(g))
	fmt.Println(len(g))
	fmt.Println(len(string(g)))
}
