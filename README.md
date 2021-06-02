# OnionMsg Daemon

[![DeepSource](https://deepsource.io/gh/Craumix/onionmsg.svg/?label=active+issues&show_trend=true)](https://deepsource.io/gh/Craumix/onionmsg/?ref=repository-badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/craumix/onionmsg)](https://goreportcard.com/report/github.com/craumix/onionmsg)
[![Tests](https://github.com/Craumix/onionmsg/actions/workflows/tests.yaml/badge.svg)](https://github.com/Craumix/onionmsg/actions/workflows/tests.yaml)

**⚠️ This programm is not considered stable atm, and will receive breaking changes. ⚠️**

OnionMsg is intended to be a P2P, anonymous and secure Messenger over the Tor Network.  
This is only a Daemon that should expose functionality over a TCP or UNIX Socket using a API to other **local** CLI or GUI clients.  
This Daemon is **not** intended to be used similarly to a E-Mail Server.

Features:
- P2P, no centralized servers needed
- Anonymous, no permanent identifiers, no logging
- Secure, all communication is signed and exclusively over Tor
