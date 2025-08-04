package blocklogger

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BlockLoggerReader struct {
	mtx sync.Mutex

	path   string
	events []*ConsensusEvent
}

func NewReader(p string) *BlockLoggerReader {
	var b BlockLoggerReader
	b.path = p
	f, err := os.Open(b.path)
	if err != nil {
		panic(err)
	}
	h := bufio.NewReader(f)
	defer f.Close()
	for {
		line, _, err := h.ReadLine()
		if err != nil {
			break
		}
		event, err := NewConsensusEventFromJson(line)
		if err != nil {
			panic(err)
		}
		if event.IsRoundEnd && event.IsRoundStart {
			event.IsRoundStart = false
		}
		b.events = append(b.events, event)
	}
	return &b
}

func (b *BlockLoggerReader) CalculateTPS(i, j int) (float64, float64, error) {
	var start, end time.Time
	var count, commit, inner int
	if i == 1 && j == i {
		return 0.0, 0.0, nil
	}
	if i == 1 {
		i = 2
	}

	for _, event := range b.events {
		if event.IsRoundStart && event.Height == int64(i) {
			start = event.Time
		}
		if event.IsRoundEnd && event.Height == int64(j) {
			end = event.Time
		}
		if event.IsRoundEnd && strings.HasPrefix(event.Message, BLOCK_FINISH_HEIGHT) && event.Height >= int64(i) && event.Height <= int64(j) {
			finishDataStr := event.Message[len(BLOCK_FINISH_HEIGHT)+1 : len(event.Message)-1]
			uStr := strings.Split(finishDataStr, ",")
			if u, err := strconv.Atoi(uStr[0]); err == nil {
				inner += u
			} else {
				panic(err)
			}
			if u, err := strconv.Atoi(uStr[1]); err == nil {
				commit += u
			} else {
				panic(err)
			}
			if u, err := strconv.Atoi(uStr[2]); err == nil {
				count += u
			} else {
				panic(err)
			}
		}
	}
	times := float64(end.Sub(start)/time.Microsecond) / 1000.0 / 1000.0
	return float64(commit+inner) / times, float64(commit) / float64(count) * 100.0, nil
}

func (b *BlockLoggerReader) NoneZeroPeriods() ([]int, []int, error) {
	var starts, ends = []int{}, []int{}
	stateMachine := false
	for _, event := range b.events {
		if event.IsRoundEnd && strings.HasPrefix(event.Message, BLOCK_FINISH_HEIGHT) {
			finishDataStr := event.Message[len(BLOCK_FINISH_HEIGHT)+1 : len(event.Message)-1]
			uStr := strings.Split(finishDataStr, ",")
			var total = 0
			if u, err := strconv.Atoi(uStr[0]); err == nil {
				total += u
			} else {
				panic(err)
			}
			if u, err := strconv.Atoi(uStr[2]); err == nil {
				total += u
			} else {
				panic(err)
			}
			if stateMachine && total == 0 {
				ends = append(ends, int(event.Height)-1)
				stateMachine = false
			} else if !stateMachine && total != 0 {
				starts = append(starts, int(event.Height))
				stateMachine = true
			}
		}
	}
	if stateMachine {
		ends = append(ends, int(b.events[len(b.events)-1].Height)-1)
	}
	return starts, ends, nil
}
