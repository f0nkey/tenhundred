# Ten Hundred Bot
A [Discord](https://discordapp.com/) bot inspired by [xkcd's Simple Writer](https://xkcd.com/simplewriter/) and [Factorio's Discord](https://discord.gg/kvgdT24) channel #ten-hundred.
- Set a Discord channel where only the 1000(ten hundred) most common words in English can be used.
- "Mutes" a user, restricting them to the 1000 most common words.

![Invite Ten Hundred Bot](https://i.imgur.com/4gF2uIe.png)

# Usage
`!t set` Only allows the 1000 most common words in the channel this is ran in.

`!t rem` Removes the restriction to the currently set channel.

`!t mute   (@User)`  Restricts a user to using only the 1000 most common words.

`!t unmute (@User)`  Unmutes a user.

`!t prefix (newPrefix)` Changes the prefix this bot responds to.

# Setup Locally
1. Download the latest release and run once to generate config.json.
2. [Create a bot](https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token) and add its Bot Token to config.json.
3. [Enable Developer Mode](https://support.discordapp.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) and add your user ID to admins in config.json.
4. Add your server ID to config.json.
5. Type `!t help` in any channel.
6. Type `!t mute @yourBestFriend` or `!tmute @yourself` in any channel.

