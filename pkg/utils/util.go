package utils

import ecpb "github.com/tommenx/pvproto/pkg/proto/executorpb"

func ToCount(num int64, unit ecpb.Unit) int64 {
	var KB, MB, GB int64
	KB = 1 << 10
	MB = KB << 10
	GB = MB << 10
	if unit == ecpb.Unit_B {
		return num
	} else if unit == ecpb.Unit_KB {
		return num * KB
	} else if unit == ecpb.Unit_MB {
		return num * MB
	} else if unit == ecpb.Unit_GB {
		return num * GB
	}
	return 0
}
