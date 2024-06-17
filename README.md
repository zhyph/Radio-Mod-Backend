# Radio-Mod-Backend
Files for hosting your own [Mordhau Radio Mod](https://mod.io/g/mordhau/m/radio-mod) backend!

## Prerequisites

- In order to use the download fallback method (which is necessary for downloading age-restricted YouTube videos), [YT-DLP](https://github.com/yt-dlp/yt-dlp) needs to be installed on your backend server.
- To avoid getting your IP temporarily blocked by YouTube, I highly recommend adding a list of proxy servers via the config.json file. For reference, my Radio Mod backend server uses a list of 100 IPv6 proxies.

## Installation

1. In your game server's game.ini file, add the line ```Endpoint="http://myBackendServerIPorDomainName:myBackendServerPort"``` under the ```[RadioMod]``` section
2. Download the correct precompiled binary for your server's operating system from [here](https://github.com/TheSaltySeaCow/Radio-Mod-Backend/releases/latest)
3. Make sure the file is executable (if on Linux, ```chmod +x radio-linux-x```)
4. Run the file once for it to generate the config.json file
5. Edit config.json to include your server details
6. Run the file again! (I recommend running it from a process manager such as [PM2](https://pm2.keymetrics.io/docs/usage/quick-start/))
