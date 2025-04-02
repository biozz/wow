# jot

A small tool for jotting down notes from Telegram.

I wanted something like Telegram's "saved messages", but to be able to
forward the messages to my notes, which I manage in [Obsidian](https://obsidian.md/).

The idea is:

- there is a Telegram bot
- you send message to a bot and it saves it to `inbox` folder
- the bot is running in a Docker container
- there is also a [Syncthing](https://syncthing.net/) instance, also in a Docker container
- they are both pointed to the same volume (folder)
- the utility saves the messages in a certain format with a front-matter
- if you edit the message, it is updated in the folder as well
