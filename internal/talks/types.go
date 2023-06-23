package talks

// Talk is a struct that represents a talk.
type Talk struct {
	ID    int    `json:"id"`
	UUID  string `json:"uuid"`
	Title string `json:"title"`
}
