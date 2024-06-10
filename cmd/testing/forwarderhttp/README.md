# Forwarder HTTP test utility

This utility exercises the Forwarder lambda code for the case where the backend is an HTTP server.

At a bare minimum, you must provide a list of input files via the command line args:

```
go run main.go ./my-test.json ./another-file.csv
```

The output of the program will be a directory containing all request bodies
sent to the HTTP server. Each output filename will be named according to the
hash of contents. This is useful for verifying code changes do not affect how
data is chunked. 

When processing files, the code will apply the same presets that the Forwarder
lambda does. You may have to use `content-type` or `content-encoding` flags to
fake either object attribute, since neither is a property of the local
filesystem. For example, to test an AWS Config file, you would have to set `content-encoding`:

```
go run main.go \
    -content-encoding=gzip \
    ./123456789012_Config_us-west-2_ConfigHistory_AWS::SSM::ManagedInstanceInventory_20240607T130841Z_20240607T190342Z_1.json.gz
```

In the lambda case, `content-encoding` is already set in S3. In the local
testing case, we must configure it manually.

## Profiling

To dump a profile of the executed code, set `-profile`:

```
go run main.go \
    -profile=mem \
    ...
```

You can then explore the file through `go tool`, e.g:
```
go tool pprof -http=:8080 forwarder-post/mem.pprof
```
