# Akichat CORE

This packages contains all the functionalities to interact
with hentakihabara chat.

## Install

```bash
go get github.com/CasvalDOT/akichat-core
```

## How to use

Include in your code like:

```go
package main

import (
    core "github.com/CasvalDOT/akichat-core"
)

func main() {
    chat := core.NewChat("hentakihabara")

    // Login
    err := chat.Login("username", "password")

    // Read messages
    msgs,err := chat.ReadMessages("0")

    // Write message
    err = chat.WriteMessage("message")

    // Write private message
    err = chat.WritePrivateMessage("username", "message")

    // Logout
    err = chat.Logout()

}
```
