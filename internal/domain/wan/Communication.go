package wan

type StatusAsk struct {
	NeedLink       bool `json:"needLink,omitempty"`
	NeedStatistics bool `json:"needStatistics,omitempty"`
}

type StatisticsInfo struct {
	Finished []string `json:"finished,omitempty"`
	Studying []string `json:"studying,omitempty"`
}

type LinkInfo struct {
	Link string `json:"link"`
}

type FinishInfo struct {
	Nick  string `json:"nick,omitempty"`
	Score int    `json:"score,omitempty"`
}

type ExpiredInfo struct {
	Nick string `json:"nick,omitempty"`
}
