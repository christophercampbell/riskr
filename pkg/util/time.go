package util

import "time"

// NowFn allows test override of time.Now.
var NowFn = time.Now
