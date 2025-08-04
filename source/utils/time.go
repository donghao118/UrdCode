package utils

import (
	"emulator/proto/third_party"
	"time"
)

func ThirdPartyProtoTime(t time.Time) *third_party.Timestamp {
	return third_party.New(t)
}

func ThirdPartyUnmarshalTime(pt *third_party.Timestamp) time.Time {
	return pt.AsTime().Local().In(time.Local)
}
