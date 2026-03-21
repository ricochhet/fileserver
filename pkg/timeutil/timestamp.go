package timeutil

import "time"

func TimeStamp() string {
	return time.Now().Format("2006-01-02_15-04-05")
}
