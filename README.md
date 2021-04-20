# TorMsg Daemon

[![CodeFactor](https://www.codefactor.io/repository/github/craumix/tormsg/badge)](https://www.codefactor.io/repository/github/craumix/tormsg)
[![DeepSource](https://deepsource.io/gh/Craumix/tormsg.svg/?label=active+issues&show_trend=true)](https://deepsource.io/gh/Craumix/tormsg/?ref=repository-badge)

TorMsg is intended to be a P2P, anonymous and secure Messenger over the Tor Network.  
This is only a Daemon that should expose functionality over a TCP or UNIX Socket using a REST-API to other **local** CLI or GUI clients.  
This Daemon is **not** intended to be used similarly to a E-Mail Server.

Features:
- P2P, no centralized servers needed
- Anonymous, no permanent identifiers, no logging
- Secure, all communication is signed and exclusively over Tor

Potential Drawbacks:
- Slow / Long data transfers
- Requirement for Sender & Receiver to be online at the same time