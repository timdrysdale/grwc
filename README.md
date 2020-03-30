# grwc
![alt text][status]

GRWC gob reconnecting websocket for service relay

## Features
- specify topic to relay 
- set connection exclusivity (unique or all)
- gob/de-gobbing of ```srgob``` messages

## What happened?

Using ```grwc``` and ```crossbar``` together to relay ```ssh``` would have forced clients to be aware of the stream multiplexing, which wasn't all that pretty - it meant all clients received all streams from that service, for a start. That wasn't necessarily a security risk because of encryption (no worse than any other relay), but no one wants a mess under the carpet like sending data that isn't going to be used. A better idea came along - that being [shellbar](https://github.com/timdrysdale/shellbar), which yields a cleaner and more efficient design because it can work with ```sa``` to do the multiplexing of multiple clients streams as an issue handled solely between the server and the relay, leaving clients none-the-wiser and less burdened with spam.

[status]: https://img.shields.io/badge/dead-avoid-red "Dead status; avoid"