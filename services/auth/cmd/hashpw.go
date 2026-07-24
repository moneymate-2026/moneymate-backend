//go:build ignore

package main

import (
    "fmt"
    "os"

    "github.com/alexedwards/argon2id"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "usage: go run hashpw.go <password>\n")
        os.Exit(1)
    }
    hash, err := argon2id.CreateHash(os.Args[1], argon2id.DefaultParams)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(hash)
}