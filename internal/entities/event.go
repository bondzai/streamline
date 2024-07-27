package entities

type Event struct {
	Id      string  `json:"id"`
	Message *string `json:"message"`
}
