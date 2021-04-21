# TorMsg Daemon

[![CodeFactor](https://www.codefactor.io/repository/github/craumix/tormsg/badge)](https://www.codefactor.io/repository/github/craumix/tormsg)
[![DeepSource](https://deepsource.io/gh/Craumix/tormsg.svg/?label=active+issues&show_trend=true)](https://deepsource.io/gh/Craumix/tormsg/?ref=repository-badge)
[![Tests](https://github.com/Craumix/tormsg/actions/workflows/test.yaml/badge.svg)](https://github.com/Craumix/tormsg/actions/workflows/test.yaml)

**⚠️ This programm is not considered stable atm, and will receive breaking changes. ⚠️**

TorMsg is intended to be a P2P, anonymous and secure Messenger over the Tor Network.  
This is only a Daemon that should expose functionality over a TCP or UNIX Socket using a REST-API to other **local** CLI or GUI clients.  
This Daemon is **not** intended to be used similarly to a E-Mail Server.

Features:
- P2P, no centralized servers needed
- Anonymous, no permanent identifiers, no logging
- Secure, all communication is signed and exclusively over Tor

Potential Drawbacks:
- Slow/Long data transfers
- Requirement for Sender & Receiver to be online at the same time

<hr>

### Concept:
(Any "ID" described here is a pair of an Onion-Service-ID and another ed25519 Public-Key.  
Usually formatted as `serviceID + "@" + publicKey`)

Any user can generate an arbitrary amount of *ContactIDs* which are similar to usernames for regular messengers.  
Although they differ in some aspects:
- They can be created and deleted at any time.
- An arbitrary amount can be generated.
- Users who initiate contact don't need a ContactID.
- ContactIDs are unneeded except for negotiating a ChatRoom, so they are deleted as soon as they have been utilized.

*RoomIDs (or ConversationIDs)* are used for Messaging inside a ChatRoom. They are uniquely generated for every user for each ChatRoom.
