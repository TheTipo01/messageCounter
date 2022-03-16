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
	data, _ := json.Marshal(server[guildID].model)

	_, err := db.Exec("UPDATE servers SET model=? WHERE id=?", data, guildID)

	if err != nil {
		lit.Error("Error updating model: %s", err.Error())
	}
}

// loadModel loads the model from the db or builds it if it doesn't exist
func loadModel(guildID string) *gomarkov.Chain {
	var (
		data  []byte
		chain gomarkov.Chain
	)

	_ = db.QueryRow("SELECT model FROM servers WHERE id=?", guildID).Scan(&data)

	if len(data) == 0 {
		return nil
	} else {
		_ = json.Unmarshal(data, &chain)
		return &chain
	}
}
