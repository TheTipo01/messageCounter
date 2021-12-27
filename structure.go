package main

// Server holds data about a guild
type Server struct {
	numberOfMessages int
}

type kv struct {
	Key   string
	Value int
}

// LightMessage is used to unmarshall only the parts of the messages that we are interested when reading from the DB
type LightMessage struct {
	Content string `json:"content"`
	Author  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"author"`
}
