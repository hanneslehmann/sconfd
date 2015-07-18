package main

import (
   "github.com/vaughan0/go-ini"
   "fmt"
   "os"
   "log"
   "github.com/garyburd/redigo/redis"
)

const (
    ADDRESS = "127.0.0.1:6379"
)

var (
    conn redis.Conn
)

// Connect to Redis
func init() {
    var err error
    conn, err = redis.Dial("tcp", ":6379")
    if err != nil {
        log.Fatalf("Couldn't connect to Redis: %v\n", err)
    }
}


func main() {
    defer conn.Close()
    prefix := "template"
    content := "content"

    conn.Do("SELECT", 1)

    argsWithoutProg := os.Args[1:]

    if len(argsWithoutProg) == 1 {
      println("Parsing ", os.Args[1])
      file, err := ini.LoadFile( os.Args[1])
      if err != nil {
      }
      conn.Do("HMSET", fmt.Sprint(prefix,":", os.Args[1],":meta"), "seperator", "=")

      for name, _ := range file {
         fmt.Printf("Section name: %s\n", name)
         for key, value := range file[name] {
            fmt.Printf("%s => %s\n", key, value)
            if len(name)>1 {
              _,err := conn.Do("HMSET", fmt.Sprint(prefix,":", os.Args[1],":",content,":",name), key, value)
              if err != nil {
                log.Fatalf("Error setting hash: %v\n", err)
              }

            } else {
              _,err := conn.Do("HMSET", fmt.Sprint(prefix,":", os.Args[1],":",content), key, value)
              if err != nil {
                log.Fatalf("Error setting hash: %v\n", err)
              }

            }
         }
      }
    }

}
