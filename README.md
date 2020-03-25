# Ten Hundred Bot
A [Discord](https://discordapp.com/) bot inspired by [xkcd's Simple Writer](https://xkcd.com/simplewriter/) and [Factorio's Discord](https://discord.gg/kvgdT24) channel #ten-hundred.
- Set a channel where only the 1000(ten hundred) most common words in English can be used.
- "Mutes" a user, restricting them to the 1000 most common words.

![Invite Ten Hundred Bot](https://i.imgur.com/4gF2uIe.png)

# Usage
`!th set` Only allows the 1000 most common words in the channel this is ran in. Supports only one channel.

`!th rem` Removes the restriction to the currently set channel.

`!th mute   (@User)`  Restricts a user to using only the 1000 most common words.

`!th unmute (@User)`  Unmutes a user.

`!th prefix (newPrefix)` Changes the prefix this bot responds to.

# Setup Locally
1. Download the latest release and run once to generate config.json.
2. [Create a bot](https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token) and add its Bot Token to config.json.
3. [Enable Developer Mode](https://support.discordapp.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) and add your user ID to admins in config.json.
4. Add your server ID to config.json.
5. Type `!th help` in any channel.
6. Type `!th mute @yourBestFriend` or `!th mute @yourself` in any channel.

