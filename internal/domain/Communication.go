package domain

type StatusAsk struct {
	NeedLink       bool              `json:"needLink,omitempty"`
	NeedStatistics bool              `json:"needStatistics,omitempty"`
	BindUsers      map[string]string `json:"bindUsers"`
}

type StatisticsInfo struct {
	Finished    []string `json:"finished,omitempty"`
	Studying    []string `json:"studying,omitempty"`
	Expired     []string `json:"expired,omitempty"`
	Waiting     []string `json:"waiting,omitempty"`
	NotFinished []string `json:"notFinished,omitempty"`
}

type LinkInfo struct {
	Link string `json:"link"`
}

type FinishInfo struct {
	Nick  string `json:"nick,omitempty"`
	Score int    `json:"score,omitempty"`
}

type NotifyInfo struct {
	Nick    string `json:"nick,omitempty"`
	Success bool   `json:"success"`
}

type SendToDingUser struct {
	UserId   string `json:"userId,omitempty"`
	MsgKey   string `json:"msgKey,omitempty"`
	MsgParam string `json:"msgParam,omitempty"`
}
