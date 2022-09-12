# messageCounter
[![Go Report Card](https://goreportcard.com/badge/github.com/TheTipo01/messageCounter)](https://goreportcard.com/report/github.com/TheTipo01/messageCounter)

messageCounter - a discord bot to track messages and ghostpings

## Features

- Show ghostpings on a separate webpage
- Sends random message every monday from a certain channel in a certain guild
- Send a nice as the message number 69420

For the random message part, you need to add a row to the config table with the server ID (guildID), the channel from where to get messages (channelID), and where to send the random message (channelToID). You can also add a offset for the message count part of the bot, as sometimes the numbers that the bot get and the one that discord returns are different.

## Install
Get a release for your system in the [release](https://github.com/TheTipo01/messageCounter/releases) tab, modify the provided `example_config.yml`, deploy the website on your favourite webserver, and you're good to go!
