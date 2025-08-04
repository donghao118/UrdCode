package blocklogger

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"sync"

	tmos "github.com/tendermint/tendermint/libs/os"
)

type BlockWriter interface {
	OnStart() error
	// WriteBrief(*ConsensusEvent) error
	Write(*ConsensusEvent) error
	OnStop() error
}

type BlockLoggerWriter struct {
	mtx          sync.Mutex
	dir          string
	nodeName     string
	chain_id     string
	briefHandler *os.File
	handler      *os.File
	briefWriter  *bufio.Writer
	writer       *bufio.Writer
}

var _ BlockWriter = (*BlockLoggerWriter)(nil)

func NewBlockWriter(dir, nodeName, chain_id string) *BlockLoggerWriter {
	return &BlockLoggerWriter{
		mtx:      sync.Mutex{},
		dir:      dir,
		nodeName: nodeName,
		chain_id: chain_id,
	}
}
func (writer *BlockLoggerWriter) brief_path() string {
	return path.Join(writer.dir, writer.nodeName+"-blocklogger-brief.txt")
}
func (writer *BlockLoggerWriter) path() string {
	return path.Join(writer.dir, writer.nodeName+"-blocklogger.txt")
}
func (writer *BlockLoggerWriter) OnStart() error {
	brief_path := writer.brief_path()
	if err := tmos.EnsureDir(writer.dir, 0666); err != nil {
		return err
	}
	path := writer.path()

	if bf, err := os.OpenFile(brief_path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
		return err
	} else {
		writer.briefHandler = bf
		writer.briefWriter = bufio.NewWriter(bf)
	}

	if b, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
		return err
	} else {
		writer.handler = b
		writer.writer = bufio.NewWriter(b)
	}
	return nil
}
func (writer *BlockLoggerWriter) OnStop() error {
	err1 := writer.briefHandler.Close()
	err2 := writer.handler.Close()

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}
	return nil
}
func (writer *BlockLoggerWriter) Write(event *ConsensusEvent) error {
	writer.mtx.Lock()
	defer writer.mtx.Unlock()
	if event.IsRoundStart {
		if _, err := writer.writer.WriteString("\n\n"); err != nil {
			return err
		}
	}
	err1 := event.Write(writer.writer)
	err2 := event.WriteJson(writer.briefWriter)
	if err1 != nil || err2 != nil {
		if err1 == nil {
			return fmt.Errorf(fmt.Sprintf("nil | %v", err2))
		} else if err2 == nil {
			return fmt.Errorf(fmt.Sprintf("%v | nil", err1))
		}
		return fmt.Errorf(fmt.Sprintf("%v | %v", err1, err2))
	}
	err1 = writer.briefWriter.Flush()
	err2 = writer.writer.Flush()
	if err1 != nil || err2 != nil {
		if err1 == nil {
			return fmt.Errorf(fmt.Sprintf("nil | %v", err2))
		} else if err2 == nil {
			return fmt.Errorf(fmt.Sprintf("%v | nil", err1))
		}
		return fmt.Errorf(fmt.Sprintf("%v | %v", err1, err2))
	}
	return nil
}

// ===================================================================================

type NilBlockWriter struct{}

var _ BlockWriter = (*NilBlockWriter)(nil)

func NewNilBlockWriter() BlockWriter       { return &NilBlockWriter{} }
func (nbw *NilBlockWriter) OnStart() error { return nil }

// func (nbw *NilBlockWriter) WriteBrief(*ConsensusEvent) error { return nil }
func (nbw *NilBlockWriter) Write(*ConsensusEvent) error { return nil }
func (nbw *NilBlockWriter) OnStop() error               { return nil }
