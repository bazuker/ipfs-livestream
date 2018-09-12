# ipfs-livestream (ALPHA)

This program in combination with [IPFS](https://github.com/ipfs/go-ipfs)
provides a video streaming tool that allows to continuously push parts of the video stream to IPFS network.
Any stream is a pointer to a sync.json file that is essentially a collection of IPFS addresses pointing to the parts of the videostream.

This project is a raw implimentation of the idea and has much to work on.
The main concerns for now are:
1. The playback of the stream in the browser is a bit glitchy and hardly works over a long distance due to a current state of IPFS project itself.
2. The [web user interface](https://github.com/kisulken/ipfs-livestream/blob/master/watch.html) of the player is... not really good. Help needed
3. The code works but it's certainly not production ready yet
4. Requires a bunch of configuration before start

Do not hesitate to open an issue or make a pull request. Any help is appreciated
