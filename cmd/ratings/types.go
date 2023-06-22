package ratings

type Rating struct {
	TalkUuid string `json:"talk_uuid"`
	Value    int64  `json:"value"`
}
