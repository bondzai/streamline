package entities

type Event struct {
	Id           string  `json:"-"`
	LoginSession *string `json:"loginSession"`
}