/* sconfd.go
   Author: https://github.com/hanneslehmann
   Licence: free to use, no warranties!
*/
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	ADDRESS = "127.0.0.1:6379"
)

var (
	c, r              redis.Conn
	whatchlist        []string
	template_settings map[string]string
	config_settings   map[string]string
)

// Connect to Redis
func init() {
	var err error
	c, err = redis.Dial("tcp", ":6379")
	if err != nil {
		log.Fatalf("Couldn't connect to Redis: %v\n", err)
	}
	r, err = redis.Dial("tcp", ":6379")
	if err != nil {
		log.Fatalf("Couldn't connect to Redis: %v\n", err)
	}
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, ok := set[item]
	return ok
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func hget(c redis.Conn, query string) string {
	ret, err := c.Do(query)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return fmt.Sprint(ret)
}

func main() {
	defer c.Close()
	defer r.Close()
	prefix := "template:"
	//content := "content"
	config := "config"

	c.Do("SELECT", 1)
	r.Do("SELECT", 1)

	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) == 1 {
		config = fmt.Sprint("config:", os.Args[1], ":")
		fmt.Printf("Looking for changes of config: %s and ", os.Args[1])
		strs, err := redis.Strings(c.Do("KEYS", fmt.Sprint(config, "*")))
		whatchlist := []string{config}
		if err != nil {
			log.Fatal(err)
		}
		for _, element := range strs {
			arr := strings.Split(element, ":")
			if !contains(whatchlist, fmt.Sprint("template:", arr[2], ":")) {
				whatchlist = append(whatchlist, fmt.Sprint("template:", arr[2], ":"))
				fmt.Printf(" template:%s", arr[2])
			}
		}
		fmt.Printf(" \n")
		psc := redis.PubSubConn{Conn: r}
		psc.Subscribe("__keyevent@1__:hset")
		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
				params := strings.Split(string(n.Data), ":")
				if len(params) > 2 {
					ck := fmt.Sprint(params[0], ":", params[1], ":")
					if contains(whatchlist, ck) {
						key_index := 0
						if params[0] == "config" {
							key_index = 2
						}
						if params[0] == "template" {
							key_index = 1
						}
						if key_index > 0 {
							key_c := fmt.Sprint(config, params[key_index], ":content")
							key_m := fmt.Sprint(config, params[key_index], ":meta")
							key_tc := fmt.Sprint(prefix, params[key_index], ":content")
							key_tm := fmt.Sprint(prefix, params[key_index], ":meta")
							//sep := hget(c, fmt.Sprint("HGET",key_t, ":meta seperator"))
							com, err := redis.String(c.Do("HGET", key_tm, "comment"))
							if err != nil {
								fmt.Println(err)
								os.Exit(1)
							}
							sep, err := redis.String(c.Do("HGET", key_tm, "seperator"))
							if err != nil {
								fmt.Println(err)
								os.Exit(1)
							}
							fp, err := redis.String(c.Do("HGET", key_m, "filepath"))
							if err != nil {
								fmt.Println(err)
								os.Exit(1)
							}
							template_settings, err := redis.StringMap(c.Do("HGETALL", key_tc))
							if err != nil {
								fmt.Println(err)
								os.Exit(1)
							}
							config_settings, err := redis.StringMap(c.Do("HGETALL", key_c))
							if err != nil {
								fmt.Println(err)
								os.Exit(1)
							}
							for key, value := range config_settings {
								_, ok := template_settings[key]
								if ok {
									template_settings[key] = value
								} else {
									// fmt.Printf("key not found %s \n", key)
								}
							}
							f, err := os.Create(fp)
							check(err)
							defer f.Close()
							w := bufio.NewWriter(f)
							w.WriteString(fmt.Sprint(com, " ", time.Now().Format(time.RFC850), " Update from sconfd\n"))
							for key, value := range template_settings {
								w.WriteString(fmt.Sprint(key, " ", sep, " ", value, "\n"))
							}
							w.Flush()
							f.Close()
							fmt.Printf("File %s updated with changes from %s\n", fp, params[0])
						}
					}
				}
			case error:
				fmt.Printf("error: %v\n", n)
				return
			}
		}

	}

}
