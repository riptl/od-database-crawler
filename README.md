# od-database Go crawler 🚀
[![Build Status](https://travis-ci.org/terorie/od-database-crawler.svg?branch=master)](https://travis-ci.org/terorie/od-database-crawler)
> by terorie 2018 :P

 * Crawler for [__OD-Database__](https://github.com/simon987/od-database)
 * Crawls HTTP open directories (standard Web Server Listings)
 * Gets name, path, size and modification time of all files
 * Lightweight and fast: __over 9000 requests per second__ on a standard laptop

https://od-db.the-eye.eu/

#### Usage

 1. With Config File (if `config.yml` found in working dir)
    - Download [default config](https://github.com/terorie/od-database-crawler/blob/master/config.yml)
    - Set `server.url` and `server.token`
    - Start with `./od-database-crawler server --config <file>`
 
 2. With Flags or env
    - Override config file if it exists
    - `--help` for list of flags
    - Every flag is available as an environment variable:
      `--server.crawl_stats` ➡️ `OD_SERVER_CRAWL_STATS`
    - Start with `./od-database-crawler server <flags>`
