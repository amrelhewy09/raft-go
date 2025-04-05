# To run a cluster of 3 nodes ports 8080, 8081, 8082 in 3 different terminals

## Node 1
```ID=1 ADDR=localhost:8080 PEERS=localhost:8081,localhost:8082 go run main.go```
## Node 2
```ID=2 ADDR=localhost:8081 PEERS=localhost:8080,localhost:8082 go run main.go ```
## Node 3
```ID=3 ADDR=localhost:8082 PEERS=localhost:8081,localhost:8080 go run main.go```
