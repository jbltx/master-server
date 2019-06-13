package main

func main() {
	s := NewMasterServer()
	s.Listen("0.0.0.0:8080")
}
