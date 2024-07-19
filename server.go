package main

import (
    "fmt"
    "log"
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
    conn *websocket.Conn
    send chan []byte
}

var clients = make(map[*Client]bool)
var broadcast = make(chan []byte)
var mutex = &sync.Mutex{}

func handleConnections(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := &Client{conn: conn, send: make(chan []byte)}
    mutex.Lock()
    clients[client] = true
    mutex.Unlock()

    go handleClientMessages(client)

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Printf("Error reading message: %v", err)
            mutex.Lock()
            delete(clients, client)
            mutex.Unlock()
            break
        }
        broadcast <- msg
    }
}

func handleClientMessages(client *Client) {
    for {
        msg := <-client.send
        err := client.conn.WriteMessage(websocket.TextMessage, msg)
        if err != nil {
            log.Printf("Error writing message: %v", err)
            client.conn.Close()
            mutex.Lock()
            delete(clients, client)
            mutex.Unlock()
            break
        }
    }
}

func handleMessages() {
    for {
        msg := <-broadcast
        mutex.Lock()
        for client := range clients {
            select {
            case client.send <- msg:
            default:
                close(client.send)
                delete(clients, client)
            }
        }
        mutex.Unlock()
    }
}

func main() {
    fmt.Println("Starting WebSocket server on port 8080...")
    http.HandleFunc("/ws", handleConnections)
    go handleMessages()
    log.Fatal(http.ListenAndServe(":8080", nil))
}
