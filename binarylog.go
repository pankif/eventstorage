package binarylog

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
)

func New(basePath string, errWriter io.Writer) (*binaryLogger, error) {
	b := &binaryLogger{
		basePath:       basePath,
		buf:            new(bytes.Buffer),
		encodeBuf:      make([]byte, 3),
		logFilesMap:    make(logFilesMap),
		logFilesCount:  0,
		logFileMaxSize: 100 * MB,
		errWriter:      errWriter,
	}

	if err := b.initRegistryFile(); err != nil {
		return nil, err
	}

	if err := b.initLogFile(); err != nil {
		return nil, err
	}

	b.logFileSize = b.calculateLogFileSize()

	lastLine := b.logFileSize / lineLength
	lineBuffer := make([]byte, lineLength)
	_, _ = b.logFile.ReadAt(lineBuffer, lastLine*lineLength)

	rawLine := strings.NewReplacer(" ", "", "\n", "").Replace(string(lineBuffer))
	res, _ := hex.DecodeString(rawLine)
	b.lastLineBytesCount = len(res)

	return b, nil
}

func (b *binaryLogger) insertData(data []byte) int64 {
	var dataLen int64 = 0

	for i := range data {
		var l int64 = 2

		hex.Encode(b.encodeBuf, data[i:i+1])
		b.lastLineBytesCount++

		if b.lastLineBytesCount >= 16 {
			b.encodeBuf[2] = '\n'
			b.lastLineBytesCount = 0
			l++
		} else if b.lastLineBytesCount%2 == 0 {
			b.encodeBuf[2] = ' '
			l++
		}

		dataLen += l
		b.buf.Write(b.encodeBuf[:l])
	}

	return dataLen
}

func (b *binaryLogger) Log(data []byte) (writtenLen int64, err error) {
	b.locker.Lock()
	defer b.locker.Unlock()

	writtenLen = b.insertData(data)
	writtenLen += b.insertData(RowDelimiter)
	b.logFileSize += writtenLen
	b.insertsCount++

	if b.autoFlushCount > 0 && b.insertsCount >= b.autoFlushCount {
		if _, err = b.flush(); err != nil {
			return
		}
	}

	if b.logFileSize >= b.logFileMaxSize {
		if _, err = b.flush(); err != nil {
			return
		}

		if err = b.rotateLogFile(); err != nil {
			return
		}
	}

	return
}

func (b *binaryLogger) flush() (count int, err error) {
	// todo - err for check nil log file

	if b.insertsCount > 0 {
		if _, err := b.logFile.Write(b.buf.Bytes()); err != nil {
			return 0, errors.New("flush failed: " + err.Error())
		} else {
			b.buf.Truncate(0)
			count = b.insertsCount
			b.insertsCount = 0
		}
	}

	return
}

func (b *binaryLogger) Flush() (count int, err error) {
	b.locker.Lock()
	defer b.locker.Unlock()
	return b.flush()
}

func (b *binaryLogger) SetAutoFlushCount(count int) {
	b.autoFlushCount = count
}

func (b *binaryLogger) GetAutoFlushCount() int {
	return b.autoFlushCount
}

func (b *binaryLogger) SetAutoFlushTime(period time.Duration) error {
	if b.autoFlushTime != 0 {
		return ErrAutoFlushTimeAlreadySet
	}

	if period <= 0 {
		return ErrAutoFlushTimeTooLow
	}

	b.autoFlushTime = period

	go func() {
		for range time.Tick(period) {
			// todo - support Shutdown function to exit from gorutine
			if _, err := b.Flush(); err != nil {
				b.logErrorString("time flush failed: " + err.Error())
			}
		}
	}()

	return nil
}

func (b *binaryLogger) Read(offset int64, count int64, whence int) ([]byte, error) {
	buffer := make([]byte, count)

	if err := b.ReadTo(&buffer, offset, whence); err != nil {
		return []byte{}, err
	}

	return buffer, nil
}

func (b *binaryLogger) ReadTo(buffer *[]byte, offset int64, whence int) error {
	_, err := b.logFile.Seek(offset, whence)

	if err != nil {
		return err
	}

	if _, err = b.logFile.Read(*buffer); err != nil {
		return err
	}

	return nil
}

func (b *binaryLogger) Decode(data []byte) ([]byte, error) {
	pure := make([]byte, 0, len(data))

	for _, v := range data {
		if v != Space && v != LineBreak && v != EmptyByte {
			pure = append(pure, v)
		}
	}

	dist := make([]byte, hex.DecodedLen(len(pure)))

	if _, err := hex.Decode(dist, pure); err != nil {
		return dist, err
	}

	return dist, nil
}
