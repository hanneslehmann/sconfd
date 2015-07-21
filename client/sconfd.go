/* sconfd.go
   Author: https://github.com/hanneslehmann
   Licence: free to use, no warranties!
*/
package main

import (
	"bufio"
	"flag"
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
	redis_address     string
	c, r              redis.Conn
	whatchlist        []string
	template_settings map[string]string
	config_settings   map[string]string
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Usage of sconfd:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
}

// Connect to Redis
func init_redis(url string) redis.Conn {
	var err error
	var c redis.Conn
	c, err = redis.Dial("tcp", url)
	if err != nil {
		log.Fatalf("Couldn't connect to Redis: %v\n", err)
	}
	return c
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

func main() {
	prefix := "template:"
	config := "config"

	var clientID string
	flag.StringVar(&clientID, "id", "", "the unique Client ID for identification (mandatory)")
	flag.StringVar(&redis_address, "a", ADDRESS, "the URL with format IP:PORT for connecting database (optional)")
	flag.Parse()

	c := init_redis(redis_address)
	defer c.Close()
	r := init_redis(redis_address)
	defer r.Close()

	c.Do("SELECT", 1)
	r.Do("SELECT", 1)

	if len(clientID) > 1 {
		fmt.Printf("Connected to: tcp://%s \n", redis_address)
		config = fmt.Sprint("config:", clientID, ":")
		fmt.Printf("Looking for changes of config:%s and ", clientID)
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
						switch params[0] {
						case "template":
							key_index = 1
						case "config":
							key_index = 2
						}
						if key_index > 0 {
							key_c := fmt.Sprint(config, params[key_index], ":content")
							key_m := fmt.Sprint(config, params[key_index], ":meta")
							key_tc := fmt.Sprint(prefix, params[key_index], ":content")
							key_tm := fmt.Sprint(prefix, params[key_index], ":meta")
							com, err := redis.String(c.Do("HGET", key_tm, "comment"))
							check(err)
							sep, err := redis.String(c.Do("HGET", key_tm, "seperator"))
							check(err)
							fp, err := redis.String(c.Do("HGET", key_m, "filepath"))
							check(err)
							template_settings, err := redis.StringMap(c.Do("HGETALL", key_tc))
							check(err)
							config_settings, err := redis.StringMap(c.Do("HGETALL", key_c))
							check(err)
							for key, value := range config_settings {
								_, ok := template_settings[key]
								if ok {
									template_settings[key] = value
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

	} else {
		Usage()
	}

}
