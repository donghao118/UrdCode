package blocklogger

import (
	"encoding/json"
	"io"
	"time"

	"text/template"
)

type ConsensusEvent struct {
	Height                   int64
	Round                    int32
	RoundStep                string
	IsRoundStart, IsRoundEnd bool
	Message                  string
	Time                     time.Time
}

func NewConsensusEvent(height int64, round int32, roundStep string, IsStart, IsEnd bool, Message string) *ConsensusEvent {
	return &ConsensusEvent{
		Height:       height,
		Round:        round,
		RoundStep:    roundStep,
		IsRoundStart: IsStart, IsRoundEnd: IsEnd,
		Message: Message,
		Time:    time.Now(),
	}
}

const ConsensusEventTemplate = `[{{.Time}}](Height={{.Height}}, Round={{.Round}}) RoundStep={{.RoundStep}}   Event="{{.Message}}"
`

func (ce *ConsensusEvent) Write(writer io.Writer) error {
	t := template.Must(template.New("ConsensusEvent").Parse(ConsensusEventTemplate))
	if err := t.Execute(writer, ce); err != nil {
		return err
	}
	return nil
}
func (ce *ConsensusEvent) WriteJson(writer io.Writer) error {
	o, err := json.Marshal(ce)
	if err != nil {
		return err
	}
	o = append(o, '\n')
	if _, err := writer.Write(o); err != nil {
		return err
	}
	return nil
}
func NewConsensusEventFromJson(s []byte) (*ConsensusEvent, error) {
	var u ConsensusEvent
	err := json.Unmarshal(s, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
