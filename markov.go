package main

import (
	"github.com/bwmarrin/lit"
	"github.com/goccy/go-json"
	"github.com/mb-14/gomarkov"
	"strings"
	"sync"
)

// Returns a model for a given server
func buildModel(guildID string) *gomarkov.Chain {
	var (
		data []byte
		msg  *LightMessage
		wg   sync.WaitGroup
	)

	rows, err := db.Query("SELECT message FROM messages WHERE guildID = ?", guildID)
	if err != nil {
		lit.Error("Error querying db: %s", err.Error())
		return nil
	}

	chain := gomarkov.NewChain(1)

	for rows.Next() {
		err = rows.Scan(&data)
		if err != nil {
			continue
		}

		err = json.Unmarshal(data, &msg)
		if err != nil {
			continue
		}

		if !msg.Author.Bot {
			wg.Add(1)

			go func() {
				chain.Add(strings.Split(msg.Content, " "))
				wg.Done()
			}()
		}
	}

	wg.Wait()

	return chain
}

// saveModel updates the model on the database
func saveModel(guildID string) {
	if server[guildID] != nil {
		data, _ := json.Marshal(server[guildID].model)

		_, err := db.Exec("UPDATE servers SET model=? WHERE id=?", data, guildID)

		if err != nil {
			lit.Error("Error updating model: %s", err.Error())
		}
	} else {
		lit.Warn("Server map for guild %s is nil", guildID)
	}
}

// saveAllModels saves all the models in the map server
func saveAllModels() {
	for guildID := range server {
		saveModel(guildID)
	}
}

// loadModel loads the model from the db
func loadModel() {
	var (
		data    []byte
		guildID string
	)

	rows, _ := db.Query("SELECT model, id FROM servers")

	for rows.Next() {
		_ = rows.Scan(&data, &guildID)

		server[guildID] = &Server{numberOfMessages: 0, model: gomarkov.NewChain(1)}

		if len(data) == 0 {
			server[guildID].model = buildModel(guildID)
		} else {
			_ = json.Unmarshal(data, &server[guildID].model)
		}
	}
}
