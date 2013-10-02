# CodeCube

Runnable code snippets


## Instructions

1. Get a server with Docker installed
2. Install golang v1.1 (note: this is not the default Ubuntu version)
3. Copy this project to the server
4. Install nginx and install the config file in conf/nginx.conf (it may need
   some minor changes)
5. Go in the server/ directory, and run `docker build -t runner .` to create
   a tagged docker image that's ready for running code
6. Set up quotas ([instructions here][1]), and `apt-get install quota`
7. Build a pool of uids with quotas attached (`./set-quotas.sh`)
8. Compile the go app with `go build .`
9. Run the go app with a process manager (look at conf/upstart.conf for
   inspiration)

[1]: https://www.digitalocean.com/community/articles/how-to-enable-user-quotas


## Future Plans

- Collaborative editing using share.js or ot.js
- Improvements to the go server to allow sending output and config changes
  (e.g. switching language) to multiple clients, probably using SSE


