run:
	go run main.go

unpack4:
	for i in {1..4}; do curl 127.0.0.1:8090/unpack; done

unpack20:
	for i in {1..20}; do curl 127.0.0.1:8090/unpack; done

report:
	go tool pprof -http=:8080 "localhost:6060/debug/pprof/heap"
