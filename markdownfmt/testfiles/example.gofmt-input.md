Valid but incorrectly formatted code:
```go
f, err := os.Open(something)
    if err != nil {
    // handle..
}
defer f.Close() // What if an error occurs here?

// Write something to file... etc.
```

Invalid code:

```go
// Non parsable go code should be handled, but will be not go fmt-ed.
f, err := os.Open(...)
    if err != nil {
    // handle..
}
defer f.Close() // What if an error occurs here?

// Write something to file... etc.
```
